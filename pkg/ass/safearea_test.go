package ass

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const safeStyleName = "Default"

func safeStyle(name string, alignment Alignment, left, right, vertical int) Style {
	return Style{
		Name: name, FontName: FontDejaVuSans, Alignment: alignment,
		MarginL: left, MarginR: right, MarginV: vertical,
	}
}

func safeEvent(style string, left, right, vertical int) Event {
	return Event{Style: style, End: time.Second, Text: "x", MarginL: left, MarginR: right, MarginV: vertical}
}

func safeDocument(t *testing.T, width, height int, styles []Style, events ...Event) *Document {
	t.Helper()

	script := NewScript("Safe", width, height)
	for _, style := range styles {
		script.AddStyle(style)
	}
	for _, event := range events {
		script.AddEvent(event)
	}
	data, err := script.MarshalText()
	require.NoError(t, err)

	return parseDocument(t, string(data))
}

func mustSafeArea(t *testing.T, name string) SafeArea {
	t.Helper()

	area, err := LookupSafeArea(name)
	require.NoError(t, err)

	return area
}

func styleIssue(edge string, margin, required int) SafeAreaIssue {
	return SafeAreaIssue{Style: safeStyleName, Edge: edge, Margin: margin, Required: required}
}

func eventIssue(event int, edge string, margin, required int) SafeAreaIssue {
	return SafeAreaIssue{Style: safeStyleName, Event: event, Edge: edge, Margin: margin, Required: required}
}

func TestSafeAreasAndLookup(t *testing.T) {
	t.Parallel()

	areas := SafeAreas()
	names := make([]string, 0, len(areas))
	for _, area := range areas {
		names = append(names, area.Name)
	}
	assert.Equal(t, []string{SafeAreaAll, "tiktok", "shorts", "reels"}, names)

	// Each row is the area's margins in pixels on a 1080 x 1920 canvas.
	testCases := []struct {
		name       string
		top        float64
		bottom     float64
		left       float64
		right      float64
		lowerRight float64
		lowerFrom  float64
	}{
		{name: SafeAreaAll, top: 288, bottom: 672, left: 120, right: 192, lowerRight: 300, lowerFrom: 840},
		{name: "tiktok", top: 240, bottom: 660, left: 120, right: 120, lowerRight: 300, lowerFrom: 840},
		{name: "shorts", top: 288, bottom: 672, left: 48, right: 192, lowerRight: 192, lowerFrom: 1920},
		{name: "reels", top: 268.8, bottom: 672, left: 64.8, right: 64.8, lowerRight: 64.8, lowerFrom: 1920},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			area := mustSafeArea(t, tc.name)
			assert.Equal(t, tc.name, area.Name)
			assert.InDelta(t, tc.top, area.TopRatio*referenceHeight, 1e-9)
			assert.InDelta(t, tc.bottom, area.BottomRatio*referenceHeight, 1e-9)
			assert.InDelta(t, tc.left, area.LeftRatio*referenceWidth, 1e-9)
			assert.InDelta(t, tc.right, area.RightRatio*referenceWidth, 1e-9)
			assert.InDelta(t, tc.lowerRight, area.LowerRightRatio*referenceWidth, 1e-9)
			assert.InDelta(t, tc.lowerFrom, area.LowerFromRatio*referenceHeight, 1e-9)
		})
	}

	t.Run("returns independent values", func(t *testing.T) {
		t.Parallel()

		changed := SafeAreas()
		changed[0].Name = "changed"
		assert.Equal(t, SafeAreaAll, mustSafeArea(t, SafeAreaAll).Name)
	})

	t.Run("reports a missing name", func(t *testing.T) {
		t.Parallel()

		_, err := LookupSafeArea("missing")
		require.ErrorIs(t, err, commerr.ErrNotFound)
	})
}

