package ass_test

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/psyb0t/assctl/pkg/ass"
)

const documentHeader = "[Script Info]\n[V4+ Styles]\nFormat: Name, Fontname\nStyle: Default,DejaVu Sans\n" +
	"[Events]\nFormat: Style, End, Start, Text\n"

func TestParseReorderedColumns(t *testing.T) {
	t.Parallel()

	source := "[sCrIpT InFo]\n; ignored: setting\nTitle: Clip\n[V4+ Styles]\nFormat: Fontname, Name\n" +
		"Style: DejaVu Sans,Default\n[Events]\nFormat: Style, End, Start, Text\n" +
		"Unknown: opaque\nComment: Default,0:00:01.00,0:00:00.00,ñ, 世界"
	document, err := ass.Parse(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	contents := document.Contents()
	if contents.Info["Title"] != "Clip" || len(contents.Info) != 1 ||
		len(contents.Styles) != 1 || contents.Styles[0]["Name"] != "Default" ||
		len(contents.Events) != 1 || contents.Events[0]["Text"] != "ñ, 世界" {
		t.Fatalf("unexpected contents: %+v", contents)
	}
	contents.Events[0]["Text"] = "changed"
	if document.String() != source || document.Contents().Events[0]["Text"] != "ñ, 世界" {
		t.Fatal("inspection changed the document")
	}
	updated, err := document.Apply(ass.Edit{Events: []ass.EventEdit{
		{Index: 1, Fields: map[string]string{"Text": "a, b, c"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(source, "ñ, 世界", "a, b, c", 1); updated.String() != want {
		t.Fatalf("unexpected edited comment: %q", updated.String())
	}
}

func TestParseRejectsMalformedFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
	}{
		{name: "missing script section", source: "[Events]\n"},
		{name: "invalid UTF-8", source: "[Script Info]\nTitle: \xff\n"},
		{name: "NUL", source: "[Script Info]\nTitle: a\x00b\n"},
		{name: "style before format", source: "[Script Info]\n[V4+ Styles]\nStyle: Default\n"},
		{name: "event before format", source: "[Script Info]\n[Events]\nDialogue: Default,hello\n"},
		{name: "text before another field", source: "[Script Info]\n[Events]\nFormat: Text, Style\nDialogue: hello,Default\n"},
		{name: "truncated event", source: documentHeader + "Dialogue: Default,0:00:01.00\n"},
		{name: "extra style field", source: "[Script Info]\n[V4+ Styles]\nFormat: Name\nStyle: Default,extra\n"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			document, err := ass.Parse(strings.NewReader(tc.source))
			if document != nil || !errors.Is(err, ass.ErrInvalidDocument) {
				t.Fatalf("Parse returned %v, %v", document, err)
			}
		})
	}
}

func TestParsePropagatesReadFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("read failed")
	document, err := ass.Parse(iotest.ErrReader(failure))
	if document != nil || !errors.Is(err, failure) {
		t.Fatalf("Parse returned %v, %v", document, err)
	}
}

func TestValidateRejectsBrokenReferencesAndTiming(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
	}{
		{name: "empty style name", source: strings.Replace(documentHeader, "Style: Default,", "Style: ,", 1)},
		{name: "missing name column", source: strings.Replace(documentHeader, "Format: Name,", "Format: Label,", 1)},
		{name: "duplicate style", source: strings.Replace(documentHeader, "[Events]", "Style: Default,Other\n[Events]", 1)},
		{name: "missing referenced style", source: documentHeader + "Dialogue: Other,0:00:01.00,0:00:00.00,hello\n"},
		{name: "end before start", source: documentHeader + "Dialogue: Default,0:00:00.00,0:00:01.00,hello\n"},
		{name: "equal endpoints", source: documentHeader + "Dialogue: Default,0:00:01.00,0:00:01.00,hello\n"},
		{name: "minutes out of range", source: documentHeader + "Dialogue: Default,0:60:00.00,0:00:00.00,hello\n"},
		{name: "malformed centiseconds", source: documentHeader + "Dialogue: Default,0:00:01.1,0:00:00.00,hello\n"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			document, err := ass.Parse(strings.NewReader(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if err := document.Validate(); !errors.Is(err, ass.ErrInvalidDocument) {
				t.Fatalf("Validate returned %v", err)
			}
		})
	}
}

func TestApplyIsAtomicOnFailure(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		edit ass.Edit
		want error
	}{
		{name: "invalid info key", edit: ass.Edit{Info: map[string]string{"[Events]": "bad"}}, want: ass.ErrInvalidValue},
		{name: "unknown style", edit: ass.Edit{Styles: []ass.StyleEdit{{Name: "Missing"}}}, want: ass.ErrNotFound},
		{name: "duplicate style", edit: ass.Edit{Styles: []ass.StyleEdit{
			{Name: "Default", Fields: map[string]string{"Fontsize": "51"}}, {Name: "Default"},
		}}, want: ass.ErrInvalidValue},
		{name: "unknown field", edit: ass.Edit{Styles: []ass.StyleEdit{
			{Name: "Default", Fields: map[string]string{"Missing": "anything"}},
		}}, want: ass.ErrNotFound},
		{name: "duplicate field ignoring case", edit: ass.Edit{Styles: []ass.StyleEdit{
			{Name: "Default", Fields: map[string]string{"Fontname": "One", "fontname": "Two"}},
		}}, want: ass.ErrInvalidValue},
		{name: "comma in style field", edit: ass.Edit{Styles: []ass.StyleEdit{
			{Name: "Default", Fields: map[string]string{"Fontname": "Invalid,Font"}},
		}}, want: ass.ErrInvalidValue},
		{name: "event zero", edit: ass.Edit{Events: []ass.EventEdit{{Index: 0}}}, want: ass.ErrNotFound},
		{name: "past last event", edit: ass.Edit{Events: []ass.EventEdit{{Index: 2}}}, want: ass.ErrNotFound},
		{name: "duplicate event", edit: ass.Edit{Events: []ass.EventEdit{
			{Index: 1, Fields: map[string]string{"Text": "first"}}, {Index: 1},
		}}, want: ass.ErrInvalidValue},
		{name: "invalid override", edit: ass.Edit{Events: []ass.EventEdit{
			{Index: 1, Override: &ass.Override{FontName: new("bad\\n")}},
		}}, want: ass.ErrInvalidValue},
		{name: "invalid UTF-8 replacement", edit: ass.Edit{Events: []ass.EventEdit{
			{Index: 1, Fields: map[string]string{"Text": "bad\xff"}},
		}}, want: ass.ErrInvalidDocument},
		{name: "unresolved renamed style", edit: ass.Edit{Styles: []ass.StyleEdit{
			{Name: "Default", Fields: map[string]string{"Name": "Renamed"}},
		}}, want: ass.ErrInvalidDocument},
	}
	original := fixture(t)
	document, err := ass.Parse(strings.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.edit.Info == nil {
				tc.edit.Info = map[string]string{"Title": "Partial update"}
			}
			updated, err := document.Apply(tc.edit)
			if updated != nil || !errors.Is(err, tc.want) {
				t.Fatalf("Apply returned %v, %v, want %v", updated, err, tc.want)
			}
			if document.String() != original || document.Contents().Info["Title"] != "Example" {
				t.Fatal("failed edit changed the source")
			}
		})
	}
}

func TestAddSettingToHeaderWithoutNewline(t *testing.T) {
	t.Parallel()

	document, err := ass.Parse(strings.NewReader("[Script Info]"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := document.Apply(ass.Edit{Info: map[string]string{"Title": "New"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "[Script Info]\nTitle: New\n"; updated.String() != want {
		t.Fatalf("edited document = %q, want %q", updated.String(), want)
	}
}
