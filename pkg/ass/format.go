package ass

import (
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

const (
	numberPrecision  = 1000
	pixelTolerance   = 1e-9
	standardScale    = 100
	outputMode       = 0o600
	shortHexRGBLen   = 3
	longHexRGBLen    = 6
	bgrHexLen        = 6
	abgrHexLen       = 8
	alphaHexLen      = 2
	fieldSeparator   = ","
	formatSeparator  = ", "
	headerSeparator  = ": "
	lineBreakTag     = "\\N"
	fieldDelimiters  = fieldSeparator + lineBreakChars
	headerKeyInvalid = "[]:;" + lineBreakChars
)

var (
	styleColumns = []string{
		columnName, columnFontname, columnFontsize, columnPrimaryColour, columnSecondaryColour,
		columnOutlineColour, columnBackColour, columnBold, columnItalic, columnUnderline, columnStrikeOut,
		columnScaleX, columnScaleY, columnSpacing, columnAngle, columnBorderStyle, columnOutline,
		columnShadow, columnAlignment, columnMarginL, columnMarginR, columnMarginV, columnEncoding,
	}
	eventColumns = []string{
		columnLayer, columnStart, columnEnd, columnStyle, columnName,
		columnMarginL, columnMarginR, columnMarginV, columnEffect, columnText,
	}
	// Script Info keys set through typed Script fields, which custom headers may not repeat.
	typedHeaders = []string{infoTitle, infoScriptType, infoPlayResX, infoPlayResY, infoKerning, infoLanguage}
	languageCode = regexp.MustCompile("^[a-z]{2}$")
)

// borderStyles are the accepted Style.BorderStyle values. Zero is the unset
// value, which renderers draw as an outline and drop shadow.
var borderStyles = []BorderStyle{0, BorderStyleOutlineDropShadow, BorderStyleOpaqueBox, BorderStyleEventBox}

type scriptHeader struct{ name, value string }

var hexDigits = regexp.MustCompile("^[[:xdigit:]]+$")

// NewScript starts an empty script at the given coordinate resolution.
func NewScript(title string, width, height int) *Script {
	script := &Script{Title: title}
	script.Resolution.Width = width
	script.Resolution.Height = height

	return script
}

// AddStyle appends a style definition.
func (s *Script) AddStyle(style Style) {
	s.Styles = append(s.Styles, style)
}

// AddEvent appends a dialogue event.
func (s *Script) AddEvent(event Event) {
	s.Events = append(s.Events, event)
}

// ParseRGB accepts three or six hexadecimal RGB digits, with an optional #.
func ParseRGB(value string) (Color, error) {
	raw := strings.TrimPrefix(value, "#")
	if !hexDigits.MatchString(raw) {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "parse RGB color")
	}
	switch len(raw) {
	case shortHexRGBLen:
		raw = strings.Repeat(raw[0:1], 2) + strings.Repeat(raw[1:2], 2) + strings.Repeat(raw[2:3], 2)
	case longHexRGBLen:
	default:
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "RGB color needs three or six digits")
	}

	return Color(hexPrefix + opaqueAlphaHex + strings.ToUpper(raw[4:6]+raw[2:4]+raw[0:2])), nil
}

func colorValue(color Color, alpha Alpha, fallback Color) (string, error) {
	if color == "" {
		color = fallback
	}
	value := hexValue(string(color))
	if !hexDigits.MatchString(value) || (len(value) != bgrHexLen && len(value) != abgrHexLen) {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "ASS color needs BGR or ABGR hexadecimal digits")
	}

	opacity := opaqueAlphaHex
	if len(value) == abgrHexLen {
		opacity, value = value[:alphaHexLen], value[alphaHexLen:]
	}
	if alpha != "" {
		opacity = hexValue(string(alpha))
		if len(opacity) != alphaHexLen || !hexDigits.MatchString(opacity) {
			return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "ASS alpha needs two hexadecimal digits")
		}
	}

	return hexPrefix + opacity + value, nil
}

func hexValue(value string) string {
	return strings.TrimSuffix(strings.TrimPrefix(strings.ToUpper(value), hexPrefix), hexSuffix)
}

func integer(value int) string { return strconv.Itoa(value) }