func TestCheckSafeAreaStyleMargins(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		area      string
		alignment Alignment
		left      int
		right     int
		vertical  int
		want      []SafeAreaIssue
	}{
		{name: "all clears bottom text", area: SafeAreaAll, alignment: AlignmentBottomCenter, left: 120, right: 300, vertical: 672},
		{
			name: "all bottom margin one short", area: SafeAreaAll, alignment: AlignmentBottomCenter, left: 120, right: 300, vertical: 671,
			want: []SafeAreaIssue{styleIssue(EdgeBottom, 671, 672)},
		},
		{
			name: "all left margin one short", area: SafeAreaAll, alignment: AlignmentBottomCenter, left: 119, right: 300, vertical: 672,
			want: []SafeAreaIssue{styleIssue(EdgeLeft, 119, 120)},
		},
		{
			name: "all lower-right column needs 300", area: SafeAreaAll, alignment: AlignmentBottomCenter, left: 120, right: 299, vertical: 672,
			want: []SafeAreaIssue{styleIssue(EdgeRight, 299, 300)},
		},
		{
			name: "all reports every edge", area: SafeAreaAll, alignment: AlignmentBottomCenter,
			want: []SafeAreaIssue{
				styleIssue(EdgeBottom, 0, 672), styleIssue(EdgeLeft, 0, 120), styleIssue(EdgeRight, 0, 300),
			},
		},
		{
			name: "all anchor line on the lower boundary uses the upper right margin", area: SafeAreaAll,
			alignment: AlignmentBottomCenter, left: 120, right: 192, vertical: 1080,
		},
		{
			name: "all anchor line below the boundary uses the lower right margin", area: SafeAreaAll,
			alignment: AlignmentBottomCenter, left: 120, right: 192, vertical: 1079,
			want: []SafeAreaIssue{styleIssue(EdgeRight, 192, 300)},
		},
		{name: "all clears top text", area: SafeAreaAll, alignment: AlignmentTopCenter, left: 120, right: 192, vertical: 288},
		{
			name: "all top margin one short", area: SafeAreaAll, alignment: AlignmentTopCenter, left: 120, right: 192, vertical: 287,
			want: []SafeAreaIssue{styleIssue(EdgeTop, 287, 288)},
		},
		{
			name: "all top text hanging into the lower column", area: SafeAreaAll,
			alignment: AlignmentTopCenter, left: 120, right: 192, vertical: 900,
			want: []SafeAreaIssue{styleIssue(EdgeRight, 192, 300)},
		},
		{
			name: "all top text reports every edge", area: SafeAreaAll, alignment: AlignmentTopLeft,
			want: []SafeAreaIssue{
				styleIssue(EdgeTop, 0, 288), styleIssue(EdgeLeft, 0, 120), styleIssue(EdgeRight, 0, 192),
			},
		},
		{name: "all clears centered text", area: SafeAreaAll, alignment: AlignmentMiddleCenter, left: 120, right: 300},
		{
			name: "all centered text ignores the vertical margin but sits in the lower column", area: SafeAreaAll,
			alignment: AlignmentMiddleCenter, left: 120, right: 192, vertical: 5,
			want: []SafeAreaIssue{styleIssue(EdgeRight, 192, 300)},
		},
		{name: "tiktok clears bottom text", area: "tiktok", alignment: AlignmentBottomCenter, left: 120, right: 300, vertical: 660},
		{
			name: "tiktok bottom margin one short", area: "tiktok", alignment: AlignmentBottomCenter, left: 120, right: 300, vertical: 659,
			want: []SafeAreaIssue{styleIssue(EdgeBottom, 659, 660)},
		},
		{name: "tiktok clears top text", area: "tiktok", alignment: AlignmentTopRight, left: 120, right: 120, vertical: 240},
		{
			name: "tiktok top margin one short", area: "tiktok", alignment: AlignmentTopRight, left: 120, right: 120, vertical: 239,
			want: []SafeAreaIssue{styleIssue(EdgeTop, 239, 240)},
		},
		{
			name: "tiktok centered text needs the button column", area: "tiktok", alignment: AlignmentMiddleRight, left: 120, right: 120,
			want: []SafeAreaIssue{styleIssue(EdgeRight, 120, 300)},
		},
		{name: "shorts clears bottom text", area: "shorts", alignment: AlignmentBottomRight, left: 48, right: 192, vertical: 672},
		{
			name: "shorts right margin one short", area: "shorts", alignment: AlignmentBottomRight, left: 48, right: 191, vertical: 672,
			want: []SafeAreaIssue{styleIssue(EdgeRight, 191, 192)},
		},
		{
			name: "shorts left margin one short", area: "shorts", alignment: AlignmentMiddleLeft, left: 47, right: 192,
			want: []SafeAreaIssue{styleIssue(EdgeLeft, 47, 48)},
		},
		{name: "reels clears bottom text", area: "reels", alignment: AlignmentBottomLeft, left: 65, right: 65, vertical: 672},
		{name: "reels clears top text", area: "reels", alignment: AlignmentTopCenter, left: 65, right: 65, vertical: 269},
		{
			name: "reels top margin one short", area: "reels", alignment: AlignmentTopCenter, left: 65, right: 65, vertical: 268,
			want: []SafeAreaIssue{styleIssue(EdgeTop, 268, 269)},
		},
		{
			name: "reels side margins one short", area: "reels", alignment: AlignmentMiddleCenter, left: 64, right: 64,
			want: []SafeAreaIssue{styleIssue(EdgeLeft, 64, 65), styleIssue(EdgeRight, 64, 65)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			document := safeDocument(t, 1080, 1920,
				[]Style{safeStyle(safeStyleName, tc.alignment, tc.left, tc.right, tc.vertical)})
			issues, err := document.CheckSafeArea(mustSafeArea(t, tc.area))
			require.NoError(t, err)
			assert.Equal(t, tc.want, issues)
		})
	}
}

