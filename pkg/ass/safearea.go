package ass

import (
	"strconv"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

// Platform safe areas on a 1080 x 1920 vertical video, from each platform's
// own guidance:
//   - TikTok: the In-Feed Standard LTR safe zone template on TikTok's In-Feed
//     ads page, measured from the image. The right edge widens below y=840,
//     where the like, comment and share buttons sit.
//   - YouTube Shorts: Google's Universal Video Ad Safe Zones, vertical video.
//   - Instagram Reels: Meta's Reels ad guide, about 14% top, 35% bottom and
//     6% on each side.
const (
	referenceWidth  = 1080.0
	referenceHeight = 1920.0

	tiktokTop        = 240 / referenceHeight
	tiktokBottom     = 660 / referenceHeight
	tiktokSide       = 120 / referenceWidth
	tiktokLowerRight = 300 / referenceWidth
	tiktokLowerFrom  = 840 / referenceHeight

	shortsTop    = 288 / referenceHeight
	shortsBottom = 672 / referenceHeight
	shortsLeft   = 48 / referenceWidth
	shortsRight  = 192 / referenceWidth

	reelsTop    = 0.14
	reelsBottom = 0.35
	reelsSide   = 0.06

	// SafeAreaAll is the name of the area that clears every listed platform.
	SafeAreaAll = "all"
)

// Edges named in a SafeAreaIssue.
const (
	EdgeTop    = "top"
	EdgeBottom = "bottom"
	EdgeLeft   = "left"
	EdgeRight  = "right"
)

// SafeArea is the part of a vertical video a platform's buttons and captions
// leave uncovered, as fractions of the script width and height. Text that
// reaches below LowerFromRatio of the height needs LowerRightRatio on the
// right instead of RightRatio.
type SafeArea struct {
	Name            string
	TopRatio        float64
	BottomRatio     float64
	LeftRatio       float64
	RightRatio      float64
	LowerRightRatio float64
	LowerFromRatio  float64
}

// SafeAreaIssue is a style or event whose placement reaches into the unsafe
// part of the video. Event is the one-based event number, or zero for a style.
type SafeAreaIssue struct {
	Style    string
	Event    int
	Edge     string
	Margin   int
	Required int
}

// SafeAreas returns the platform safe areas and the combined area that clears all of them.
func SafeAreas() []SafeArea {
	return []SafeArea{
		{
			Name: SafeAreaAll, TopRatio: shortsTop, BottomRatio: shortsBottom,
			LeftRatio: tiktokSide, RightRatio: shortsRight,
			LowerRightRatio: tiktokLowerRight, LowerFromRatio: tiktokLowerFrom,
		},
		{
			Name: "tiktok", TopRatio: tiktokTop, BottomRatio: tiktokBottom,
			LeftRatio: tiktokSide, RightRatio: tiktokSide,
			LowerRightRatio: tiktokLowerRight, LowerFromRatio: tiktokLowerFrom,
		},
		{
			Name: "shorts", TopRatio: shortsTop, BottomRatio: shortsBottom,
			LeftRatio: shortsLeft, RightRatio: shortsRight,
			LowerRightRatio: shortsRight, LowerFromRatio: 1,
		},
		{
			Name: "reels", TopRatio: reelsTop, BottomRatio: reelsBottom,
			LeftRatio: reelsSide, RightRatio: reelsSide,
			LowerRightRatio: reelsSide, LowerFromRatio: 1,
		},
	}
}

// LookupSafeArea returns a named safe area, or an error wrapping commerr.ErrNotFound.
func LookupSafeArea(name string) (SafeArea, error) {
	for _, area := range SafeAreas() {
		if area.Name == name {
			return area, nil
		}
	}

	return SafeArea{}, ctxerrors.Wrapf(commerr.ErrNotFound, "safe area %q", name)
}

type placement struct {
	alignment Alignment
	marginL   int
	marginR   int
	marginV   int
}

// CheckSafeArea reports every style, and every event with its own margins,
// whose margins let text reach into the area's unsafe edges. It reads margins
// and alignment only; inline \pos and \move tags are not checked.
func (d *Document) CheckSafeArea(area SafeArea) ([]SafeAreaIssue, error) {
	width, height, err := d.resolution()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "resolve script canvas")
	}

	styles := make(map[string]placement, len(d.styles))
	var issues []SafeAreaIssue
	for _, row := range d.styles {
		name := strings.TrimSpace(row.field(columnName))
		style, err := rowPlacement(row)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "read style %q", name)
		}
		styles[name] = style
		issues = append(issues, area.check(style, width, height, name, 0)...)
	}
	for index, row := range d.events {
		name := strings.TrimSpace(row.field(columnStyle))
		event, err := rowPlacement(row)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "read event %d", index+1)
		}
		if event.marginL == 0 && event.marginR == 0 && event.marginV == 0 {
			continue
		}
		style := styles[name]
		// A zero event margin keeps the style's margin for that side.
		for _, side := range []struct{ event, style *int }{
			{&event.marginL, &style.marginL}, {&event.marginR, &style.marginR}, {&event.marginV, &style.marginV},
		} {
			if *side.event == 0 {
				*side.event = *side.style
			}
		}
		event.alignment = style.alignment
		issues = append(issues, area.check(event, width, height, name, index+1)...)
	}

	return issues, nil
}

func rowPlacement(row record) (placement, error) {
	var result placement
	for _, field := range []struct {
		column string
		target *int
	}{
		{columnMarginL, &result.marginL}, {columnMarginR, &result.marginR}, {columnMarginV, &result.marginV},
	} {
		if err := parseField(row, field.column, field.target); err != nil {
			return placement{}, ctxerrors.Wrap(err, "read margin")
		}
	}
	if row.fieldIndex(columnAlignment) >= 0 {
		var alignment int
		if err := parseField(row, columnAlignment, &alignment); err != nil {
			return placement{}, ctxerrors.Wrap(err, "read alignment")
		}
		result.alignment = Alignment(alignment)
	}

	return result, nil
}

func parseField(row record, column string, target *int) error {
	value := strings.TrimSpace(row.field(column))
	if value == "" {
		return nil
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return ctxerrors.Wrapf(commerr.ErrInvalidValue, "%s %q is not a whole number", column, value)
	}
	*target = number

	return nil
}

func (a SafeArea) check(p placement, width, height int, style string, event int) []SafeAreaIssue {
	var issues []SafeAreaIssue
	add := func(edge string, margin, need int) {
		if margin < need {
			issues = append(issues, SafeAreaIssue{Style: style, Event: event, Edge: edge, Margin: margin, Required: need})
		}
	}

	// The anchor row decides which vertical margin matters and where the text's
	// anchor line sits. Text reaches the lower area when that line is below it.
	lowerFrom := a.LowerFromRatio * float64(height)
	var anchor float64
	switch {
	case p.alignment >= AlignmentTopLeft:
		add(EdgeTop, p.marginV, scaled(height, a.TopRatio))
		anchor = float64(p.marginV)
	case p.alignment >= AlignmentMiddleLeft:
		anchor = float64(height) / 2
	default:
		add(EdgeBottom, p.marginV, scaled(height, a.BottomRatio))
		anchor = float64(height - p.marginV)
	}
	reachesLower := anchor > lowerFrom

	right := a.RightRatio
	if reachesLower {
		right = a.LowerRightRatio
	}
	add(EdgeLeft, p.marginL, scaled(width, a.LeftRatio))
	add(EdgeRight, p.marginR, scaled(width, right))

	return issues
}
