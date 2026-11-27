package ass

import (
	"slices"
	"strconv"
	"strings"

	"github.com/psyb0t/ctxerrors"
)

const (
	blackGoldPreset = "black-gold"
	inlineASSChars  = "{\\"
	fallbackHeight  = 1080
)

// Preset sets the complete look of a subtitle style. Ratios are relative to script height.
type Preset struct {
	Name            string
	Font            string
	Text            Color
	Highlight       Color
	Background      Color
	OutlineColor    Color
	BackgroundAlpha Alpha
	FontRatio       float64
	MarginRatio     float64
	Outline         float64
	Shadow          float64
	Padding         float64
	Bold            bool
	Alignment       Alignment
}

func presets() []Preset {
	return []Preset{
		{
			Name: blackGoldPreset, Font: "DejaVu Sans", Text: ColorWhite, Highlight: "&H0053D4FF",
			Background: ColorBlack, OutlineColor: ColorBlack, BackgroundAlpha: AlphaOpaque,
			FontRatio: 0.04, MarginRatio: 0.1, Outline: 2.5, Shadow: 1, Padding: 5,
			Bold: true, Alignment: AlignmentBottomCenter,
		},
		{
			Name: "ivory-navy", Font: "DejaVu Serif", Text: "&H00E8F7FF", Highlight: "&H00FFCD88",
			Background: "&H00412A15", OutlineColor: "&H00301D0C", BackgroundAlpha: "&H18&",
			FontRatio: 0.043, MarginRatio: 0.085, Outline: 2, Shadow: 1.5, Padding: 5,
			Alignment: AlignmentBottomCenter,
		},
		{
			Name: "mint-plum", Font: "DejaVu Sans Mono", Text: "&H00C3FDD1", Highlight: "&H00A5C2FF",
			Background: "&H004E2140", OutlineColor: "&H0032182D", BackgroundAlpha: "&H20&",
			FontRatio: 0.038, MarginRatio: 0.11, Outline: 2, Shadow: 1, Padding: 6,
			Bold: true, Alignment: AlignmentBottomCenter,
		},
		{
			Name: "clean", Font: "DejaVu Sans", Text: ColorWhite, Highlight: ColorSkyBlue,
			Background: ColorBlack, OutlineColor: ColorBlack, BackgroundAlpha: AlphaTransparent,
			FontRatio: 0.036, MarginRatio: 0.06, Outline: 3, Shadow: 1.5,
			Alignment: AlignmentBottomCenter,
		},
	}
}

// Presets returns the built-in looks in display order.
func Presets() []Preset { return presets() }

// LookupPreset returns the requested look.
func LookupPreset(name string) (Preset, error) {
	for _, preset := range presets() {
		if preset.Name == name {
			return preset, nil
		}
	}

	return Preset{}, ctxerrors.Wrapf(ErrNotFound, "preset %q", name)
}

func (p Preset) foreground(name string, height int) Style {
	return Style{
		Name: name, FontName: p.Font, FontSize: float64(height) * p.FontRatio,
		PrimaryColor: p.Text, SecondColor: p.Highlight, OutlineColor: p.OutlineColor,
		ShadowColor: p.Background, ShadowAlpha: p.BackgroundAlpha, Bold: p.Bold,
		ScaleX: standardScale, ScaleY: standardScale, BorderStyle: BorderStyleOutlineDropShadow,
		Outline: p.Outline, Shadow: p.Shadow, Alignment: p.Alignment,
		MarginV: int(float64(height) * p.MarginRatio), Encoding: EncodingDefault,
	}
}

func (p Preset) box(style Style) Style {
	style.OutlineColor, style.OutlineAlpha = p.Background, p.BackgroundAlpha
	style.BorderStyle, style.Outline = BorderStyleOpaqueBox, p.Padding

	return style
}

// Theme replaces the selected style fields. A blank name applies the preset to every style.
func (d *Document) Theme(preset Preset, name string) (*Document, error) {
	height := fallbackHeight
	if item, ok := d.info["playresy"]; ok {
		value, err := strconv.Atoi(strings.TrimSpace(item.values[0]))
		if err == nil && value > 0 {
			height = value
		}
	}

	edit := Edit{Styles: make([]StyleEdit, 0, len(d.styles))}
	columns := strings.Split(styleFormat, ", ")
	for _, row := range d.styles {
		styleName := strings.TrimSpace(row.field("Name"))
		if name != "" && name != styleName {
			continue
		}
		style := preset.foreground(styleName, height)
		if preset.BackgroundAlpha != AlphaTransparent {
			style = preset.box(style)
		}
		values, err := styleFields(style)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "render theme style")
		}
		fields := make(map[string]string, len(columns))
		for _, column := range []string{
			"Fontname", "Fontsize", "PrimaryColour", "SecondaryColour", "OutlineColour",
			"BackColour", "Bold", "BorderStyle", "Outline", "Shadow", "Alignment", "MarginV",
		} {
			fields[column] = values[slices.Index(columns, column)]
		}
		edit.Styles = append(edit.Styles, StyleEdit{Name: styleName, Fields: fields})
	}
	if len(edit.Styles) == 0 {
		return nil, ctxerrors.Wrapf(ErrNotFound, "style %q", name)
	}

	return d.Apply(edit)
}

// CaptionSpec supplies timed lines and the visual settings for a new caption script.
type CaptionSpec struct {
	Title        string
	Width        int
	Height       int
	Dialogues    []Dialogue
	Uppercase    bool
	WordsPerLine int
}

// Captions builds timed captions with separate text and background layers when the preset has a box.
func (p Preset) Captions(spec CaptionSpec) Script {
	script := NewScript(spec.Title, spec.Width, spec.Height)
	text := p.foreground("Text", spec.Height)
	script.AddStyle(text)
	background := p.box(p.foreground("Background", spec.Height))
	boxVisible := p.BackgroundAlpha != AlphaTransparent
	if boxVisible {
		background.PrimaryAlpha = AlphaTransparent
		background.SecondAlpha = AlphaTransparent
		background.Shadow = 0
		script.AddStyle(background)
	}
	for _, dialogue := range spec.Dialogues {
		line := dialogue.Text
		if !strings.ContainsAny(line, inlineASSChars) {
			if spec.Uppercase {
				line = strings.ToUpper(line)
			}
			line = wrapWords(line, spec.WordsPerLine)
		}
		event := Event{Start: dialogue.Start, End: dialogue.End, Text: line, Style: text.Name}
		if boxVisible {
			underlay := event
			underlay.Style = background.Name
			script.AddEvent(underlay)
			event.Layer = 1
		}
		script.AddEvent(event)
	}

	return *script
}

func wrapWords(text string, limit int) string {
	if limit < 1 {
		return text
	}
	words := strings.Fields(text)
	if len(words) <= limit {
		return text
	}

	lines := (len(words) + limit - 1) / limit
	wrapped := make([]string, 0, lines)
	for count := lines; count > 0; count-- {
		size := (len(words) + count - 1) / count
		wrapped = append(wrapped, strings.Join(words[:size], " "))
		words = words[size:]
	}

	return strings.Join(wrapped, "\\N")
}