func TestCheckSafeAreaEveryAlignment(t *testing.T) {
	t.Parallel()

	for alignment := AlignmentBottomLeft; alignment <= AlignmentTopRight; alignment++ {
		t.Run(fmt.Sprintf("alignment %d", alignment), func(t *testing.T) {
			t.Parallel()

			// 120, 300 and 672 pixels clear the combined area for every anchor.
			document := safeDocument(t, 1080, 1920, []Style{safeStyle(safeStyleName, alignment, 120, 300, 672)})
			issues, err := document.CheckSafeArea(mustSafeArea(t, SafeAreaAll))
			require.NoError(t, err)
			assert.Empty(t, issues)

			bare := safeDocument(t, 1080, 1920, []Style{safeStyle(safeStyleName, alignment, 0, 0, 0)})
			issues, err = bare.CheckSafeArea(mustSafeArea(t, SafeAreaAll))
			require.NoError(t, err)

			var edges []string
			for _, issue := range issues {
				edges = append(edges, issue.Edge)
			}
			switch {
			case alignment >= AlignmentTopLeft:
				assert.Equal(t, []string{EdgeTop, EdgeLeft, EdgeRight}, edges)
			case alignment >= AlignmentMiddleLeft:
				assert.Equal(t, []string{EdgeLeft, EdgeRight}, edges)
			default:
				assert.Equal(t, []string{EdgeBottom, EdgeLeft, EdgeRight}, edges)
			}
		})
	}
}

