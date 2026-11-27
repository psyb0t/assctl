package ass

import (
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseDocument(t *testing.T, source string) *Document {
	t.Helper()

	document, err := Parse(strings.NewReader(source))
	require.NoError(t, err)

	return document
}

// zeroFont encodes to the single data line "!!!!".
func zeroFont(name string) EmbeddedFont {
	return EmbeddedFont{Name: name, Data: []byte{0, 0, 0}}
}

func TestEmbedFontsLayout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		fonts  []EmbeddedFont
		want   string
	}{
		{
			name:   "adds a section at the end",
			source: "[Script Info]\nTitle: T\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\nTitle: T\n\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "ends the last line when the file has no final newline",
			source: "[Script Info]\nTitle: T",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\nTitle: T\n\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "keeps a trailing blank line",
			source: "[Script Info]\nTitle: T\n\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\nTitle: T\n\n\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "uses CRLF for a CRLF file",
			source: "[Script Info]\r\nTitle: T\r\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\r\nTitle: T\r\n\r\n[Fonts]\r\nfontname: a.ttf\r\n!!!!\r\n",
		},
		{
			name:   "uses CRLF when the last line has no ending",
			source: "[Script Info]\r\nTitle: T",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\r\nTitle: T\r\n\r\n[Fonts]\r\nfontname: a.ttf\r\n!!!!\r\n",
		},
		{
			name:   "uses LF for a header-only file",
			source: "[Script Info]",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "keeps a byte order mark",
			source: "\ufeff[Script Info]\nTitle: T\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "\ufeff[Script Info]\nTitle: T\n\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "writes several fonts in order",
			source: "[Script Info]\n",
			fonts:  []EmbeddedFont{zeroFont("b.ttf"), zeroFont("a.ttf")},
			want:   "[Script Info]\n\n[Fonts]\nfontname: b.ttf\n!!!!\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "does not count a fontname line outside the section",
			source: "[Script Info]\nfontname: a.ttf\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\nfontname: a.ttf\n\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "appends to a section before the next section",
			source: "[Script Info]\n\n[Fonts]\nfontname: old.ttf\nAAAA\n\n[Events]\nFormat: Layer\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n\n[Fonts]\nfontname: old.ttf\nAAAA\nfontname: a.ttf\n!!!!\n\n[Events]\nFormat: Layer\n",
		},
		{
			name:   "appends to a section that touches the next header",
			source: "[Script Info]\n[Fonts]\nfontname: old.ttf\nAAAA\n[Events]\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n[Fonts]\nfontname: old.ttf\nAAAA\nfontname: a.ttf\n!!!!\n[Events]\n",
		},
		{
			name:   "appends to a section at the end before its blank lines",
			source: "[Script Info]\n[Fonts]\nfontname: old.ttf\nAAAA\n\n\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n[Fonts]\nfontname: old.ttf\nAAAA\nfontname: a.ttf\n!!!!\n\n\n",
		},
		{
			name:   "ends the last data line of a section without a final newline",
			source: "[Script Info]\n[Fonts]\nfontname: old.ttf\nAAAA",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n[Fonts]\nfontname: old.ttf\nAAAA\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "fills a header-only section",
			source: "[Script Info]\n[Fonts]",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n[Fonts]\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "matches the section header without regard to case",
			source: "[Script Info]\n[fOnTs]\nfontname: old.ttf\nAAAA\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\n[fOnTs]\nfontname: old.ttf\nAAAA\nfontname: a.ttf\n!!!!\n",
		},
		{
			name:   "appends to a CRLF section",
			source: "[Script Info]\r\n[Fonts]\r\nfontname: old.ttf\r\nAAAA\r\n\r\n[Events]\r\n",
			fonts:  []EmbeddedFont{zeroFont("a.ttf")},
			want:   "[Script Info]\r\n[Fonts]\r\nfontname: old.ttf\r\nAAAA\r\nfontname: a.ttf\r\n!!!!\r\n\r\n[Events]\r\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := parseDocument(t, tc.source)
			updated, err := original.EmbedFonts(tc.fonts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, updated.String())
			assert.Equal(t, tc.source, original.String(), "the source document changed")
			assert.Equal(t, original.Contents(), updated.Contents())
		})
	}
}

func TestEmbedFontsRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	const source = "[Script Info]\n[Fonts]\nfontname: Old.ttf\nAAAA\n"

	testCases := []struct {
		name   string
		source string
		fonts  []EmbeddedFont
	}{
		{name: "name already embedded", source: source, fonts: []EmbeddedFont{zeroFont("Old.ttf")}},
		{name: "name already embedded in another case", source: source, fonts: []EmbeddedFont{zeroFont("old.TTF")}},
		{
			name:   "name embedded with extra spacing",
			source: "[Script Info]\n[Fonts]\nfontname:   Old.ttf  \nAAAA\n",
			fonts:  []EmbeddedFont{zeroFont("Old.ttf")},
		},
		{name: "second font repeats an embedded name", source: source, fonts: []EmbeddedFont{zeroFont("new.ttf"), zeroFont("old.ttf")}},
		{name: "name repeated in the request", source: source, fonts: []EmbeddedFont{zeroFont("a.ttf"), zeroFont("a.ttf")}},
		{name: "name repeated with no section", source: "[Script Info]\n", fonts: []EmbeddedFont{zeroFont("a.ttf"), zeroFont("A.ttf")}},
		{name: "empty name", source: source, fonts: []EmbeddedFont{{Data: []byte{1}}}},
		{name: "multi-line name", source: source, fonts: []EmbeddedFont{{Name: "a\nb", Data: []byte{1}}}},
		{name: "empty data", source: source, fonts: []EmbeddedFont{{Name: "a.ttf"}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := parseDocument(t, tc.source)
			updated, err := original.EmbedFonts(tc.fonts)
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Nil(t, updated)
			assert.Equal(t, tc.source, original.String())
		})
	}
}

func TestEmbedFontsWithoutFontsChangesNothing(t *testing.T) {
	t.Parallel()

	original := parseDocument(t, fixture(t))
	for _, fonts := range [][]EmbeddedFont{nil, {}} {
		updated, err := original.EmbedFonts(fonts)
		require.NoError(t, err)
		assert.Equal(t, original.String(), updated.String())
	}
}

func TestEmbedFontsRoundTripsFontData(t *testing.T) {
	t.Parallel()

	randomFont := func(name string, size int) EmbeddedFont {
		return EmbeddedFont{Name: name, Data: randomData(size)}
	}
	first, second, third := randomFont("first.ttf", 1000), randomFont("second.otf", 61), randomFont("third.ttf", 2)

	// The script writer places [Fonts] between the styles and the events.
	script := NewScript("Embedded", 1080, 1920)
	script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans})
	script.Fonts = []EmbeddedFont{first}
	written, err := script.MarshalText()
	require.NoError(t, err)

	document := parseDocument(t, string(written))
	updated, err := document.EmbedFonts([]EmbeddedFont{second, third})
	require.NoError(t, err)
	require.NoError(t, updated.Validate())

	want := map[string][]byte{first.Name: first.Data, second.Name: second.Data, third.Name: third.Data}
	assert.Equal(t, want, embeddedFontsOf(t, updated.String()))

	_, err = updated.EmbedFonts([]EmbeddedFont{second})
	require.ErrorIs(t, err, commerr.ErrInvalidValue, "embedding the same font twice")

	t.Run("a fixture with styles and events", func(t *testing.T) {
		t.Parallel()

		source := fixture(t)
		embedded, err := parseDocument(t, source).EmbedFonts([]EmbeddedFont{first, second})
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(embedded.String(), source), "existing lines changed")
		assert.Equal(t, map[string][]byte{first.Name: first.Data, second.Name: second.Data},
			embeddedFontsOf(t, embedded.String()))
		assert.Equal(t, parseDocument(t, source).Contents(), embedded.Contents())
	})
}
