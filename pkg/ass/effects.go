package ass

import (
	"strconv"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

// ScrollDirection selects the vertical scroll effect.
type ScrollDirection string

const (
	ScrollUp   ScrollDirection = "Scroll up"
	ScrollDown ScrollDirection = "Scroll down"
)

const (
	effectBanner         = "Banner"
	effectSeparator      = ";"
	bannerLeftToRight    = "1"
	bannerRightToLeft    = "0"
	escapedOpenBrace     = `\{{}`
	openBrace            = "{"
	embeddedFontLineSize = 80
	embeddedFontOffset   = 33
	sixBits              = 0x3f
)

// BannerEffect scrolls the event horizontally across the screen, one script
// pixel every delay milliseconds. The event is drawn on one line. fadeWidth
// is kept in the effect for other renderers; libass ignores it.
func BannerEffect(delay int, leftToRight bool, fadeWidth int) (Effect, error) {
	if delay < 0 || fadeWidth < 0 {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "banner delay and fade width must not be negative")
	}
	direction := bannerRightToLeft
	if leftToRight {
		direction = bannerLeftToRight
	}
	parts := []string{effectBanner, strconv.Itoa(delay), direction}
	if fadeWidth > 0 {
		parts = append(parts, strconv.Itoa(fadeWidth))
	}

	return Effect(strings.Join(parts, effectSeparator)), nil
}

// ScrollEffect scrolls the event vertically between the script rows top and
// bottom, one pixel every delay milliseconds. fadeHeight is kept for other
// renderers; libass ignores it.
func ScrollEffect(direction ScrollDirection, top, bottom, delay, fadeHeight int) (Effect, error) {
	if direction != ScrollUp && direction != ScrollDown {
		return "", ctxerrors.Wrapf(commerr.ErrInvalidValue, "scroll direction %q", direction)
	}
	if top < 0 || bottom < 0 || delay < 0 || fadeHeight < 0 {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "scroll rows, delay and fade height must not be negative")
	}
	parts := []string{string(direction), strconv.Itoa(top), strconv.Itoa(bottom), strconv.Itoa(delay)}
	if fadeHeight > 0 {
		parts = append(parts, strconv.Itoa(fadeHeight))
	}

	return Effect(strings.Join(parts, effectSeparator)), nil
}

// EscapeText makes plain text safe to put in an event: an opening brace would
// otherwise start an override block and hide the text after it. libass shows
// the escaped brace; VSFilter shows a backslash followed by the text.
func EscapeText(text string) string {
	return strings.ReplaceAll(text, openBrace, escapedOpenBrace)
}

// encodeFont writes data in the ASS embedded-file encoding: each three bytes
// become four characters holding six bits each plus 33, in lines of 80.
func encodeFont(data []byte) []string {
	var encoded strings.Builder
	for start := 0; start < len(data); start += 3 {
		chunk := data[start:min(start+3, len(data))]
		var value uint32
		for index, b := range chunk {
			value |= uint32(b) << (16 - 8*index)
		}
		for index := range len(chunk) + 1 {
			encoded.WriteByte(byte((value>>(18-6*index))&sixBits) + embeddedFontOffset)
		}
	}

	text := encoded.String()
	lines := make([]string, 0, len(text)/embeddedFontLineSize+1)
	for start := 0; start < len(text); start += embeddedFontLineSize {
		lines = append(lines, text[start:min(start+embeddedFontLineSize, len(text))])
	}

	return lines
}

func fontLines(fonts []EmbeddedFont) ([]string, error) {
	var lines []string
	seen := make(map[string]bool, len(fonts))
	for _, font := range fonts {
		key := strings.ToLower(font.Name)
		if strings.TrimSpace(font.Name) != font.Name || font.Name == "" ||
			strings.ContainsAny(font.Name, lineBreakChars) || seen[key] {
			return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "embedded font name %q is empty, repeated or not one line", font.Name)
		}
		if len(font.Data) == 0 {
			return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "embedded font %q is empty", font.Name)
		}
		seen[key] = true
		lines = append(lines, fontNamePrefix+font.Name)
		lines = append(lines, encodeFont(font.Data)...)
	}

	return lines, nil
}
