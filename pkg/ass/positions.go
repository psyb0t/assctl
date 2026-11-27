package ass

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

const (
	defaultScriptWidth  = 384
	defaultScriptHeight = 288
	maxScriptDimension  = math.MaxInt32
	legacyWideWidth     = 1280
	legacyWideHeight    = 1024
)

// TextPosition sets style alignment and margins as fractions of the script canvas.
// Existing event margins and inline positioning retain precedence.
type TextPosition struct {
	Name             string
	Alignment        Alignment
	MarginRatio      float64
	LeftMarginRatio  float64
	RightMarginRatio float64
}

// TextPositions returns independent values in display order. Portrait names describe intended use.
func TextPositions() []TextPosition {
	return []TextPosition{
		{
			Name: "portrait-upper", Alignment: AlignmentTopCenter, MarginRatio: socialUpperMarginRatio,
			LeftMarginRatio: socialSideMarginRatio, RightMarginRatio: socialSideMarginRatio,
		},
		{
			Name: "portrait-center", Alignment: AlignmentMiddleCenter,
			LeftMarginRatio: socialSideMarginRatio, RightMarginRatio: socialSideMarginRatio,
		},
		{
			Name: "portrait-lower-middle", Alignment: AlignmentBottomCenter, MarginRatio: socialLowerMarginRatio,
			LeftMarginRatio: socialSideMarginRatio, RightMarginRatio: socialSideMarginRatio,
		},
	}
}

// LookupTextPosition returns a named placement, or an error wrapping commerr.ErrNotFound.
func LookupTextPosition(name string) (TextPosition, error) {
	for _, position := range TextPositions() {
		if position.Name == name {
			return position, nil
		}
	}
	return TextPosition{}, ctxerrors.Wrapf(commerr.ErrNotFound, "text position %q", name)
}

func (p TextPosition) style(style Style, width, height int) (Style, error) {
	if width <= 0 || height <= 0 || width > maxScriptDimension || height > maxScriptDimension {
		return Style{}, ctxerrors.Wrap(commerr.ErrInvalidValue, "position needs positive 32-bit script dimensions")
	}
	if p.Alignment < AlignmentBottomLeft || p.Alignment > AlignmentTopRight {
		return Style{}, ctxerrors.Wrap(commerr.ErrInvalidValue, "text alignment must be from 1 to 9")
	}
	for _, ratio := range []float64{p.MarginRatio, p.LeftMarginRatio, p.RightMarginRatio} {
		if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 || ratio >= 1 {
			return Style{}, ctxerrors.Wrap(commerr.ErrInvalidValue, "position ratios must be finite and in [0,1)")
		}
	}
	if p.LeftMarginRatio+p.RightMarginRatio >= 1 ||
		(p.Alignment >= AlignmentMiddleLeft && p.Alignment <= AlignmentMiddleRight && p.MarginRatio != 0) {
		return Style{}, ctxerrors.Wrap(commerr.ErrInvalidValue, "position has no horizontal space or uses an ignored center margin")
	}
	style.Alignment = p.Alignment
	style.MarginL = int(float64(width) * p.LeftMarginRatio)
	style.MarginR = int(float64(width) * p.RightMarginRatio)
	style.MarginV = int(float64(height) * p.MarginRatio)
	return style, nil
}

// WithTextPosition returns a script with position geometry applied to every style.
func (s Script) WithTextPosition(position TextPosition) (Script, error) {
	result := s
	result.Styles = slices.Clone(s.Styles)
	if _, err := position.style(Style{}, s.Resolution.Width, s.Resolution.Height); err != nil {
		return Script{}, ctxerrors.Wrap(err, "resolve text position")
	}
	for index, style := range result.Styles {
		updated, err := position.style(style, s.Resolution.Width, s.Resolution.Height)
		if err != nil {
			return Script{}, ctxerrors.Wrap(err, "position script style")
		}
		result.Styles[index] = updated
	}
	return result, nil
}

// WithTextPosition changes style defaults, preserving all event fields and inline tags.
// An empty style name selects every style.
func (d *Document) WithTextPosition(position TextPosition, name string) (*Document, error) {
	width, height, err := d.resolution()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "resolve script canvas")
	}
	style, err := position.style(Style{}, width, height)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "resolve text position")
	}
	edit := Edit{}
	for _, row := range d.styles {
		styleName := strings.TrimSpace(row.field(columnName))
		if name != "" && name != styleName {
			continue
		}
		edit.Styles = append(edit.Styles, StyleEdit{Name: styleName, Fields: map[string]string{
			columnAlignment: integer(int(style.Alignment)), columnMarginL: integer(style.MarginL),
			columnMarginR: integer(style.MarginR), columnMarginV: integer(style.MarginV),
		}})
	}
	if len(edit.Styles) == 0 {
		return nil, ctxerrors.Wrapf(commerr.ErrNotFound, "style %q", name)
	}
	updated, err := d.Apply(edit)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "apply text position")
	}

	return updated, nil
}

func (d *Document) resolution() (int, int, error) {
	var dimensions [2]int
	for index, key := range []string{strings.ToLower(infoPlayResX), strings.ToLower(infoPlayResY)} {
		row, ok := d.info[key]
		if !ok {
			continue
		}
		value, err := strconv.ParseInt(strings.TrimSpace(row.values[0]), 10, 32)
		if err != nil || value <= 0 {
			return 0, 0, ctxerrors.Wrapf(commerr.ErrInvalidValue, "invalid %s script dimension", key)
		}
		dimensions[index] = int(value)
	}
	width, height := dimensions[0], dimensions[1]
	switch {
	case width == 0 && height == 0:
		return defaultScriptWidth, defaultScriptHeight, nil
	case height == 0 && width == legacyWideWidth:
		height = legacyWideHeight
	case width == 0 && height == legacyWideHeight:
		width = legacyWideWidth
	case height == 0:
		height = max(1, width-1-(width-1)/4)
	case width == 0:
		width = int(min(int64(height)+int64(height)/3, int64(maxScriptDimension)))
	}
	return width, height, nil
}
