package ass

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderEveryThemeAndPosition(t *testing.T) {
	t.Parallel()

	for _, preset := range Presets() {
		for _, position := range TextPositions() {
			for _, canvas := range [][2]int{{540, 960}, {960, 540}, {640, 640}} {
				name := fmt.Sprintf("%s/%s/%dx%d", preset.Name, position.Name, canvas[0], canvas[1])
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					// Geometry stays deterministic offline; upstream font families have a separate real-download test.
					offline := preset
					offline.Font = FontDejaVuSans
					script := offline.Captions(CaptionSpec{
						Width: canvas[0], Height: canvas[1], WordsPerLine: 3,
						Dialogues: []Dialogue{{End: 2 * time.Second, Text: "MAKE EVERY WORD COUNT"}},
					})
					positioned, err := script.WithTextPosition(position)
					require.NoError(t, err)

					box := visibleBounds(renderFrame(t, positioned, time.Second, canvas[0], canvas[1]))
					require.False(t, box.Empty(), "caption rendered no visible pixels")

					center := float64(box.Min.Y+box.Max.Y) / (2 * float64(canvas[1]))
					low, high := 0.45, 0.65
					switch position.Name {
					case "portrait-upper":
						low, high = 0.15, 0.38
					case "portrait-center":
						low, high = 0.45, 0.55
					}
					assert.GreaterOrEqual(t, center, low, "placement %v", box)
					assert.LessOrEqual(t, center, high, "placement %v", box)
					assert.GreaterOrEqual(t, box.Min.X, canvas[0]/20, "placement %v", box)
					assert.LessOrEqual(t, box.Max.X, canvas[0]*19/20, "placement %v", box)
				})
			}
		}
	}
}