// number writes value with at most three decimals, so ratio arithmetic such as
// 1920 * 0.045 prints as 86.4 rather than 86.39999999999999.
func number(value float64) string {
	rounded := math.Round(value*numberPrecision) / numberPrecision
	if rounded == 0 {
		// Rounding a small negative value gives negative zero, which prints as "-0".
		rounded = 0
	}

	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

func styleBoolean(value bool) string {
	if value {
		return styleTrue
	}

	return styleFalse
}

func styleFields(style Style) ([]string, error) {
	if !finite(style.FontSize, style.ScaleX, style.ScaleY, style.Spacing, style.Angle, style.Outline, style.Shadow) {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "non-finite style value")
	}
	if style.FontSize < 0 || style.ScaleX < 0 || style.ScaleY < 0 || style.Outline < 0 || style.Shadow < 0 {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "negative style size, scale, border, or shadow")
	}
	if style.MarginL < 0 || style.MarginR < 0 || style.MarginV < 0 || style.WrapStyle < WrapStyleSmartTop || style.WrapStyle > WrapStyleSmartBottom {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "invalid style margins or wrapping")
	}
	if !slices.Contains(borderStyles, style.BorderStyle) || style.Encoding < EncodingAuto {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "border style must be 0, 1, 3 or 4 and encoding at least -1")
	}
	if strings.TrimSpace(style.Name) == "" || strings.TrimSpace(style.FontName) == "" ||
		strings.ContainsAny(style.Name+style.FontName, fieldDelimiters) {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "style names and font names must be single fields")
	}
	colors := make([]string, 0, 4)
	for _, item := range []struct {
		color Color
		alpha Alpha
		base  Color
	}{
		{style.PrimaryColor, style.PrimaryAlpha, ColorWhite},
		{style.SecondColor, style.SecondAlpha, ColorWhite},
		{style.OutlineColor, style.OutlineAlpha, ColorBlack},
		{style.ShadowColor, style.ShadowAlpha, ColorBlack},
	} {
		color, err := colorValue(item.color, item.alpha, item.base)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "format style %q", style.Name)
		}
		colors = append(colors, color)
	}

	fields := []string{style.Name, style.FontName, number(style.FontSize)}
	fields = append(fields, colors...)
	fields = append(fields,
		styleBoolean(style.Bold), styleBoolean(style.Italic), styleBoolean(style.Underline),
		styleBoolean(style.StrikeOut), number(style.ScaleX), number(style.ScaleY), number(style.Spacing),
		number(style.Angle), integer(int(style.BorderStyle)), number(style.Outline), number(style.Shadow),
		integer(int(style.Alignment)), integer(style.MarginL), integer(style.MarginR), integer(style.MarginV),
		integer(int(style.Encoding)),
	)

	return fields, nil
}

func eventFields(event Event, wrap WrapStyle) ([]string, error) {
	if event.Start < 0 || event.End <= event.Start ||
		event.End/centisecond <= event.Start/centisecond {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "event needs a positive interval at centisecond precision")
	}
	if event.Layer < 0 || event.MarginL < 0 || event.MarginR < 0 || event.MarginV < 0 ||
		strings.ContainsAny(event.Style+event.Name+string(event.Effect), fieldDelimiters) ||
		strings.ContainsAny(event.Text, lineBreakChars) {
		return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "invalid event fields, use %s for a subtitle line break", lineBreakTag)
	}

	tags, err := event.Override.Tags()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "format event override")
	}
	if wrap != WrapStyleSmartTop && (event.Override == nil || event.Override.WrapStyle == nil) {
		tags = fmt.Sprintf("{\\q%d}", wrap) + tags
	}

	return []string{
		integer(event.Layer), formatTime(event.Start), formatTime(event.End), event.Style, event.Name,
		integer(event.MarginL), integer(event.MarginR), integer(event.MarginV), string(event.Effect),
		tags + event.Text,
	}, nil
}

func formatTime(value time.Duration) string {
	return fmt.Sprintf("%d:%02d:%02d.%02d", value/time.Hour, value/time.Minute%minutesPerHour,
		value/time.Second%secondsPerMinute, value/centisecond%centisPerSecond)
}