func TestCheckSafeAreaEventMargins(t *testing.T) {
	t.Parallel()

	compliant := safeStyle(safeStyleName, AlignmentBottomCenter, 120, 300, 672)

	testCases := []struct {
		name   string
		styles []Style
		events []Event
		want   []SafeAreaIssue
	}{
		{
			name:   "an event without margins keeps the style's",
			styles: []Style{compliant},
			events: []Event{safeEvent(safeStyleName, 0, 0, 0)},
		},
		{
			name:   "an event with safe margins is clean",
			styles: []Style{compliant},
			events: []Event{safeEvent(safeStyleName, 130, 310, 700)},
		},
		{
			name:   "event left margin overrides the style",
			styles: []Style{compliant},
			events: []Event{safeEvent(safeStyleName, 50, 0, 0)},
			want:   []SafeAreaIssue{eventIssue(1, EdgeLeft, 50, 120)},
		},
		{
			name:   "event right margin overrides the style",
			styles: []Style{compliant},
			events: []Event{safeEvent(safeStyleName, 0, 250, 0)},
			want:   []SafeAreaIssue{eventIssue(1, EdgeRight, 250, 300)},
		},
		{
			name:   "event vertical margin overrides the style",
			styles: []Style{compliant},
			events: []Event{safeEvent(safeStyleName, 0, 0, 100)},
			want:   []SafeAreaIssue{eventIssue(1, EdgeBottom, 100, 672)},
		},
		{
			name:   "event margins fix an unsafe style for that event only",
			styles: []Style{safeStyle(safeStyleName, AlignmentBottomCenter, 120, 300, 100)},
			events: []Event{safeEvent(safeStyleName, 0, 0, 700)},
			want:   []SafeAreaIssue{styleIssue(EdgeBottom, 100, 672)},
		},
		{
			name:   "an event without margins is skipped even on an unsafe style",
			styles: []Style{safeStyle(safeStyleName, AlignmentBottomCenter, 120, 300, 100)},
			events: []Event{safeEvent(safeStyleName, 0, 0, 0)},
			want:   []SafeAreaIssue{styleIssue(EdgeBottom, 100, 672)},
		},
		{
			name:   "an event uses its style's alignment",
			styles: []Style{safeStyle(safeStyleName, AlignmentTopCenter, 120, 192, 288)},
			events: []Event{safeEvent(safeStyleName, 0, 0, 100)},
			want:   []SafeAreaIssue{eventIssue(1, EdgeTop, 100, 288)},
		},
		{
			name:   "an event margin can move text into the lower column",
			styles: []Style{safeStyle(safeStyleName, AlignmentTopCenter, 120, 192, 288)},
			events: []Event{safeEvent(safeStyleName, 0, 0, 900)},
			want:   []SafeAreaIssue{eventIssue(1, EdgeRight, 192, 300)},
		},
		{
			name:   "events are numbered from one in file order",
			styles: []Style{compliant},
			events: []Event{safeEvent(safeStyleName, 130, 0, 0), safeEvent(safeStyleName, 10, 0, 0), safeEvent(safeStyleName, 20, 0, 0)},
			want:   []SafeAreaIssue{eventIssue(2, EdgeLeft, 10, 120), eventIssue(3, EdgeLeft, 20, 120)},
		},
		{
			name:   "an event is checked against its own style",
			styles: []Style{compliant, safeStyle("Top", AlignmentTopCenter, 120, 192, 288)},
			events: []Event{safeEvent("Top", 0, 0, 10)},
			want:   []SafeAreaIssue{{Style: "Top", Event: 1, Edge: EdgeTop, Margin: 10, Required: 288}},
		},
		{
			name: "style and event issues are listed together, styles first",
			styles: []Style{
				compliant,
				safeStyle("Bare", AlignmentMiddleCenter, 0, 300, 0),
			},
			events: []Event{safeEvent(safeStyleName, 0, 1, 0)},
			want: []SafeAreaIssue{
				{Style: "Bare", Edge: EdgeLeft, Margin: 0, Required: 120},
				eventIssue(1, EdgeRight, 1, 300),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			document := safeDocument(t, 1080, 1920, tc.styles, tc.events...)
			issues, err := document.CheckSafeArea(mustSafeArea(t, SafeAreaAll))
			require.NoError(t, err)
			assert.Equal(t, tc.want, issues)
		})
	}
}

func TestCheckSafeAreaScalesWithTheCanvas(t *testing.T) {
	t.Parallel()

	t.Run("half size canvas halves the margins", func(t *testing.T) {
		t.Parallel()

		document := safeDocument(t, 540, 960, []Style{safeStyle(safeStyleName, AlignmentBottomCenter, 60, 150, 336)})
		issues, err := document.CheckSafeArea(mustSafeArea(t, SafeAreaAll))
		require.NoError(t, err)
		assert.Empty(t, issues)

		short := safeDocument(t, 540, 960, []Style{safeStyle(safeStyleName, AlignmentBottomCenter, 59, 149, 335)})
		issues, err = short.CheckSafeArea(mustSafeArea(t, SafeAreaAll))
		require.NoError(t, err)
		assert.Equal(t, []SafeAreaIssue{
			styleIssue(EdgeBottom, 335, 336), styleIssue(EdgeLeft, 59, 60), styleIssue(EdgeRight, 149, 150),
		}, issues)
	})

	t.Run("a script without a resolution uses the libass default canvas", func(t *testing.T) {
		t.Parallel()

		document := parseDocument(t, documentHeader)
		issues, err := document.CheckSafeArea(mustSafeArea(t, SafeAreaAll))
		require.NoError(t, err)
		assert.Equal(t, []SafeAreaIssue{
			styleIssue(EdgeBottom, 0, 101), styleIssue(EdgeLeft, 0, 43), styleIssue(EdgeRight, 0, 107),
		}, issues)
	})
}

// safeAreaCanvases include sizes whose margins do not land on whole pixels,
// where rounding to the nearest pixel would fall one pixel short.
var safeAreaCanvases = [][2]int{{1080, 1920}, {540, 960}, {720, 1280}, {1000, 1778}, {1084, 1927}, {1081, 1921}}

