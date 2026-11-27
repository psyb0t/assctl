package ass

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	bannerDelayMillis = 5
	bannerStep        = 500 * time.Millisecond
	bannerPixels      = int(bannerStep/time.Millisecond) / bannerDelayMillis
)

func textScript(effect Effect, text string) Script {
	script := drawingScript(nil)
	script.Styles[0].FontSize = 48
	script.Styles[0].PrimaryColor = ColorWhite
	script.Events[0].End = 3 * time.Second
	script.Events[0].Text = text
	script.Events[0].Effect = effect

	return script
}

func leftEdgeAt(t *testing.T, script Script, at time.Duration) int {
	t.Helper()

	box := visibleBounds(renderFrame(t, script, at, 640, 360))
	require.False(t, box.Empty(), "no text at %v", at)

	return box.Min.X
}

func TestRenderBannerEffectMovesText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		leftToRight bool
		wantShift   int
	}{
		{name: "right to left", leftToRight: false, wantShift: -bannerPixels},
		{name: "left to right", leftToRight: true, wantShift: bannerPixels},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			effect, err := BannerEffect(bannerDelayMillis, tc.leftToRight, 0)
			require.NoError(t, err)
			script := textScript(effect, "BANNER")

			earlier := leftEdgeAt(t, script, time.Second)
			later := leftEdgeAt(t, script, time.Second+bannerStep)
			assert.InDelta(t, tc.wantShift, later-earlier, 2, "moved from %d to %d", earlier, later)
		})
	}

	t.Run("text without an effect stays put", func(t *testing.T) {
		t.Parallel()

		script := textScript("", "BANNER")
		assert.Equal(t, leftEdgeAt(t, script, time.Second), leftEdgeAt(t, script, time.Second+bannerStep))
	})
}

func TestRenderEscapedBraceIsDrawn(t *testing.T) {
	t.Parallel()

	const plain = "{ABC}"

	raw := visibleBounds(renderFrame(t, textScript("", plain), time.Second, 640, 360))
	assert.True(t, raw.Empty(), "an unescaped brace should hide the text as an override block: %v", raw)

	escaped := visibleBounds(renderFrame(t, textScript("", EscapeText(plain)), time.Second, 640, 360))
	require.False(t, escaped.Empty(), "escaped text drew nothing")

	bare := visibleBounds(renderFrame(t, textScript("", "ABC}"), time.Second, 640, 360))
	assert.Greater(t, escaped.Dx(), bare.Dx(), "the escaped brace itself should take room")
}
