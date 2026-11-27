package ass

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/psyb0t/ctxerrors"
)

const (
	styleFormat   = "Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding"
	eventFormat   = "Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text"
	standardScale = 100
	outputMode    = 0o600
)

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
		return "", ctxerrors.Wrap(ErrInvalidValue, "parse RGB color")
	}
	switch len(raw) {
	case 3:
		raw = strings.Repeat(raw[0:1], 2) + strings.Repeat(raw[1:2], 2) + strings.Repeat(raw[2:3], 2)
	case 6:
	default:
		return "", ctxerrors.Wrap(ErrInvalidValue, "RGB color needs three or six digits")
	}

	return Color("&H00" + strings.ToUpper(raw[4:6]+raw[2:4]+raw[0:2])), nil
}

func colorValue(color Color, alpha Alpha, fallback Color) (string, error) {
	if color == "" {
		color = fallback
	}
	value := strings.TrimSuffix(strings.TrimPrefix(strings.ToUpper(string(color)), "&H"), "&")
	if !hexDigits.MatchString(value) || (len(value) != 6 && len(value) != 8) {
		return "", ctxerrors.Wrap(ErrInvalidValue, "ASS color needs BGR or ABGR hexadecimal digits")
	}

	opacity := "00"
	if len(value) == 8 {
		opacity, value = value[:2], value[2:]
	}
	if alpha != "" {
		opacity = strings.TrimSuffix(strings.TrimPrefix(strings.ToUpper(string(alpha)), "&H"), "&")
		if len(opacity) != 2 || !hexDigits.MatchString(opacity) {
			return "", ctxerrors.Wrap(ErrInvalidValue, "ASS alpha needs two hexadecimal digits")
		}
	}

	return "&H" + opacity + value, nil
}

func integer(value int) string { return strconv.Itoa(value) }

func number(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

func styleBoolean(value bool) string {
	if value {
		return "-1"
	}
	return "0"
}

func styleFields(style Style) ([]string, error) {
	if strings.TrimSpace(style.Name) == "" || strings.TrimSpace(style.FontName) == "" ||
		strings.ContainsAny(style.Name+style.FontName, ",\r\n\x00") {
		return nil, ctxerrors.Wrap(ErrInvalidValue, "style names and font names must be single fields")
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
		event.End/(10*time.Millisecond) <= event.Start/(10*time.Millisecond) {
		return nil, ctxerrors.Wrap(ErrInvalidValue, "event needs a positive interval at centisecond precision")
	}
	if event.Layer < 0 || strings.ContainsAny(event.Style+event.Name+string(event.Effect), ",\r\n\x00") ||
		strings.ContainsAny(event.Text, "\r\n\x00") {
		return nil, ctxerrors.Wrap(ErrInvalidValue, "invalid event fields, use \\N for a subtitle line break")
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
	return fmt.Sprintf("%d:%02d:%02d.%02d", value/time.Hour, value/time.Minute%60,
		value/time.Second%60, value/(10*time.Millisecond)%100)
}

// MarshalText writes a new v4+ ASS script from typed values.
func (s Script) MarshalText() ([]byte, error) {
	if s.Resolution.Width <= 0 || s.Resolution.Height <= 0 || strings.ContainsAny(s.Title, "\r\n\x00") {
		return nil, ctxerrors.Wrap(ErrInvalidValue, "script needs a positive resolution and a single-line title")
	}
	lines := []string{
		"[Script Info]", "Title: " + s.Title, "ScriptType: v4.00+", "ScaledBorderAndShadow: yes",
		"YCbCr Matrix: None", "PlayResX: " + integer(s.Resolution.Width),
		"PlayResY: " + integer(s.Resolution.Height), "", "[V4+ Styles]", "Format: " + styleFormat,
	}

	styles := make(map[string]WrapStyle, len(s.Styles))
	for _, style := range s.Styles {
		if _, exists := styles[style.Name]; exists {
			return nil, ctxerrors.Wrapf(ErrInvalidValue, "duplicate style %q", style.Name)
		}
		fields, err := styleFields(style)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "write ASS style")
		}
		styles[style.Name] = style.WrapStyle
		lines = append(lines, "Style: "+strings.Join(fields, ","))
	}

	lines = append(lines, "", "[Events]", "Format: "+eventFormat)
	for index, event := range s.Events {
		wrap, exists := styles[event.Style]
		if !exists {
			return nil, ctxerrors.Wrapf(ErrNotFound, "event %d refers to style %q", index+1, event.Style)
		}
		fields, err := eventFields(event, wrap)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "write event %d", index+1)
		}
		lines = append(lines, "Dialogue: "+strings.Join(fields, ","))
	}

	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

// Encode writes an ASS script to w.
func (s Script) Encode(w io.Writer) error {
	content, err := s.MarshalText()
	if err != nil {
		return ctxerrors.Wrap(err, "encode ASS script")
	}
	if _, err := w.Write(content); err != nil {
		return ctxerrors.Wrap(err, "write ASS script")
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
