package ass

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) {
	return len(data) / 2, nil
}

func TestWriteScriptToFileAndStream(t *testing.T) {
	t.Parallel()

	script := NewScript("Clip", 540, 960)
	script.AddStyle(Style{
		Name: "Default", FontName: FontDejaVuSans,
		PrimaryColor: "ff0000", PrimaryAlpha: AlphaHalf,
		WrapStyle: WrapStyleSmartBottom,
	})
	script.AddEvent(Event{
		Style: "Default", End: time.Second, Text: "a comma, and a line break\\Nhere",
		Override: &Override{WrapStyle: WrapStyleForced.Ptr()},
	})
	path := filepath.Join(t.TempDir(), "caption.ass")
	require.NoError(t, script.WriteToFile(path))

	saved, err := os.ReadFile(path)
	require.NoError(t, err)

	var output bytes.Buffer
	require.NoError(t, script.Encode(&output))
	assert.Equal(t, saved, output.Bytes())

	document, err := Parse(bytes.NewReader(saved))
	require.NoError(t, err)
	require.NoError(t, document.Validate())

	contents := document.Contents()
	require.Len(t, contents.Styles, 1)
	assert.Equal(t, "&H80FF0000", contents.Styles[0]["PrimaryColour"])
	require.Len(t, contents.Events, 1)
	assert.Equal(t, "{\\q2}a comma, and a line break\\Nhere", contents.Events[0]["Text"])
}

func TestScriptPropagatesOutputFailures(t *testing.T) {
	t.Parallel()

	script := NewScript("Clip", 540, 960)

	t.Run("stream writer error", func(t *testing.T) {
		t.Parallel()

		failure := errors.New("stream writer failed")
		require.ErrorIs(t, script.Encode(failingWriter{failure}), failure)
	})

	t.Run("missing directory", func(t *testing.T) {
		t.Parallel()

		err := script.WriteToFile(filepath.Join(t.TempDir(), "missing", "caption.ass"))
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("partial write", func(t *testing.T) {
		t.Parallel()

		require.ErrorIs(t, script.Encode(shortWriter{}), io.ErrShortWrite)
	})
}

func TestScriptRejectsInvalidFieldsBeforeWriting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		change  func(*Script)
		wantErr error
	}{
		{
			name:    "zero width",
			change:  func(s *Script) { s.Resolution.Width = 0 },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "newline in title",
			change:  func(s *Script) { s.Title = "Clip\n[Events]" },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "invalid font",
			change:  func(s *Script) { s.Styles[0].FontName = "Bad,Font" },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "invalid color",
			change:  func(s *Script) { s.Styles[0].PrimaryColor = "&HXX" },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "invalid alpha",
			change:  func(s *Script) { s.Styles[0].PrimaryAlpha = "0G" },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "duplicate style",
			change:  func(s *Script) { s.AddStyle(s.Styles[0]) },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "missing style",
			change:  func(s *Script) { s.Events[0].Style = "Missing" },
			wantErr: commerr.ErrNotFound,
		},
		{
			name:    "negative start",
			change:  func(s *Script) { s.Events[0].Start = -time.Second },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "sub-centisecond end",
			change:  func(s *Script) { s.Events[0].End = time.Millisecond },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "negative layer",
			change:  func(s *Script) { s.Events[0].Layer = -1 },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "newline in event",
			change:  func(s *Script) { s.Events[0].Text = "One\nTwo" },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name: "malformed override",
			change: func(s *Script) {
				s.Events[0].Override = &Override{FontName: new("Bad}Font")}
			},
			wantErr: commerr.ErrInvalidValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Clip", 540, 960)
			script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans})
			script.AddEvent(Event{Style: "Default", End: time.Second, Text: "Line"})
			tc.change(script)

			var output bytes.Buffer
			require.ErrorIs(t, script.Encode(&output), tc.wantErr)
			assert.Zero(t, output.Len())

			path := filepath.Join(t.TempDir(), "existing.ass")
			const original = "untouched contents"
			require.NoError(t, os.WriteFile(path, []byte(original), 0o600))
			require.ErrorIs(t, script.WriteToFile(path), tc.wantErr)

			saved, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, original, string(saved))
		})
	}
}