func TestEveryTextPositionClearsTheCombinedSafeArea(t *testing.T) {
	t.Parallel()

	area := mustSafeArea(t, SafeAreaAll)
	for _, canvas := range safeAreaCanvases {
		for _, position := range TextPositions() {
			t.Run(fmt.Sprintf("%s/%dx%d", position.Name, canvas[0], canvas[1]), func(t *testing.T) {
				t.Parallel()

				script := NewScript("Position", canvas[0], canvas[1])
				script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans})
				positioned, err := script.WithTextPosition(position)
				require.NoError(t, err)
				data, err := positioned.MarshalText()
				require.NoError(t, err)

				issues, err := parseDocument(t, string(data)).CheckSafeArea(area)
				require.NoError(t, err)
				assert.Empty(t, issues)
			})
		}
	}
}

func TestSocialPresetsClearTheCombinedSafeArea(t *testing.T) {
	t.Parallel()

	area := mustSafeArea(t, SafeAreaAll)
	social := []string{
		"tiktok-bold", "reels-bold", "shorts-impact", "social-box", "the-bold", "social-center", "social-hook",
	}
	for _, name := range social {
		for _, canvas := range safeAreaCanvases {
			t.Run(fmt.Sprintf("%s/%dx%d", name, canvas[0], canvas[1]), func(t *testing.T) {
				t.Parallel()

				preset, err := LookupPreset(name)
				require.NoError(t, err)
				script := preset.Captions(CaptionSpec{
					Width: canvas[0], Height: canvas[1],
					Dialogues: []Dialogue{{End: time.Second, Text: "Safe words"}},
				})
				data, err := script.MarshalText()
				require.NoError(t, err)

				issues, err := parseDocument(t, string(data)).CheckSafeArea(area)
				require.NoError(t, err)
				assert.Empty(t, issues)
			})
		}
	}

	t.Run("a preset without side margins does not", func(t *testing.T) {
		t.Parallel()

		preset, err := LookupPreset(PresetBlackGold)
		require.NoError(t, err)
		script := preset.Captions(CaptionSpec{Width: 1080, Height: 1920})
		data, err := script.MarshalText()
		require.NoError(t, err)

		issues, err := parseDocument(t, string(data)).CheckSafeArea(area)
		require.NoError(t, err)
		assert.NotEmpty(t, issues)
	})
}

func TestCheckSafeAreaRejectsUnreadableValues(t *testing.T) {
	t.Parallel()

	source := fixture(t)
	styleLine := "Style: Default,DejaVu Sans,50,&H00FFFFFF,&H00FFFFFF,&H00000000,&H00000000,-1,0,0,0,100,100,0,0,1,0,0,2,0,0,0,1"
	require.Contains(t, source, styleLine)
	eventPrefix := "Dialogue: 0,0:00:00.25,0:00:02.00,Default,,0,0,0,,"
	require.Contains(t, source, eventPrefix)

	testCases := []struct {
		name   string
		source string
	}{
		{name: "script width", source: strings.Replace(source, "PlayResX: 1080", "PlayResX: wide", 1)},
		{name: "style left margin", source: strings.Replace(source, styleLine, strings.Replace(styleLine, ",2,0,0,0,1", ",2,x,0,0,1", 1), 1)},
		{name: "style right margin", source: strings.Replace(source, styleLine, strings.Replace(styleLine, ",2,0,0,0,1", ",2,0,1.5,0,1", 1), 1)},
		{name: "style vertical margin", source: strings.Replace(source, styleLine, strings.Replace(styleLine, ",2,0,0,0,1", ",2,0,0,tall,1", 1), 1)},
		{name: "style alignment", source: strings.Replace(source, styleLine, strings.Replace(styleLine, ",0,0,2,0,0,0,1", ",0,0,low,0,0,0,1", 1), 1)},
		{name: "event left margin", source: strings.Replace(source, eventPrefix, "Dialogue: 0,0:00:00.25,0:00:02.00,Default,,left,0,0,,", 1)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.NotEqual(t, source, tc.source, "test input was not changed")
			issues, err := parseDocument(t, tc.source).CheckSafeArea(mustSafeArea(t, SafeAreaAll))
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Nil(t, issues)
		})
	}
}
