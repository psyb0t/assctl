package ass

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPositionsComposeWithEveryTheme(t *testing.T) {
	t.Parallel()

	canvases := [][2]int{{1080, 1920}, {540, 960}, {1920, 1080}, {1000, 1000}}

	for _, preset := range Presets() {
		t.Run(preset.Name, func(t *testing.T) {
			t.Parallel()

			for _, position := range TextPositions() {
				for _, canvas := range canvases {
					width, height := canvas[0], canvas[1]
					script := preset.Captions(CaptionSpec{
						Width: width, Height: height,
						Dialogues: []Dialogue{{End: time.Second, Text: "{\\k20}Test"}},
					})

					updated, err := script.WithTextPosition(position)
					require.NoError(t, err)
					require.Len(t, updated.Styles, len(script.Styles))

					for index, style := range updated.Styles {
						original := script.Styles[index]
						assert.Equal(t, original.FontName, style.FontName)
						assert.Equal(t, original.PrimaryColor, style.PrimaryColor)
						assert.Equal(t, position.Alignment, style.Alignment)
						assert.Equal(t, int(float64(width)*position.LeftMarginRatio), style.MarginL)
						assert.Equal(t, int(float64(width)*position.RightMarginRatio), style.MarginR)
						assert.Equal(t, int(float64(height)*position.MarginRatio), style.MarginV)
					}

					_, err = updated.MarshalText()
					require.NoError(t, err)
					assert.Equal(t, script.Events[0].Text, updated.Events[0].Text)
				}
			}
		})
	}
}

func TestTextPositionLookup(t *testing.T) {
	t.Parallel()

	t.Run("returns independent values", func(t *testing.T) {
		t.Parallel()

		positions := TextPositions()
		positions[0].Name = "changed"

		found, err := LookupTextPosition("portrait-upper")
		require.NoError(t, err)
		assert.Equal(t, "portrait-upper", found.Name)
		assert.Equal(t, AlignmentTopCenter, found.Alignment)
	})

	t.Run("reports a missing name", func(t *testing.T) {
		t.Parallel()

		_, err := LookupTextPosition("missing")
		require.ErrorIs(t, err, commerr.ErrNotFound)
	})
}

func TestPositionZeroMarginsAndPreservedEvents(t *testing.T) {
	t.Parallel()

	document, err := Parse(strings.NewReader(fixture(t)))
	require.NoError(t, err)

	custom := TextPosition{Alignment: AlignmentTopCenter}

	t.Run("zero ratios give zero margins", func(t *testing.T) {
		t.Parallel()

		updated, err := document.WithTextPosition(custom, "Default")
		require.NoError(t, err)

		contents := updated.Contents()
		assert.Equal(t, "0", contents.Styles[0]["MarginV"])
		assert.Equal(t, document.Contents().Events[0]["Text"], contents.Events[0]["Text"])
	})

	t.Run("missing style", func(t *testing.T) {
		t.Parallel()

		_, err := document.WithTextPosition(custom, "missing")
		require.ErrorIs(t, err, commerr.ErrNotFound)
	})
}

func TestInvalidTextPositions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		position TextPosition
	}{
		{name: "zero value", position: TextPosition{}},
		{name: "alignment above range", position: TextPosition{Alignment: 10}},
		{name: "negative margin", position: TextPosition{Alignment: 2, MarginRatio: -0.1}},
		{name: "margin of one", position: TextPosition{Alignment: 2, MarginRatio: 1}},
		{name: "NaN left margin", position: TextPosition{Alignment: 2, LeftMarginRatio: math.NaN()}},
		{name: "infinite right margin", position: TextPosition{Alignment: 2, RightMarginRatio: math.Inf(1)}},
		{name: "no horizontal space", position: TextPosition{Alignment: 2, LeftMarginRatio: 0.6, RightMarginRatio: 0.4}},
		{name: "ignored center margin", position: TextPosition{Alignment: 5, MarginRatio: 0.1}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Test", 1080, 1920)
			_, err := script.WithTextPosition(tc.position)
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
		})
	}

	t.Run("zero width script", func(t *testing.T) {
		t.Parallel()

		script := NewScript("Test", 0, 1920)
		_, err := script.WithTextPosition(TextPositions()[0])
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
	})
}

func TestPositionUsesLibassResolutionInference(t *testing.T) {
	t.Parallel()

	t.Run("infers missing dimensions", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name   string
			info   string
			width  int
			height int
		}{
			{name: "neither dimension", info: "", width: 384, height: 288},
			{name: "legacy wide width only", info: "PlayResX: 1280\n", width: 1280, height: 1024},
			{name: "legacy wide height only", info: "PlayResY: 1024\n", width: 1280, height: 1024},
			{name: "width only", info: "PlayResX: 1000\n", width: 1000, height: 750},
			{name: "height only", info: "PlayResY: 900\n", width: 1200, height: 900},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := fixture(t)
				source = strings.ReplaceAll(source, "PlayResX: 1080\n", "")
				source = strings.ReplaceAll(source, "PlayResY: 1920\n", tc.info)
				document, err := Parse(strings.NewReader(source))
				require.NoError(t, err)

				updated, err := document.WithTextPosition(TextPositions()[2], "")
				require.NoError(t, err)

				style := updated.Contents().Styles[0]
				assert.Equal(t, strconv.Itoa(int(float64(tc.height)*0.4)), style["MarginV"])
				assert.Equal(t, strconv.Itoa(int(float64(tc.width)*0.18)), style["MarginL"])
			})
		}
	})

	t.Run("rejects invalid dimensions", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			value string
		}{
			{name: "not a number", value: "bad"},
			{name: "zero", value: "0"},
			{name: "negative", value: "-1"},
			{name: "beyond 32 bits", value: "2147483648"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := strings.ReplaceAll(fixture(t), "PlayResX: 1080", "PlayResX: "+tc.value)
				document, err := Parse(strings.NewReader(source))
				require.NoError(t, err)

				_, err = document.WithTextPosition(TextPositions()[0], "")
				require.ErrorIs(t, err, commerr.ErrInvalidValue)
			})
		}
	})
}

func TestLiteralASSAndConcurrentEditsPreserveInput(t *testing.T) {
	t.Parallel()

	source := fixture(t)
	document, err := Parse(strings.NewReader(source))
	require.NoError(t, err)

	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			for _, position := range TextPositions() {
				updated, err := document.WithTextPosition(position, "")
				if !assert.NoError(t, err) {
					return
				}
				assert.NoError(t, updated.Validate())

				contents := updated.Contents()
				contents.Info["Title"] = "caller mutation"
			}
		})
	}
	workers.Wait()

	assert.Equal(t, source, document.String())
}