func TestThemeRejectsMissingStyleAndInvalidPalette(t *testing.T) {
	t.Parallel()

	source := strings.Replace(fixture(t), "PlayResY: 1920\n", "", 1)
	document, err := Parse(strings.NewReader(source))
	require.NoError(t, err)

	preset, err := LookupPreset("clean")
	require.NoError(t, err)

	themed, err := document.Theme(preset, "")
	require.NoError(t, err)
	require.Len(t, themed.Contents().Styles, 1)

	fontSize, err := strconv.ParseFloat(themed.Contents().Styles[0]["Fontsize"], 64)
	require.NoError(t, err)
	assert.InDelta(t, 29.16, fontSize, 0.00001)

	_, err = document.Theme(preset, "Missing")
	require.ErrorIs(t, err, commerr.ErrNotFound)

	preset.Text = "bad color"
	_, err = document.Theme(preset, "Default")
	require.ErrorIs(t, err, commerr.ErrInvalidValue)

	assert.Equal(t, source, document.String())
}

func FuzzGeneratedOverrideRoundTrip(f *testing.F) {
	f.Add("hello, world", 0, 100)
	f.Add("\xff", 0, 100)
	f.Add(`{\k20}café\Nworld`, 5, 200)
	f.Fuzz(func(t *testing.T, text string, margin, end int) {
		if len(text) > 4096 {
			return
		}
		script := drawingScript(nil)
		script.Events[0].Text = text
		script.Events[0].MarginV = margin
		script.Events[0].End = time.Duration(end) * time.Millisecond
		data, err := script.MarshalText()
		if err != nil {
			return
		}
		document, err := Parse(bytes.NewReader(data))
		require.NoError(t, err)
		require.NoError(t, document.Validate())
		assert.Equal(t, string(data), document.String())
	})
}

func TestTypedWriterRejectsInvalidUTF8AndNegativeStyleGeometry(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		change func(*Script)
	}{
		{name: "title", change: func(s *Script) { s.Title = "\xff" }},
		{name: "event text", change: func(s *Script) { s.Events[0].Text = "\xff" }},
		{name: "event name", change: func(s *Script) { s.Events[0].Name = "\xff" }},
		{name: "font name", change: func(s *Script) { s.Styles[0].FontName = "\xff" }},
		{name: "info value", change: func(s *Script) { s.Info = map[string]string{"Comment": "\xff"} }},
		{name: "negative font size", change: func(s *Script) { s.Styles[0].FontSize = -1 }},
		{name: "negative scale x", change: func(s *Script) { s.Styles[0].ScaleX = -1 }},
		{name: "negative scale y", change: func(s *Script) { s.Styles[0].ScaleY = -1 }},
		{name: "negative outline", change: func(s *Script) { s.Styles[0].Outline = -1 }},
		{name: "negative shadow", change: func(s *Script) { s.Styles[0].Shadow = -1 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := drawingScript(nil)
			tc.change(&script)

			data, err := script.MarshalText()
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Empty(t, data)
		})
	}
}

func TestMaximumDurationRoundTrip(t *testing.T) {
	t.Parallel()

	script := drawingScript(nil)
	script.Events[0].End = time.Duration(math.MaxInt64)
	data, err := script.MarshalText()
	require.NoError(t, err)

	document, err := Parse(bytes.NewReader(data))
	require.NoError(t, err)
	require.NoError(t, document.Validate())
}

func TestScriptCommentEventsAndTypedWrapModes(t *testing.T) {
	t.Parallel()

	script := NewScript("Metadata", 1080, 1920)
	script.Info = map[string]string{"WrapStyle": "2", "YCbCr Matrix": "TV.709", "Collisions": "Normal"}
	script.AddStyle(Style{Name: "Text", FontName: FontDejaVuSans})
	script.AddEvent(Event{Comment: true, Style: "Text", End: time.Second, Text: "Comment only"})
	data, err := script.MarshalText()
	require.NoError(t, err)

	document, err := Parse(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, "TV.709", document.Contents().Info["YCbCr Matrix"])
	assert.Contains(t, string(data), "Comment: ")

	assert.Equal(t, WrapStyle(2), WrapStyleNoWrap)
	assert.Equal(t, WrapStyle(2), WrapStyleForced)
	assert.Equal(t, WrapStyle(3), WrapStyleSmartBottom)
}

func TestScriptInfoHeaderOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		info map[string]string
		want string
	}{
		{
			name: "no custom headers",
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\n" +
				"ScaledBorderAndShadow: yes\nYCbCr Matrix: None\nPlayResX: 1080\nPlayResY: 1920",
		},
		{
			name: "custom headers sorted by lowercase key",
			info: map[string]string{"WrapStyle": "2", "Collisions": "Normal", "audio file": "a.wav"},
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\naudio file: a.wav\nCollisions: Normal\n" +
				"ScaledBorderAndShadow: yes\nWrapStyle: 2\nYCbCr Matrix: None\nPlayResX: 1080\nPlayResY: 1920",
		},
		{
			name: "custom headers replace defaults of the same name",
			info: map[string]string{"scaledborderandshadow": "no", "YCbCr Matrix": "TV.709"},
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\n" +
				"scaledborderandshadow: no\nYCbCr Matrix: TV.709\nPlayResX: 1080\nPlayResY: 1920",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Metadata", 1080, 1920)
			script.Info = tc.info
			data, err := script.MarshalText()
			require.NoError(t, err)

			section, _, found := strings.Cut(string(data), "\n\n")
			require.True(t, found)
			assert.Equal(t, tc.want, section)
		})
	}
}

func TestScriptRejectsInvalidInfoHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		info map[string]string
	}{
		{name: "title has a typed field", info: map[string]string{"Title": "duplicate"}},
		{name: "script type has a typed field", info: map[string]string{"scripttype": "v4.00"}},
		{name: "width has a typed field", info: map[string]string{"PlayResX": "1"}},
		{name: "height has a typed field", info: map[string]string{"playresy": "1"}},
		{name: "colon in key", info: map[string]string{"bad:key": "value"}},
		{name: "leading semicolon in key", info: map[string]string{";Key": "value"}},
		{name: "inner semicolon in key", info: map[string]string{"Key;more": "value"}},
		{name: "bracket in key", info: map[string]string{"[Key]": "value"}},
		{name: "empty key", info: map[string]string{"": "value"}},
		{name: "padded key", info: map[string]string{" Key": "value"}},
		{name: "newline in value", info: map[string]string{"Key": "bad\nvalue"}},
		{name: "keys differing only by case", info: map[string]string{"Key": "one", "key": "two"}},
		{name: "kerning has a typed field", info: map[string]string{"Kerning": "yes"}},
		{name: "kerning in another case", info: map[string]string{"kerning": "no"}},
		{name: "language has a typed field", info: map[string]string{"Language": "en"}},
		{name: "language in another case", info: map[string]string{"LANGUAGE": "en"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Metadata", 1080, 1920)
			script.Info = tc.info

			data, err := script.MarshalText()
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Empty(t, data)
		})
	}
}

func fixture(t *testing.T) string {
	t.Helper()

	script := NewScript("Example", 1080, 1920)
	script.AddStyle(Style{
		Name: "Default", FontName: FontDejaVuSans, FontSize: 50,
		ScaleX: 100, ScaleY: 100, Alignment: AlignmentBottomCenter,
		BorderStyle: BorderStyleOutlineDropShadow, WrapStyle: WrapStyleSmartBottom,
		Encoding: EncodingDefault, Bold: true,
	})
	script.AddEvent(Event{
		Start: 250 * time.Millisecond, End: 2 * time.Second, Style: "Default",
		Text: "{\\k25}Hello, {\\k50}world",
	})
	data, err := script.MarshalText()
	require.NoError(t, err)

	return string(data)
}

func TestScriptKerningAndLanguageHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		kerning  bool
		language string
		info     map[string]string
		want     string
	}{
		{
			name: "neither",
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\n" +
				"ScaledBorderAndShadow: yes\nYCbCr Matrix: None\nPlayResX: 1080\nPlayResY: 1920",
		},
		{
			name: "kerning", kerning: true,
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\nKerning: yes\n" +
				"ScaledBorderAndShadow: yes\nYCbCr Matrix: None\nPlayResX: 1080\nPlayResY: 1920",
		},
		{
			name: "language", language: "ar",
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\nLanguage: ar\n" +
				"ScaledBorderAndShadow: yes\nYCbCr Matrix: None\nPlayResX: 1080\nPlayResY: 1920",
		},
		{
			name: "both before the custom headers", kerning: true, language: "ja", info: map[string]string{"Collisions": "Normal"},
			want: "[Script Info]\nTitle: Metadata\nScriptType: v4.00+\nKerning: yes\nLanguage: ja\nCollisions: Normal\n" +
				"ScaledBorderAndShadow: yes\nYCbCr Matrix: None\nPlayResX: 1080\nPlayResY: 1920",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Metadata", 1080, 1920)
			script.Kerning, script.Language, script.Info = tc.kerning, tc.language, tc.info
			data, err := script.MarshalText()
			require.NoError(t, err)

			section, _, found := strings.Cut(string(data), "\n\n")
			require.True(t, found)
			assert.Equal(t, tc.want, section)

			document, err := Parse(bytes.NewReader(data))
			require.NoError(t, err)
			assert.Equal(t, tc.kerning, document.Contents().Info["Kerning"] == "yes")
			assert.Equal(t, tc.language, document.Contents().Info["Language"])
		})
	}
}

func TestScriptRejectsInvalidLanguage(t *testing.T) {
	t.Parallel()

	for _, language := range []string{
		"EN", "En", "e", "eng", "e1", "1e", "en-US", " en", "en ", "e n", "en\n", "\nen", "é", "ée", "ab\x00",
	} {
		t.Run(language, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Metadata", 1080, 1920)
			script.Language = language
			data, err := script.MarshalText()
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Empty(t, data)
		})
	}
}

func TestScriptWritesEmbeddedFontsBetweenStylesAndEvents(t *testing.T) {
	t.Parallel()

	script := NewScript("Fonts", 640, 360)
	script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans})
	script.AddEvent(Event{Style: "Default", End: time.Second, Text: "Line"})

	t.Run("no fonts writes no section", func(t *testing.T) {
		t.Parallel()

		data, err := script.MarshalText()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "[Fonts]")
	})

	t.Run("fonts are written in order", func(t *testing.T) {
		t.Parallel()

		withFonts := *script
		withFonts.Fonts = []EmbeddedFont{
			{Name: "b.ttf", Data: []byte{0, 0, 0}},
			{Name: "a.otf", Data: []byte{0xff, 0xff, 0xff, 0xff}},
		}
		data, err := withFonts.MarshalText()
		require.NoError(t, err)

		text := string(data)
		assert.Contains(t, text, "\n\n[Fonts]\nfontname: b.ttf\n!!!!\nfontname: a.otf\n`````Q\n\n[Events]\n")
		assert.Less(t, strings.Index(text, "[V4+ Styles]"), strings.Index(text, "[Fonts]"))
		assert.Less(t, strings.Index(text, "[Fonts]"), strings.Index(text, "[Events]"))

		document, err := Parse(strings.NewReader(text))
		require.NoError(t, err)
		require.NoError(t, document.Validate())
		assert.Equal(t, text, document.String())
		assert.Equal(t, map[string][]byte{"b.ttf": {0, 0, 0}, "a.otf": {0xff, 0xff, 0xff, 0xff}},
			embeddedFontsOf(t, text))
		assert.Len(t, document.Contents().Styles, 1)
		assert.Len(t, document.Contents().Events, 1)
	})

	t.Run("a font over several lines", func(t *testing.T) {
		t.Parallel()

		data := make([]byte, 500)
		for index := range data {
			data[index] = byte(index * 7)
		}
		withFonts := *script
		withFonts.Fonts = []EmbeddedFont{{Name: "big.ttf", Data: data}}
		text, err := withFonts.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, map[string][]byte{"big.ttf": data}, embeddedFontsOf(t, string(text)))
	})

	t.Run("invalid fonts are rejected before writing", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			fonts []EmbeddedFont
		}{
			{name: "empty name", fonts: []EmbeddedFont{{Data: []byte{1}}}},
			{name: "empty data", fonts: []EmbeddedFont{{Name: "a.ttf"}}},
			{name: "multi-line name", fonts: []EmbeddedFont{{Name: "a\nb.ttf", Data: []byte{1}}}},
			{name: "padded name", fonts: []EmbeddedFont{{Name: "a.ttf ", Data: []byte{1}}}},
			{name: "duplicate name", fonts: []EmbeddedFont{{Name: "a.ttf", Data: []byte{1}}, {Name: "A.TTF", Data: []byte{2}}}},
		}
		for _, tc := range testCases {
			rejected := *script
			rejected.Fonts = tc.fonts
			data, err := rejected.MarshalText()
			require.ErrorIs(t, err, commerr.ErrInvalidValue, tc.name)
			assert.Empty(t, data, tc.name)
		}
	})
}

