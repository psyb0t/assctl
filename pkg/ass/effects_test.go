package ass

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeChars follows libass's decode_chars: up to four characters of six bits
// each, offset by 33, give up to three bytes.
func decodeChars(source []byte) []byte {
	var value uint32
	for index, char := range source {
		value |= uint32((char-embeddedFontOffset)&sixBits) << (6 * (3 - index))
	}

	decoded := []byte{byte(value >> 16)}
	if len(source) >= 3 {
		decoded = append(decoded, byte(value>>8&0xff))
	}
	if len(source) >= 4 {
		decoded = append(decoded, byte(value&0xff))
	}

	return decoded
}

// decodeFont follows libass's decode_font for the joined data lines of one font.
func decodeFont(t *testing.T, lines []string) []byte {
	t.Helper()

	source := []byte(strings.Join(lines, ""))
	require.NotEqual(t, 1, len(source)%4, "libass rejects data of 4n+1 characters")

	decoded := make([]byte, 0, len(source)/4*3+2)
	position := 0
	for ; position+4 <= len(source); position += 4 {
		decoded = append(decoded, decodeChars(source[position:position+4])...)
	}
	if rest := source[position:]; len(rest) > 0 {
		decoded = append(decoded, decodeChars(rest)...)
	}

	return decoded
}

// embeddedFontsOf collects the fonts of every [Fonts] section the way libass
// reads them: a "fontname:" line starts a font and the lines after it hold its data.
func embeddedFontsOf(t *testing.T, text string) map[string][]byte {
	t.Helper()

	fonts := make(map[string][]byte)
	var name string
	var data []string
	flush := func() {
		if name != "" {
			fonts[name] = decodeFont(t, data)
		}
		name, data = "", nil
	}

	inFonts := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "[") {
			flush()
			inFonts = strings.EqualFold(line, headerFonts)

			continue
		}
		if !inFonts || line == "" {
			continue
		}
		if value, ok := strings.CutPrefix(line, "fontname:"); ok {
			flush()
			name = strings.TrimLeft(value, " ")

			continue
		}
		data = append(data, line)
	}
	flush()

	return fonts
}

// randomData returns size bytes from a generator seeded by the size, so every
// call is repeatable and no generator is shared between parallel tests.
func randomData(size int) []byte {
	random := rand.New(rand.NewPCG(7, uint64(size)))
	data := make([]byte, size)
	for index := range data {
		data[index] = byte(random.UintN(256))
	}

	return data
}

func TestBannerEffect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		delay       int
		leftToRight bool
		fadeWidth   int
		want        Effect
		wantErr     error
	}{
		{name: "right to left", delay: 50, want: "Banner;50;0"},
		{name: "left to right", delay: 50, leftToRight: true, want: "Banner;50;1"},
		{name: "fade width", delay: 30, leftToRight: true, fadeWidth: 20, want: "Banner;30;1;20"},
		{name: "zero delay", want: "Banner;0;0"},
		{name: "negative delay", delay: -1, wantErr: commerr.ErrInvalidValue},
		{name: "negative fade width", delay: 5, fadeWidth: -1, wantErr: commerr.ErrInvalidValue},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := BannerEffect(tc.delay, tc.leftToRight, tc.fadeWidth)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestScrollEffect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		direction  ScrollDirection
		top        int
		bottom     int
		delay      int
		fadeHeight int
		want       Effect
		wantErr    error
	}{
		{name: "up", direction: ScrollUp, top: 10, bottom: 200, delay: 30, want: "Scroll up;10;200;30"},
		{name: "down", direction: ScrollDown, top: 0, bottom: 100, delay: 5, want: "Scroll down;0;100;5"},
		{name: "fade height", direction: ScrollUp, top: 1, bottom: 2, delay: 3, fadeHeight: 4, want: "Scroll up;1;2;3;4"},
		{name: "unknown direction", direction: "Scroll left", wantErr: commerr.ErrInvalidValue},
		{name: "empty direction", wantErr: commerr.ErrInvalidValue},
		{name: "lowercase direction", direction: "scroll up", wantErr: commerr.ErrInvalidValue},
		{name: "negative top", direction: ScrollUp, top: -1, wantErr: commerr.ErrInvalidValue},
		{name: "negative bottom", direction: ScrollUp, bottom: -1, wantErr: commerr.ErrInvalidValue},
		{name: "negative delay", direction: ScrollDown, delay: -1, wantErr: commerr.ErrInvalidValue},
		{name: "negative fade height", direction: ScrollDown, fadeHeight: -1, wantErr: commerr.ErrInvalidValue},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ScrollEffect(tc.direction, tc.top, tc.bottom, tc.delay, tc.fadeHeight)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEffectsSurviveWritingAndParsing(t *testing.T) {
	t.Parallel()

	banner, err := BannerEffect(20, true, 8)
	require.NoError(t, err)
	scroll, err := ScrollEffect(ScrollDown, 5, 300, 12, 0)
	require.NoError(t, err)

	script := NewScript("Effects", 640, 360)
	script.AddStyle(Style{Name: "Default", FontName: FontDejaVuSans})
	script.AddEvent(Event{Style: "Default", End: time.Second, Effect: banner, Text: "one"})
	script.AddEvent(Event{Style: "Default", End: time.Second, Effect: scroll, Text: "two"})
	data, err := script.MarshalText()
	require.NoError(t, err)

	document, err := Parse(strings.NewReader(string(data)))
	require.NoError(t, err)
	events := document.Contents().Events
	require.Len(t, events, 2)
	assert.Equal(t, "Banner;20;1;8", events[0]["Effect"])
	assert.Equal(t, "Scroll down;5;300;12", events[1]["Effect"])
}

func TestEscapeText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want string
	}{
		{name: "empty"},
		{name: "plain", text: "hello, world", want: "hello, world"},
		{name: "closing brace only", text: "a}b", want: "a}b"},
		{name: "one block", text: "{note}", want: `\{{}note}`},
		{name: "text around a brace", text: "a {b", want: `a \{{}b`},
		{name: "consecutive braces", text: "{{", want: `\{{}\{{}`},
		{name: "ASS line break kept", text: `{a}\Nb`, want: `\{{}a}\Nb`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, EscapeText(tc.text))
		})
	}
}

func TestEncodeFontKnownValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		data []byte
		want []string
	}{
		{name: "empty", data: nil, want: []string{}},
		{name: "zero byte", data: []byte{0}, want: []string{"!!"}},
		{name: "max byte", data: []byte{0xff}, want: []string{"`Q"}},
		{name: "two max bytes", data: []byte{0xff, 0xff}, want: []string{"``]"}},
		{name: "three zero bytes", data: []byte{0, 0, 0}, want: []string{"!!!!"}},
		{name: "three max bytes", data: []byte{0xff, 0xff, 0xff}, want: []string{"````"}},
		{name: "three letters", data: []byte("Man"), want: []string{"47&O"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, encodeFont(tc.data))
		})
	}
}

func TestEncodeFontLineLayout(t *testing.T) {
	t.Parallel()

	// 60 bytes give exactly 80 characters.
	testCases := []struct {
		name      string
		size      int
		wantSizes []int
	}{
		{name: "one byte", size: 1, wantSizes: []int{2}},
		{name: "one full line", size: 60, wantSizes: []int{80}},
		{name: "one byte over a line", size: 61, wantSizes: []int{80, 2}},
		{name: "two full lines", size: 120, wantSizes: []int{80, 80}},
		{name: "partial last line", size: 100, wantSizes: []int{80, 54}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lines := encodeFont(make([]byte, tc.size))
			sizes := make([]int, 0, len(lines))
			for _, line := range lines {
				sizes = append(sizes, len(line))
			}
			assert.Equal(t, tc.wantSizes, sizes)
		})
	}

	t.Run("characters stay in the printable range", func(t *testing.T) {
		t.Parallel()

		data := make([]byte, 256)
		for index := range data {
			data[index] = byte(index)
		}
		for _, line := range encodeFont(data) {
			for _, char := range []byte(line) {
				assert.GreaterOrEqual(t, int(char), embeddedFontOffset)
				assert.LessOrEqual(t, int(char), embeddedFontOffset+sixBits)
			}
		}
	})
}

func TestEncodeFontRoundTripsThroughLibassDecoding(t *testing.T) {
	t.Parallel()

	t.Run("lengths zero to five", func(t *testing.T) {
		t.Parallel()

		for size := range 6 {
			data := randomData(size)
			assert.Equal(t, data, decodeFont(t, encodeFont(data)), "length %d", size)
		}
	})

	t.Run("every length across several lines", func(t *testing.T) {
		t.Parallel()

		for size := range 400 {
			data := randomData(size)
			assert.Equal(t, data, decodeFont(t, encodeFont(data)), "length %d", size)
		}
	})

	t.Run("large random font", func(t *testing.T) {
		t.Parallel()

		for _, size := range []int{100_000, 100_001, 100_002} {
			data := randomData(size)
			assert.Equal(t, data, decodeFont(t, encodeFont(data)), "length %d", size)
		}
	})

	t.Run("every byte value in every position", func(t *testing.T) {
		t.Parallel()

		var data []byte
		for value := range 256 {
			data = append(data, byte(value), byte(255-value), byte(value))
		}
		assert.Equal(t, data, decodeFont(t, encodeFont(data)))
	})
}

func TestFontLinesRejectInvalidFonts(t *testing.T) {
	t.Parallel()

	data := []byte{1, 2, 3}
	testCases := []struct {
		name  string
		fonts []EmbeddedFont
	}{
		{name: "empty name", fonts: []EmbeddedFont{{Data: data}}},
		{name: "padded name", fonts: []EmbeddedFont{{Name: " a.ttf", Data: data}}},
		{name: "trailing space", fonts: []EmbeddedFont{{Name: "a.ttf ", Data: data}}},
		{name: "line feed in name", fonts: []EmbeddedFont{{Name: "a\nb.ttf", Data: data}}},
		{name: "carriage return in name", fonts: []EmbeddedFont{{Name: "a\rb.ttf", Data: data}}},
		{name: "NUL in name", fonts: []EmbeddedFont{{Name: "a\x00b.ttf", Data: data}}},
		{name: "empty data", fonts: []EmbeddedFont{{Name: "a.ttf"}}},
		{name: "repeated name", fonts: []EmbeddedFont{{Name: "a.ttf", Data: data}, {Name: "a.ttf", Data: data}}},
		{name: "repeated name in another case", fonts: []EmbeddedFont{{Name: "a.ttf", Data: data}, {Name: "A.TTF", Data: data}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lines, err := fontLines(tc.fonts)
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Empty(t, lines)
		})
	}

	t.Run("lists each font's name then its data", func(t *testing.T) {
		t.Parallel()

		lines, err := fontLines([]EmbeddedFont{{Name: "a.ttf", Data: data}, {Name: "b.ttf", Data: []byte{9}}})
		require.NoError(t, err)
		assert.Equal(t, []string{"fontname: a.ttf", "!1)$", "fontname: b.ttf", "#1"}, lines)
	})
}
