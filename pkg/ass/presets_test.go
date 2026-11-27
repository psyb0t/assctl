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
		leftRatio   = 120.0 / 1080
		rightRatio  = 300.0 / 1080
	)

	testCases := []struct {
		name        string
		alignment   Alignment
		marginRatio float64
		leftRatio   float64
		rightRatio  float64
	}{
		{name: "tiktok-bold", alignment: AlignmentBottomCenter, marginRatio: lowerMargin, leftRatio: leftRatio, rightRatio: rightRatio},
		{name: "reels-bold", alignment: AlignmentBottomCenter, marginRatio: lowerMargin, leftRatio: leftRatio, rightRatio: rightRatio},
		{name: "shorts-impact", alignment: AlignmentBottomCenter, marginRatio: lowerMargin, leftRatio: leftRatio, rightRatio: rightRatio},
		{name: "social-center", alignment: AlignmentMiddleCenter, leftRatio: leftRatio, rightRatio: rightRatio},
		{name: "social-box", alignment: AlignmentBottomCenter, marginRatio: lowerMargin, leftRatio: leftRatio, rightRatio: rightRatio},
		{name: "social-hook", alignment: AlignmentTopCenter, marginRatio: upperMargin, leftRatio: upperMargin, rightRatio: upperMargin},
		{name: "the-bold", alignment: AlignmentBottomCenter, marginRatio: lowerMargin, leftRatio: leftRatio, rightRatio: rightRatio},
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
					assert.Equal(t, marginPixels(height, tc.marginRatio), style.MarginV)
					assert.Equal(t, preset.Font, style.FontName)
					assert.Equal(t, float64(height)*preset.FontRatio, style.FontSize)
					assert.Equal(t, marginPixels(width, tc.leftRatio), style.MarginL)
					assert.Equal(t, marginPixels(width, tc.rightRatio), style.MarginR)

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

	t.Run("exact margins on a 1080 by 1920 canvas", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			left     int
			right    int
			vertical int
		}{
			{name: "tiktok-bold", left: 120, right: 300, vertical: 768},
			{name: "social-center", left: 120, right: 300, vertical: 0},
			{name: "social-hook", left: 195, right: 195, vertical: 346},
		}
		for _, tc := range testCases {
			preset, err := LookupPreset(tc.name)
			require.NoError(t, err)

			style := preset.Captions(CaptionSpec{Width: 1080, Height: 1920}).Styles[0]
			assert.Equal(t, tc.left, style.MarginL, tc.name)
			assert.Equal(t, tc.right, style.MarginR, tc.name)
			assert.Equal(t, tc.vertical, style.MarginV, tc.name)
		}
	})
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

	rightOnly := Preset{
		Name: "right-only", Font: FontDejaVuSans, FontRatio: 0.04, BackgroundAlpha: AlphaTransparent,
		RightMarginRatio: 0.2, Alignment: AlignmentBottomCenter,
	}
	leftOnly := Preset{
		Name: "left-only", Font: FontDejaVuSans, FontRatio: 0.04, BackgroundAlpha: AlphaTransparent,
		SideMarginRatio: 0.1, Alignment: AlignmentBottomCenter,
	}
	both := leftOnly
	both.RightMarginRatio = 0.25

	testCases := []struct {
		name   string
		preset *Preset
		left   string
		right  string
	}{
		{name: "clean", left: "23", right: "37"},
		{name: "tiktok-bold", left: "120", right: "300"},
		{name: "social-hook", left: "195", right: "195"},
		{name: "social-center", left: "120", right: "300"},
		{name: "side ratio sets both sides", preset: &leftOnly, left: "108", right: "108"},
		{name: "right ratio alone writes both margins", preset: &rightOnly, left: "0", right: "216"},
		{name: "right ratio replaces the side ratio on the right", preset: &both, left: "108", right: "270"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var preset Preset
			if tc.preset != nil {
				preset = *tc.preset
			} else {
				var err error
				preset, err = LookupPreset(tc.name)
				require.NoError(t, err)
			}

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

func TestCaptionsEscapeAndWrapPlainText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		text      string
		uppercase bool
		words     int
		want      string
	}{
		{name: "empty", text: "", uppercase: true, words: 2, want: ""},
		{name: "plain text", text: "plain words", want: "plain words"},
		{name: "uppercase", text: "plain words", uppercase: true, want: "PLAIN WORDS"},
		{name: "brace is escaped", text: "a {b} c", want: `a \{{}b} c`},
		{name: "brace text is uppercased and wrapped", text: "a {b} c", uppercase: true, words: 2, want: `A \{{}B}\NC`},
		{name: "brace text is wrapped", text: "{one two three", words: 2, want: `\{{}one two\Nthree`},
		{name: "wrapped plain text", text: "one two three", words: 2, want: `one two\Nthree`},
		{name: "tags are kept", text: `{\an8}hi {there}`, uppercase: true, words: 1, want: `{\an8}hi {there}`},
		{name: "line break is kept", text: `one\Ntwo three`, uppercase: true, words: 1, want: `one\Ntwo three`},
		{name: "lone backslash keeps the text", text: `a\b {c}`, uppercase: true, want: `a\b {c}`},
	}

	for _, preset := range []string{PresetBlackGold, "clean"} {
		for _, tc := range testCases {
			t.Run(preset+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				look, err := LookupPreset(preset)
				require.NoError(t, err)
				script := look.Captions(CaptionSpec{
					Width: 1080, Height: 1920, Uppercase: tc.uppercase, WordsPerLine: tc.words,
					Dialogues: []Dialogue{{End: time.Second, Text: tc.text}},
				})

				require.NotEmpty(t, script.Events)
				for _, event := range script.Events {
					assert.Equal(t, tc.want, event.Text)
				}
				_, err = script.MarshalText()
				require.NoError(t, err)
			})
		}
	}
}

func TestCaptionsRightMarginRatio(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		side  float64
		right float64
		left  int
		want  int
	}{
		{name: "none", left: 0, want: 0},
		{name: "side ratio on both sides", side: 0.1, left: 108, want: 108},
		{name: "right ratio alone", right: 0.2, left: 0, want: 216},
		{name: "right ratio replaces the side ratio", side: 0.1, right: 0.2, left: 108, want: 216},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			preset := Preset{
				Name: "probe", Font: FontDejaVuSans, FontRatio: 0.04, BackgroundAlpha: AlphaTransparent,
				SideMarginRatio: tc.side, RightMarginRatio: tc.right, Alignment: AlignmentBottomCenter,
			}
			style := preset.Captions(CaptionSpec{Width: 1080, Height: 1920}).Styles[0]
			assert.Equal(t, tc.left, style.MarginL)
			assert.Equal(t, tc.want, style.MarginR)
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
