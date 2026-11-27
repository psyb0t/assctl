package ass

import (
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderDrawingClipMovementAndRawPrecedence(t *testing.T) {
	t.Parallel()

	const squareClip = "m 0 0 l 50 0 50 360 0 360"

	testCases := []struct {
		name     string
		override Override
		text     string
		at       time.Duration
		want     image.Rectangle
	}{
		{
			name:     "drawing",
			override: Override{DrawingMode: new(1), Position: &Position{X: 100, Y: 100}},
			want:     image.Rect(100, 100, 200, 200),
		},
		{
			name:     "vector clip",
			override: Override{DrawingMode: new(1), Position: &Position{}, VectorClip: &VectorClip{Drawing: squareClip}},
			want:     image.Rect(0, 0, 50, 100),
		},
		{
			name: "inverse vector clip",
			override: Override{
				DrawingMode: new(1), Position: &Position{},
				VectorClip: &VectorClip{Drawing: squareClip, Inverse: true},
			},
			want: image.Rect(50, 0, 100, 100),
		},
		{
			name:     "rectangle clip",
			override: Override{DrawingMode: new(1), Position: &Position{}, Clip: &Clip{X2: 50, Y2: 360}},
			want:     image.Rect(0, 0, 50, 100),
		},
		{
			name:     "movement start",
			override: Override{DrawingMode: new(1), Movement: &Movement{X2: 200}},
			want:     image.Rect(0, 0, 100, 100),
		},
		{
			name:     "movement halfway",
			override: Override{DrawingMode: new(1), Movement: &Movement{X2: 200}},
			at:       time.Second,
			want:     image.Rect(100, 0, 200, 100),
		},
		{
			name:     "explicit inline position",
			override: Override{},
			text:     `{\an7\pos(300,200)\p1}m 0 0 l 100 0 100 100 0 100`,
			want:     image.Rect(300, 200, 400, 300),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := drawingScript(&tc.override)
			if tc.text != "" {
				script.Events[0].Text = tc.text
				var err error
				script, err = script.WithTextPosition(TextPositions()[0])
				require.NoError(t, err)
			}

			got := visibleBounds(renderFrame(t, script, tc.at, 640, 360))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRenderFadeAndKaraokeTiming(t *testing.T) {
	t.Parallel()

	fadeCases := []struct {
		name string
		fade Override
	}{
		{
			name: "simple fade",
			fade: Override{SimpleFade: &SimpleFade{In: time.Second, Out: time.Second}},
		},
		{
			name: "complex fade",
			fade: Override{Fade: &Fade{A1: 255, A2: 0, A3: 255, T2: time.Second, T3: time.Second, T4: 2 * time.Second}},
		},
	}

	for _, tc := range fadeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fade := tc.fade
			fade.DrawingMode = new(1)
			fade.Position = &Position{X: 100, Y: 100}
			script := drawingScript(&fade)

			assert.True(t, visibleBounds(renderFrame(t, script, 0, 640, 360)).Empty(), "fade starts visible")
			assert.False(t, visibleBounds(renderFrame(t, script, time.Second, 640, 360)).Empty(), "fade never becomes visible")
		})
	}

	t.Run("karaoke sweep", func(t *testing.T) {
		t.Parallel()

		script := drawingScript(&Override{
			Position: &Position{X: 20, Y: 100},
			Karaoke:  &Karaoke{Mode: KaraokeSweep, Duration: time.Second},
		})
		script.Styles[0].PrimaryColor = ColorWhite
		script.Styles[0].SecondColor = ColorRed
		script.Events[0].Text = "KARAOKE"

		previous := -1
		for _, at := range []time.Duration{0, 500 * time.Millisecond, 1500 * time.Millisecond} {
			frame := renderFrame(t, script, at, 640, 360)
			white := 0
			for y := range 360 {
				for x := range 640 {
					r, g, b, _ := frame.At(x, y).RGBA()
					if r>>8 > 200 && g>>8 > 200 && b>>8 > 200 {
						white++
					}
				}
			}
			if at == 0 {
				require.Zero(t, white)
			}
			require.Greater(t, white, previous, "karaoke did not progress at %v", at)
			previous = white
		}
	})
}

func TestRenderTransformsRotationTransparencyAndComments(t *testing.T) {
	t.Parallel()

	t.Run("scale animation", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			at    time.Duration
			width int
		}{
			{name: "start", at: 0, width: 100},
			{name: "halfway", at: 500 * time.Millisecond, width: 150},
			{name: "finished", at: 1500 * time.Millisecond, width: 200},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				script := drawingScript(&Override{
					DrawingMode: new(1), Position: &Position{X: 100, Y: 100},
					Transform: []Transform{{End: time.Second, Override: &Override{ScaleX: new(200.0)}}},
				})
				box := visibleBounds(renderFrame(t, script, tc.at, 640, 360))
				assert.Equal(t, tc.width, box.Dx(), "rendered %v", box)
			})
		}
	})

	t.Run("rotation swaps drawing dimensions", func(t *testing.T) {
		t.Parallel()

		rotated := drawingScript(&Override{
			DrawingMode: new(1), Position: &Position{X: 200, Y: 150}, RotationZ: new(90.0),
		})
		rotated.Events[0].Text = "m 0 0 l 100 0 100 50 0 50"
		box := visibleBounds(renderFrame(t, rotated, time.Second, 640, 360))
		assert.Equal(t, 50, box.Dx(), "rendered %v", box)
		assert.Equal(t, 100, box.Dy(), "rendered %v", box)
	})

	t.Run("transparent fill", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			override Override
		}{
			{name: "global alpha", override: Override{DrawingMode: new(1), GlobalAlpha: new(AlphaTransparent)}},
			{name: "primary alpha", override: Override{DrawingMode: new(1), PrimaryAlpha: new(AlphaTransparent)}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				box := visibleBounds(renderFrame(t, drawingScript(&tc.override), time.Second, 640, 360))
				assert.True(t, box.Empty(), "transparent fill is visible: %v", box)
			})
		}
	})

	t.Run("comment event", func(t *testing.T) {
		t.Parallel()

		comment := drawingScript(&Override{DrawingMode: new(1)})
		comment.Events[0].Comment = true
		box := visibleBounds(renderFrame(t, comment, time.Second, 640, 360))
		assert.True(t, box.Empty(), "comment event rendered pixels: %v", box)
	})

	t.Run("outside the event interval", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			at   time.Duration
		}{
			{name: "before start", at: 0},
			{name: "after end", at: 2 * time.Second},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				script := drawingScript(&Override{DrawingMode: new(1)})
				script.Events[0].Start = time.Second
				box := visibleBounds(renderFrame(t, script, tc.at, 640, 360))
				assert.True(t, box.Empty(), "event visible outside its interval: %v", box)
			})
		}
	})
}

func TestRenderAutomaticAndForcedWrapping(t *testing.T) {
	t.Parallel()

	script := drawingScript(nil)
	script.Styles[0].FontSize = 40
	script.Styles[0].MarginL = 20
	script.Styles[0].MarginR = 20
	script.Events[0].Text = "ONE TWO THREE FOUR FIVE SIX SEVEN EIGHT NINE TEN"
	script.Events[0].Override = &Override{WrapStyle: WrapStyleEndOfLine.Ptr()}
	wrapped := visibleBounds(renderFrame(t, script, time.Second, 640, 360))

	script.Events[0].Override.WrapStyle = WrapStyleNoWrap.Ptr()
	unwrapped := visibleBounds(renderFrame(t, script, time.Second, 640, 360))
	require.False(t, unwrapped.Empty())
	assert.Greater(t, wrapped.Dy(), unwrapped.Dy())

	script.Events[0].Text = `ONE\NTWO`
	explicit := visibleBounds(renderFrame(t, script, time.Second, 640, 360))
	assert.Greater(t, explicit.Dy(), unwrapped.Dy(), "explicit line break ignored under no-wrap")
}
