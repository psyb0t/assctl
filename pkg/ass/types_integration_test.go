package ass

import (
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	boxCanvasWidth  = 640
	boxCanvasHeight = 360
	boxPadding      = 12
)

// isBoxRed reports whether a pixel has the saturated red used for the box fill.
func isBoxRed(frame image.Image, x, y int) bool {
	r, g, b, _ := frame.At(x, y).RGBA()

	return r>>8 > 200 && g>>8 < 60 && b>>8 < 60
}

// redSpan returns the first and last red column in row y, or ok false when the row has none.
func redSpan(frame image.Image, y int) (first, last int, ok bool) {
	for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
		if !isBoxRed(frame, x, y) {
			continue
		}
		if !ok {
			first, ok = x, true
		}
		last = x
	}

	return first, last, ok
}

// redRows returns the first and last row that holds a red pixel.
func redRows(frame image.Image) (top, bottom int, ok bool) {
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		if _, _, found := redSpan(frame, y); !found {
			continue
		}
		if !ok {
			top, ok = y, true
		}
		bottom = y
	}

	return top, bottom, ok
}

// boxScript shows a short line above a long line, so a box per line and a box
// for the whole event differ in shape.
func boxScript(style BorderStyle) Script {
	script := NewScript("Box", boxCanvasWidth, boxCanvasHeight)
	box := Style{
		Name: "Default", FontName: FontDejaVuSans, FontSize: 48, ScaleX: 100, ScaleY: 100,
		Alignment: AlignmentMiddleCenter, PrimaryColor: ColorWhite, BorderStyle: style,
	}
	switch style {
	case BorderStyleOpaqueBox:
		box.OutlineColor, box.Outline = ColorRed, boxPadding
	case BorderStyleEventBox:
		box.ShadowColor, box.Shadow = ColorRed, boxPadding
	}
	script.AddStyle(box)
	script.AddEvent(Event{Style: "Default", End: 2 * time.Second, Text: `I\NWWWWWWWW`})

	return *script
}

func TestRenderBorderStyleEventBoxDrawsOneBox(t *testing.T) {
	t.Parallel()

	perLine := renderFrame(t, boxScript(BorderStyleOpaqueBox), time.Second, boxCanvasWidth, boxCanvasHeight)
	wholeEvent := renderFrame(t, boxScript(BorderStyleEventBox), time.Second, boxCanvasWidth, boxCanvasHeight)

	perLineTop, perLineBottom, ok := redRows(perLine)
	require.True(t, ok, "style 3 drew no box")
	wholeTop, wholeBottom, ok := redRows(wholeEvent)
	require.True(t, ok, "style 4 drew no box")

	perLineFirstTop, perLineLastTop, _ := redSpan(perLine, perLineTop)
	perLineFirstBottom, perLineLastBottom, _ := redSpan(perLine, perLineBottom)
	wholeFirstTop, wholeLastTop, _ := redSpan(wholeEvent, wholeTop)
	wholeFirstBottom, wholeLastBottom, _ := redSpan(wholeEvent, wholeBottom)

	// Style 3 pads each line on its own, so the short first line has a narrow box.
	assert.Less(t, perLineLastTop-perLineFirstTop, (perLineLastBottom-perLineFirstBottom)/2,
		"style 3 top box %d..%d is not narrower than the bottom box %d..%d",
		perLineFirstTop, perLineLastTop, perLineFirstBottom, perLineLastBottom)

	// Style 4 is one rectangle: both outer rows span the same columns.
	assert.Equal(t, wholeFirstTop, wholeFirstBottom, "left edge differs between the top and bottom rows")
	assert.Equal(t, wholeLastTop, wholeLastBottom, "right edge differs between the top and bottom rows")

	// That rectangle is as wide as the widest line's box, up to antialiasing.
	assert.InDelta(t, perLineFirstBottom, wholeFirstTop, 2)
	assert.InDelta(t, perLineLastBottom, wholeLastTop, 2)
	assert.Greater(t, wholeLastTop-wholeFirstTop, 4*(perLineLastTop-perLineFirstTop),
		"style 4 box does not widen the short first line")
}