func TestStyleBorderStyleAndEncodingRanges(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		border       BorderStyle
		encoding     Encoding
		wantBorder   string
		wantEncoding string
		wantErr      error
	}{
		{name: "unset border", border: 0, encoding: EncodingDefault, wantBorder: "0", wantEncoding: "1"},
		{name: "outline and shadow", border: BorderStyleOutlineDropShadow, wantBorder: "1", wantEncoding: "0"},
		{name: "opaque box", border: BorderStyleOpaqueBox, wantBorder: "3", wantEncoding: "0"},
		{name: "event box", border: BorderStyleEventBox, wantBorder: "4", wantEncoding: "0"},
		{name: "automatic direction", encoding: EncodingAuto, wantBorder: "0", wantEncoding: "-1"},
		{name: "largest named encoding", encoding: EncodingOEM, wantBorder: "0", wantEncoding: "255"},
		{name: "border style 2 is not a style", border: 2, wantErr: commerr.ErrInvalidValue},
		{name: "border style 5", border: 5, wantErr: commerr.ErrInvalidValue},
		{name: "negative border style", border: -1, wantErr: commerr.ErrInvalidValue},
		{name: "encoding below automatic", encoding: -2, wantErr: commerr.ErrInvalidValue},
		{name: "very negative encoding", encoding: -1000, wantErr: commerr.ErrInvalidValue},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := NewScript("Border", 640, 360)
			script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans, BorderStyle: tc.border, Encoding: tc.encoding})
			data, err := script.MarshalText()
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr != nil {
				assert.Empty(t, data)

				return
			}

			document, err := Parse(bytes.NewReader(data))
			require.NoError(t, err)
			style := document.Contents().Styles[0]
			assert.Equal(t, tc.wantBorder, style["BorderStyle"])
			assert.Equal(t, tc.wantEncoding, style["Encoding"])
		})
	}

	assert.Equal(t, BorderStyle(4), BorderStyleEventBox)
	assert.Equal(t, Encoding(-1), EncodingAuto)
}

func TestNumbersRoundToThreeDecimals(t *testing.T) {
	t.Parallel()

	height, ratio := 1920.0, 0.045
	product := height * ratio
	require.NotEqual(t, "86.4", strconv.FormatFloat(product, 'f', -1, 64),
		"the product must carry floating point noise for this test to mean anything")

	testCases := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "ratio product", value: product, want: "86.4"},
		{name: "sum noise", value: 0.1 + 0.2, want: "0.3"},
		{name: "whole number", value: 50, want: "50"},
		{name: "one decimal", value: 2.5, want: "2.5"},
		{name: "three decimals kept", value: 1.234, want: "1.234"},
		{name: "rounded down", value: 1.2341, want: "1.234"},
		{name: "rounded up", value: 1.2346, want: "1.235"},
		{name: "tiny value", value: 0.0004, want: "0"},
		{name: "negative", value: -2.5004, want: "-2.5"},
		{name: "zero", value: 0, want: "0"},
		{name: "tiny negative value", value: -0.0004, want: "0"},
		{name: "negative zero", value: math.Copysign(0, -1), want: "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, number(tc.value))
		})
	}

	t.Run("style and override output", func(t *testing.T) {
		t.Parallel()

		script := NewScript("Numbers", 1080, 1920)
		script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans, FontSize: product})
		data, err := script.MarshalText()
		require.NoError(t, err)
		document, err := Parse(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "86.4", document.Contents().Styles[0]["Fontsize"])

		tags, err := (&Override{FontSize: &product, Position: &Position{X: product, Y: 0.1 + 0.2}}).Tags()
		require.NoError(t, err)
		assert.Equal(t, `{\pos(86.4,0.3)\fs86.4}`, tags)
	})
}