// MarshalText writes a new v4+ ASS script from typed values. Custom Info headers
// replace the generated defaults of the same name and follow them in sorted order.
func (s Script) MarshalText() ([]byte, error) {
	if s.Resolution.Width <= 0 || s.Resolution.Height <= 0 ||
		s.Resolution.Width > maxScriptDimension || s.Resolution.Height > maxScriptDimension ||
		strings.ContainsAny(s.Title, lineBreakChars) {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "script needs a positive resolution and a single-line title")
	}
	headers, err := s.infoHeaders()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "write script headers")
	}

	if s.Language != "" && !languageCode.MatchString(s.Language) {
		return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "language %q is not a lowercase ISO 639-1 code", s.Language)
	}
	fonts, err := fontLines(s.Fonts)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "embed fonts")
	}

	lines := []string{headerScriptInfo, infoTitle + headerSeparator + s.Title, infoScriptType + headerSeparator + scriptTypeV4Plus}
	if s.Kerning {
		lines = append(lines, infoKerning+headerSeparator+headerYes)
	}
	if s.Language != "" {
		lines = append(lines, infoLanguage+headerSeparator+s.Language)
	}
	for _, key := range slices.Sorted(maps.Keys(headers)) {
		lines = append(lines, headers[key].name+headerSeparator+headers[key].value)
	}
	lines = append(lines,
		infoPlayResX+headerSeparator+integer(s.Resolution.Width),
		infoPlayResY+headerSeparator+integer(s.Resolution.Height),
		"", headerStyles, recordFormat+headerSeparator+strings.Join(styleColumns, formatSeparator),
	)

	styles := make(map[string]WrapStyle, len(s.Styles))
	for _, style := range s.Styles {
		if _, exists := styles[style.Name]; exists {
			return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "duplicate style %q", style.Name)
		}
		fields, err := styleFields(style)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "write ASS style")
		}
		styles[style.Name] = style.WrapStyle
		lines = append(lines, recordStyle+headerSeparator+strings.Join(fields, fieldSeparator))
	}

	if len(fonts) > 0 {
		lines = append(lines, "", headerFonts)
		lines = append(lines, fonts...)
	}
	lines = append(lines, "", headerEvents, recordFormat+headerSeparator+strings.Join(eventColumns, formatSeparator))
	for index, event := range s.Events {
		wrap, exists := styles[event.Style]
		if !exists {
			return nil, ctxerrors.Wrapf(commerr.ErrNotFound, "event %d refers to style %q", index+1, event.Style)
		}
		fields, err := eventFields(event, wrap)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "write event %d", index+1)
		}
		kind := recordDialogue
		if event.Comment {
			kind = recordComment
		}
		lines = append(lines, kind+headerSeparator+strings.Join(fields, fieldSeparator))
	}

	content := strings.Join(lines, "\n") + "\n"
	if !utf8.ValidString(content) {
		return nil, ctxerrors.Wrap(commerr.ErrInvalidValue, "ASS text must be valid UTF-8")
	}

	return []byte(content), nil
}

// infoHeaders returns the generated default headers merged with s.Info, keyed by lowercase name.
func (s Script) infoHeaders() (map[string]scriptHeader, error) {
	headers := map[string]scriptHeader{
		strings.ToLower(infoScaledBorderAndShadow): {name: infoScaledBorderAndShadow, value: headerYes},
		strings.ToLower(infoYCbCrMatrix):           {name: infoYCbCrMatrix, value: ycbcrMatrixNone},
	}
	seen := make(map[string]bool, len(s.Info))
	for key, value := range s.Info {
		if strings.TrimSpace(key) != key || key == "" ||
			strings.ContainsAny(key, headerKeyInvalid) || strings.ContainsAny(value, lineBreakChars) {
			return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "invalid script header %q", key)
		}
		lower := strings.ToLower(key)
		if seen[lower] {
			return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "duplicate case-insensitive script header %q", key)
		}
		seen[lower] = true
		if slices.ContainsFunc(typedHeaders, func(typed string) bool { return strings.EqualFold(typed, key) }) {
			return nil, ctxerrors.Wrapf(commerr.ErrInvalidValue, "header %q has a typed Script field", key)
		}
		headers[lower] = scriptHeader{name: key, value: value}
	}

	return headers, nil
}

// Encode writes an ASS script to w.
func (s Script) Encode(w io.Writer) error {
	content, err := s.MarshalText()
	if err != nil {
		return ctxerrors.Wrap(err, "encode ASS script")
	}
	written, err := w.Write(content)
	if err != nil {
		return ctxerrors.Wrap(err, "write ASS script")
	}
	if written != len(content) {
		return ctxerrors.Wrap(io.ErrShortWrite, "write complete ASS script")
	}

	return nil
}

// WriteToFile writes an ASS script to the requested path.
func (s Script) WriteToFile(path string) error {
	content, err := s.MarshalText()
	if err != nil {
		return ctxerrors.Wrap(err, "encode ASS file")
	}
	if err := os.WriteFile(path, content, outputMode); err != nil {
		return ctxerrors.Wrap(err, "write ASS file")
	}

	return nil
}
