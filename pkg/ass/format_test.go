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
