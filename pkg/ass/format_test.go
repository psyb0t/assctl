package ass_test

import (
	"bytes"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/psyb0t/assctl/pkg/ass"
)

func TestWriteScriptToFileAndStream(t *testing.T) {
	t.Parallel()

	script := ass.NewScript("Clip", 540, 960)
	script.AddStyle(ass.Style{
		Name: "Default", FontName: "DejaVu Sans",
		PrimaryColor: "ff0000", PrimaryAlpha: ass.AlphaHalf,
		WrapStyle: ass.WrapStyleSmartBottom,
	})
	script.AddEvent(ass.Event{
		Style: "Default", End: time.Second, Text: "a comma, and a line break\\Nhere",
		Override: &ass.Override{WrapStyle: ass.WrapStyleForced.Ptr()},
	})
	path := filepath.Join(t.TempDir(), "caption.ass")
	if err := script.WriteToFile(path); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := script.Encode(&output); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), saved) {
		t.Fatal("file and stream output differ")
	}
	document, err := ass.Parse(bytes.NewReader(saved))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	contents := document.Contents()
	if got := contents.Styles[0]["PrimaryColour"]; got != "&H80FF0000" {
		t.Fatalf("primary color = %q", got)
	}
	if got := contents.Events[0]["Text"]; got != "{\\q3}a comma, and a line break\\Nhere" {
		t.Fatalf("event text = %q", got)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestScriptPropagatesOutputFailures(t *testing.T) {
	t.Parallel()

	script := ass.NewScript("Clip", 540, 960)
	failure := errors.New("stream writer failed")
	if err := script.Encode(failingWriter{failure}); !errors.Is(err, failure) {
		t.Fatalf("Encode returned %v", err)
	}
	if err := script.WriteToFile(filepath.Join(t.TempDir(), "missing", "caption.ass")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("WriteToFile returned %v", err)
	}
}

func TestScriptRejectsInvalidFieldsBeforeWriting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		change func(*ass.Script)
		want   error
	}{
		{name: "zero width", change: func(s *ass.Script) { s.Resolution.Width = 0 }, want: ass.ErrInvalidValue},
		{name: "newline in title", change: func(s *ass.Script) { s.Title = "Clip\n[Events]" }, want: ass.ErrInvalidValue},
		{name: "invalid font", change: func(s *ass.Script) { s.Styles[0].FontName = "Bad,Font" }, want: ass.ErrInvalidValue},
		{name: "invalid color", change: func(s *ass.Script) { s.Styles[0].PrimaryColor = "&HXX" }, want: ass.ErrInvalidValue},
		{name: "invalid alpha", change: func(s *ass.Script) { s.Styles[0].PrimaryAlpha = "0G" }, want: ass.ErrInvalidValue},
		{name: "duplicate style", change: func(s *ass.Script) { s.AddStyle(s.Styles[0]) }, want: ass.ErrInvalidValue},
		{name: "missing style", change: func(s *ass.Script) { s.Events[0].Style = "Missing" }, want: ass.ErrNotFound},
		{name: "negative start", change: func(s *ass.Script) { s.Events[0].Start = -time.Second }, want: ass.ErrInvalidValue},
		{name: "sub-centisecond end", change: func(s *ass.Script) { s.Events[0].End = time.Millisecond }, want: ass.ErrInvalidValue},
		{name: "negative layer", change: func(s *ass.Script) { s.Events[0].Layer = -1 }, want: ass.ErrInvalidValue},
		{name: "newline in event", change: func(s *ass.Script) { s.Events[0].Text = "One\nTwo" }, want: ass.ErrInvalidValue},
		{name: "malformed override", change: func(s *ass.Script) {
			s.Events[0].Override = &ass.Override{FontName: new("Bad}Font")}
		}, want: ass.ErrInvalidValue},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := ass.NewScript("Clip", 540, 960)
			script.AddStyle(ass.Style{Name: "Default", FontName: "DejaVu Sans"})
			script.AddEvent(ass.Event{Style: "Default", End: time.Second, Text: "Line"})
			tc.change(script)

			var output bytes.Buffer
			if err := script.Encode(&output); !errors.Is(err, tc.want) || output.Len() != 0 {
				t.Fatalf("Encode returned %v and wrote %d bytes", err, output.Len())
			}
			path := filepath.Join(t.TempDir(), "existing.ass")
			const original = "untouched contents"
			if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := script.WriteToFile(path); !errors.Is(err, tc.want) {
				t.Fatalf("WriteToFile returned %v", err)
			}
			saved, err := os.ReadFile(path)
			if err != nil || string(saved) != original {
				t.Fatalf("existing file contains %q, error %v", saved, err)
			}
		})
	}
}

func TestThemeRejectsMissingStyleAndInvalidPalette(t *testing.T) {
	t.Parallel()

	source := strings.Replace(fixture(t), "PlayResY: 1920\n", "", 1)
	document, err := ass.Parse(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	preset, err := ass.LookupPreset("clean")
	if err != nil {
		t.Fatal(err)
	}
	themed, err := document.Theme(preset, "")
	if err != nil {
		t.Fatal(err)
	}
	fontSize, err := strconv.ParseFloat(themed.Contents().Styles[0]["Fontsize"], 64)
	if err != nil || math.Abs(fontSize-38.88) > 0.00001 {
		t.Fatalf("font size without PlayResY = %f, %v", fontSize, err)
	}
	if _, err := document.Theme(preset, "Missing"); !errors.Is(err, ass.ErrNotFound) {
		t.Fatalf("Theme returned %v for missing style", err)
	}
	preset.Text = "bad color"
	if _, err := document.Theme(preset, "Default"); !errors.Is(err, ass.ErrInvalidValue) {
		t.Fatalf("Theme returned %v for invalid palette", err)
	}
	if document.String() != source {
		t.Fatal("theming changed the original document")
	}
}
