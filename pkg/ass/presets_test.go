package ass

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTripAndTheme(t *testing.T) {
	t.Parallel()

	original := "\ufeff" + strings.ReplaceAll(fixture(t), "\n", "\r\n") +
		"\r\n[Aegisub Extradata]\r\nData: 0,example,opaque\r\n"
	document, err := Parse(strings.NewReader(original))
	require.NoError(t, err)
	require.NoError(t, document.Validate())
	require.Equal(t, original, document.String())

	look, err := LookupPreset("ivory-navy")
	require.NoError(t, err)

	themed, err := document.Theme(look, "Default")
	require.NoError(t, err)

	contents := themed.Contents()
	require.Len(t, contents.Styles, 1)
	fields := contents.Styles[0]
	assert.Equal(t, FontDejaVuSerif, fields["Fontname"])
	assert.Equal(t, "&H00E8F7FF", fields["PrimaryColour"])
	assert.Equal(t, "&H00FFCD88", fields["SecondaryColour"])
	assert.Equal(t, "&H18412A15", fields["OutlineColour"])
	assert.Equal(t, "3", fields["BorderStyle"])

	for _, preserved := range strings.Split(original, "\r\n") {
		if strings.HasPrefix(preserved, "Style:") {
			continue
		}
		assert.Contains(t, themed.String(), preserved, "theme lost an untouched line")
	}
	require.Len(t, contents.Events, 1)
	assert.Equal(t, "{\\q3}{\\k25}Hello, {\\k50}world", contents.Events[0]["Text"])
}

func TestCaptionLayersAndPresets(t *testing.T) {
	t.Parallel()

	const tagged = "{\\k10}one {\\k20}two\\Nthree four"

	testCases := []struct {
		name   string
		layers int
	}{
		{name: PresetBlackGold, layers: 2},
		{name: "ivory-navy", layers: 2},
		{name: "mint-plum", layers: 2},
		{name: "clean", layers: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			preset, err := LookupPreset(tc.name)
			require.NoError(t, err)

			script := preset.Captions(CaptionSpec{
				Title: "Timed text", Width: 1080, Height: 1920,
				Dialogues: []Dialogue{
					{Start: 0, End: time.Second, Text: "one two three four"},
					{Start: time.Second, End: 2 * time.Second, Text: tagged},
				},
				Uppercase: true, WordsPerLine: 2,
			})

			require.Len(t, script.Styles, tc.layers)
			require.Len(t, script.Events, 2*tc.layers)
			assert.Equal(t, "ONE TWO\\NTHREE FOUR", script.Events[tc.layers-1].Text)
			assert.Equal(t, tagged, script.Events[len(script.Events)-1].Text)

			_, err = script.MarshalText()
			require.NoError(t, err)
		})
	}
}

func TestSocialPresetsScaleAndPreserveDialogue(t *testing.T) {
	t.Parallel()

	const (
		lowerMargin = 0.4
		upperMargin = 0.18
	)

	testCases := []struct {
		name        string
		alignment   Alignment
		marginRatio float64
	}{
		{name: "tiktok-bold", alignment: AlignmentBottomCenter, marginRatio: lowerMargin},
		{name: "reels-bold", alignment: AlignmentBottomCenter, marginRatio: lowerMargin},
		{name: "shorts-impact", alignment: AlignmentBottomCenter, marginRatio: lowerMargin},
		{name: "social-center", alignment: AlignmentMiddleCenter, marginRatio: 0},
		{name: "social-box", alignment: AlignmentBottomCenter, marginRatio: lowerMargin},
		{name: "social-hook", alignment: AlignmentTopCenter, marginRatio: upperMargin},
		{name: "the-bold", alignment: AlignmentBottomCenter, marginRatio: lowerMargin},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			preset, err := LookupPreset(tc.name)
			require.NoError(t, err)

			for _, height := range []int{1920, 960} {
				t.Run(strconv.Itoa(height), func(t *testing.T) {
					t.Parallel()

					width := height * 9 / 16
					script := preset.Captions(CaptionSpec{
						Width: width, Height: height,
						Dialogues: []Dialogue{{Start: time.Second, End: 2 * time.Second, Text: "{\\k30}Hello, world"}},
					})

					style := script.Styles[0]
					assert.Equal(t, tc.alignment, style.Alignment)
					assert.Equal(t, int(float64(height)*tc.marginRatio), style.MarginV)
					assert.Equal(t, preset.Font, style.FontName)
					assert.Equal(t, float64(height)*preset.FontRatio, style.FontSize)
					assert.Equal(t, int(float64(width)*0.18), style.MarginL)
					assert.Equal(t, style.MarginL, style.MarginR)

					last := script.Events[len(script.Events)-1]
					assert.Equal(t, "{\\k30}Hello, world", last.Text)
					assert.Equal(t, time.Second, last.Start)
					assert.Equal(t, 2*time.Second, last.End)

					data, err := script.MarshalText()
					require.NoError(t, err)
					document, err := Parse(strings.NewReader(string(data)))
					require.NoError(t, err)

					updated, err := document.Theme(preset, "Text")
					require.NoError(t, err)
					assert.Equal(t, document.Contents().Events[0]["Text"], updated.Contents().Events[0]["Text"])
				})
			}
		})
	}
}

func TestSocialThemeMarginsAndExistingPositionOverrides(t *testing.T) {
	t.Parallel()

	document, err := Parse(strings.NewReader(fixture(t)))
	require.NoError(t, err)

	document, err = document.Apply(Edit{
		Styles: []StyleEdit{{Name: "Default", Fields: map[string]string{"MarginL": "23", "MarginR": "37"}}},
		Events: []EventEdit{{Index: 1, Fields: map[string]string{"MarginL": "99", "Text": "{\\pos(12,34)}Hello"}}},
	})
	require.NoError(t, err)

	testCases := []struct {
		name  string
		left  string
		right string
	}{
		{name: "clean", left: "23", right: "37"},
		{name: "tiktok-bold", left: "194", right: "194"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			preset, err := LookupPreset(tc.name)
			require.NoError(t, err)

			updated, err := document.Theme(preset, "Default")
			require.NoError(t, err)

			contents := updated.Contents()
			assert.Equal(t, tc.left, contents.Styles[0]["MarginL"])
			assert.Equal(t, tc.right, contents.Styles[0]["MarginR"])
			assert.Equal(t, "{\\pos(12,34)}Hello", contents.Events[0]["Text"])
			assert.Equal(t, "99", contents.Events[0]["MarginL"])
		})
	}
}

func TestLookupPreset(t *testing.T) {
	t.Parallel()

	t.Run("default preset is built in", func(t *testing.T) {
		t.Parallel()

		preset, err := LookupPreset(DefaultPreset)
		require.NoError(t, err)
		assert.Equal(t, PresetBlackGold, preset.Name)
	})

	t.Run("reports a missing name", func(t *testing.T) {
		t.Parallel()

		_, err := LookupPreset("missing")
		require.ErrorIs(t, err, commerr.ErrNotFound)
	})
}
