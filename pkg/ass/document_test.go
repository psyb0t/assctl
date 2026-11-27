package ass

import (
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const documentHeader = "[Script Info]\n[V4+ Styles]\nFormat: Name, Fontname\nStyle: Default,DejaVu Sans\n" +
	"[Events]\nFormat: Style, End, Start, Text\n"

func TestParseReorderedColumns(t *testing.T) {
	t.Parallel()

	source := "[sCrIpT InFo]\n; ignored: setting\nTitle: Clip\n[V4+ Styles]\nFormat: Fontname, Name\n" +
		"Style: DejaVu Sans,Default\n[Events]\nFormat: Style, End, Start, Text\n" +
		"Unknown: opaque\nComment: Default,0:00:01.00,0:00:00.00,ñ, 世界"
	document, err := Parse(strings.NewReader(source))
	require.NoError(t, err)
	require.NoError(t, document.Validate())

	contents := document.Contents()
	assert.Equal(t, map[string]string{"Title": "Clip"}, contents.Info)
	require.Len(t, contents.Styles, 1)
	assert.Equal(t, "Default", contents.Styles[0]["Name"])
	require.Len(t, contents.Events, 1)
	assert.Equal(t, "ñ, 世界", contents.Events[0]["Text"])

	contents.Events[0]["Text"] = "changed"
	assert.Equal(t, source, document.String())
	assert.Equal(t, "ñ, 世界", document.Contents().Events[0]["Text"])

	updated, err := document.Apply(Edit{Events: []EventEdit{
		{Index: 1, Fields: map[string]string{"Text": "a, b, c"}},
	}})
	require.NoError(t, err)
	assert.Equal(t, strings.Replace(source, "ñ, 世界", "a, b, c", 1), updated.String())
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

			document, err := Parse(strings.NewReader(tc.source))
			require.ErrorIs(t, err, ErrInvalidDocument)
			assert.Nil(t, document)
		})
	}
}

func TestParsePropagatesReadFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("read failed")
	document, err := Parse(iotest.ErrReader(failure))
	require.ErrorIs(t, err, failure)
	assert.Nil(t, document)
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

			document, err := Parse(strings.NewReader(tc.source))
			require.NoError(t, err)
			require.ErrorIs(t, document.Validate(), ErrInvalidDocument)
		})
	}
}

func TestApplyIsAtomicOnFailure(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		edit    Edit
		wantErr error
	}{
		{
			name:    "invalid info key",
			edit:    Edit{Info: map[string]string{"[Events]": "bad"}},
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "unknown style",
			edit:    Edit{Styles: []StyleEdit{{Name: "Missing"}}},
			wantErr: commerr.ErrNotFound,
		},
		{
			name: "duplicate style",
			edit: Edit{Styles: []StyleEdit{
				{Name: "Default", Fields: map[string]string{"Fontsize": "51"}}, {Name: "Default"},
			}},
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name: "unknown field",
			edit: Edit{Styles: []StyleEdit{
				{Name: "Default", Fields: map[string]string{"Missing": "anything"}},
			}},
			wantErr: commerr.ErrNotFound,
		},
		{
			name: "duplicate field ignoring case",
			edit: Edit{Styles: []StyleEdit{
				{Name: "Default", Fields: map[string]string{"Fontname": "One", "fontname": "Two"}},
			}},
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name: "comma in style field",
			edit: Edit{Styles: []StyleEdit{
				{Name: "Default", Fields: map[string]string{"Fontname": "Invalid,Font"}},
			}},
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "event zero",
			edit:    Edit{Events: []EventEdit{{Index: 0}}},
			wantErr: commerr.ErrNotFound,
		},
		{
			name:    "past last event",
			edit:    Edit{Events: []EventEdit{{Index: 2}}},
			wantErr: commerr.ErrNotFound,
		},
		{
			name: "duplicate event",
			edit: Edit{Events: []EventEdit{
				{Index: 1, Fields: map[string]string{"Text": "first"}}, {Index: 1},
			}},
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name: "invalid override",
			edit: Edit{Events: []EventEdit{
				{Index: 1, Override: &Override{FontName: new("bad\\n")}},
			}},
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name: "invalid UTF-8 replacement",
			edit: Edit{Events: []EventEdit{
				{Index: 1, Fields: map[string]string{"Text": "bad\xff"}},
			}},
			wantErr: ErrInvalidDocument,
		},
		{
			name: "unresolved renamed style",
			edit: Edit{Styles: []StyleEdit{
				{Name: "Default", Fields: map[string]string{"Name": "Renamed"}},
			}},
			wantErr: ErrInvalidDocument,
		},
	}

	original := fixture(t)
	document, err := Parse(strings.NewReader(original))
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.edit.Info == nil {
				tc.edit.Info = map[string]string{"Title": "Partial update"}
			}
			updated, err := document.Apply(tc.edit)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, updated)
			assert.Equal(t, original, document.String())
			assert.Equal(t, "Example", document.Contents().Info["Title"])
		})
	}
}

func TestAddSettingToHeaderWithoutNewline(t *testing.T) {
	t.Parallel()

	document, err := Parse(strings.NewReader("[Script Info]"))
	require.NoError(t, err)

	updated, err := document.Apply(Edit{Info: map[string]string{"Title": "New"}})
	require.NoError(t, err)
	assert.Equal(t, "[Script Info]\nTitle: New\n", updated.String())
}

func TestParseRejectsAmbiguousFormatColumns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		fields string
	}{
		{name: "duplicate ignoring case", fields: "Name, name"},
		{name: "trailing empty column", fields: "Name,"},
		{name: "leading empty column", fields: ",Name"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := "[Script Info]\n[V4+ Styles]\nFormat: " + tc.fields + "\nStyle: One,Two\n"
			document, err := Parse(strings.NewReader(input))
			require.ErrorIs(t, err, ErrInvalidDocument)
			assert.Nil(t, document)
		})
	}
}

func TestASSInputSizeBoundary(t *testing.T) {
	t.Parallel()

	const prefix = "[Script Info]\n;"
	input := prefix + strings.Repeat("x", maxInputBytes+1-len(prefix))

	t.Run("at the limit", func(t *testing.T) {
		t.Parallel()

		document, err := Parse(strings.NewReader(input[:maxInputBytes]))
		require.NoError(t, err)
		require.True(t, document.String() == input[:maxInputBytes], "parse changed the source bytes")
	})

	t.Run("over the limit", func(t *testing.T) {
		t.Parallel()

		document, err := Parse(strings.NewReader(input))
		require.ErrorIs(t, err, ErrInputTooLarge)
		assert.Nil(t, document)
	})
}

func TestTimestampRejectsDurationOverflow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		end  string
	}{
		{name: "hours past the duration range", end: "2562048:00:00.00"},
		{name: "last second past the duration range", end: "2562047:47:16.86"},
		{name: "enormous hours", end: "999999999999999999:00:00.00"},
	}

	source := fixture(t)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := strings.ReplaceAll(source, "0:00:02.00", tc.end)
			document, err := Parse(strings.NewReader(input))
			require.NoError(t, err)
			require.ErrorIs(t, document.Validate(), ErrInvalidDocument)
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
		document, err := Parse(strings.NewReader(source))
		if err != nil {
			return
		}
		assert.Equal(t, source, document.String())
	})
}
