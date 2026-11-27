package ass_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/psyb0t/assctl/pkg/ass"
)

func fixture(t *testing.T) string {
	t.Helper()
	script := ass.NewScript("Example", 1080, 1920)
	script.AddStyle(ass.Style{
		Name: "Default", FontName: "DejaVu Sans", FontSize: 50,
		ScaleX: 100, ScaleY: 100, Alignment: ass.AlignmentBottomCenter,
		BorderStyle: ass.BorderStyleOutlineDropShadow, WrapStyle: ass.WrapStyleSmartBottom,
		Encoding: ass.EncodingDefault, Bold: true,
	})
	script.AddEvent(ass.Event{
		Start: 250 * time.Millisecond, End: 2 * time.Second, Style: "Default",
		Text: "{\\k25}Hello, {\\k50}world",
	})
	data, err := script.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRoundTripAndTheme(t *testing.T) {
	original := "\ufeff" + strings.ReplaceAll(fixture(t), "\n", "\r\n") +
		"\r\n[Aegisub Extradata]\r\nData: 0,example,opaque\r\n"
	document, err := ass.Parse(strings.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	if document.String() != original {
		t.Fatal("read changed original bytes")
	}

	look, err := ass.LookupPreset("ivory-navy")
	if err != nil {
		t.Fatal(err)
	}
	themed, err := document.Theme(look, "Default")
	if err != nil {
		t.Fatal(err)
	}
	fields := themed.Contents().Styles[0]
	if fields["Fontname"] != "DejaVu Serif" || fields["PrimaryColour"] != "&H00E8F7FF" ||
		fields["SecondaryColour"] != "&H00FFCD88" || fields["OutlineColour"] != "&H18412A15" ||
		fields["BorderStyle"] != "3" {
		t.Fatalf("theme style mismatch: %v", fields)
	}
	for _, preserved := range strings.Split(original, "\r\n") {
		if strings.HasPrefix(preserved, "Style:") {
			continue
		}
		if !strings.Contains(themed.String(), preserved) {
			t.Fatalf("theme lost untouched line %q", preserved)
		}
	}
	if themed.Contents().Events[0]["Text"] != "{\\q2}{\\k25}Hello, {\\k50}world" {
		t.Fatal("event tags or payload changed")
	}
}

func TestEditScriptStyleAndEvent(t *testing.T) {
	document, err := ass.Parse(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := document.Apply(ass.Edit{
		Info:   map[string]string{"Title": "New title", "WrapStyle": "1"},
		Styles: []ass.StyleEdit{{Name: "Default", Fields: map[string]string{"FontSize": "44"}}},
		Events: []ass.EventEdit{{
			Index: 1, Fields: map[string]string{"Start": "0:00:00.50", "Text": "Hi, world"},
			Override: &ass.Override{Position: &ass.Position{X: 12, Y: 34}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := updated.Contents()
	if result.Info["Title"] != "New title" || result.Info["WrapStyle"] != "1" ||
		result.Styles[0]["Fontsize"] != "44" || result.Events[0]["Start"] != "0:00:00.50" ||
		result.Events[0]["Text"] != "{\\pos(12,34)}Hi, world" {
		t.Fatalf("edits not reflected: %+v", result)
	}
	if _, err := document.Apply(ass.Edit{
		Events: []ass.EventEdit{{Index: 1, Fields: map[string]string{"Style": "Injected\r\nDialogue:"}}},
	}); !errors.Is(err, ass.ErrInvalidValue) {
		t.Fatalf("expected invalid field, got %v", err)
	}
}

func TestOverrideFormatAndColorValidation(t *testing.T) {
	red, err := ass.ParseRGB("#f00")
	if err != nil || red != ass.ColorRed {
		t.Fatalf("parse red: %q, %v", red, err)
	}
	if _, err := ass.ParseRGB("#ff"); !errors.Is(err, ass.ErrInvalidValue) {
		t.Fatalf("expected invalid RGB, got %v", err)
	}

	size := 25.0
	override := &ass.Override{
		PrimaryColor: &red,
		Movement:     &ass.Movement{X1: 1, Y1: 2, X2: 3, Y2: 4},
		Fade:         &ass.Fade{A1: 255, A2: 0, A3: 255, T2: time.Second, T3: 2 * time.Second},
		Transform: []ass.Transform{{
			End: 2 * time.Second, Accel: 1.5,
			Override: &ass.Override{FontSize: &size},
		}},
		Clip: &ass.Clip{X1: 0, Y1: 0, X2: 40, Y2: 40, Inverse: true},
	}
	tags, err := override.Tags()
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{
		"\\1c&H0000FF&", "\\move(1,2,3,4)", "\\fade(255,0,255,0,1000,2000,0)",
		"\\t(0,2000,1.5,\\fs25)", "\\iclip(0,0,40,40)",
	} {
		if !strings.Contains(tags, tag) {
			t.Fatalf("override missing %q in %q", tag, tags)
		}
	}

	recursive := &ass.Override{}
	recursive.Transform = []ass.Transform{{Override: recursive}}
	if _, err := recursive.Tags(); !errors.Is(err, ass.ErrInvalidValue) {
		t.Fatalf("expected cycle to stop at the nesting limit, got %v", err)
	}
}

func TestCaptionLayersAndPresets(t *testing.T) {
	const tagged = "{\\k10}one {\\k20}two\\Nthree four"
	for _, name := range []string{"black-gold", "ivory-navy", "mint-plum", "clean"} {
		t.Run(name, func(t *testing.T) {
			preset, err := ass.LookupPreset(name)
			if err != nil {
				t.Fatal(err)
			}
			script := preset.Captions(ass.CaptionSpec{
				Title: "Timed text", Width: 1080, Height: 1920,
				Dialogues: []ass.Dialogue{
					{Start: 0, End: time.Second, Text: "one two three four"},
					{Start: time.Second, End: 2 * time.Second, Text: tagged},
				},
				Uppercase: true, WordsPerLine: 2,
			})
			count := 2
			if name == "clean" {
				count = 1
			}
			if len(script.Events) != 2*count || len(script.Styles) != count ||
				script.Events[count-1].Text != "ONE TWO\\NTHREE FOUR" ||
				script.Events[len(script.Events)-1].Text != tagged {
				t.Fatalf("incorrect caption layers: %+v", script.Events)
			}
			if _, err := script.MarshalText(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func FuzzParsePreservesBytes(f *testing.F) {
	f.Add("[Script Info]\r\nTitle: Demo\r\n[Other]\r\n x\r\n")
	f.Add("[Script Info]\n[Events]\nFormat: Layer, Start, End, Style, Text\nDialogue: 0,0:00:00.00,0:00:01.00,A,Hi, there")
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 32<<10 {
			t.Skip()
		}
		document, err := ass.Parse(strings.NewReader(source))
		if err == nil && document.String() != source {
			t.Fatal("parse changed the source bytes")
		}
	})
}
