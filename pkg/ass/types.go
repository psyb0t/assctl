// Package ass reads, writes, and styles Advanced SubStation Alpha subtitles.
package ass

import "time"

// Alpha holds an ASS transparency value from 00 for opaque to FF for transparent.
type Alpha string

// Alpha values for opaque, fully transparent and half transparent.
const (
	AlphaOpaque      Alpha = "&H00&"
	AlphaTransparent Alpha = "&HFF&"
	AlphaHalf        Alpha = "&H80&"
)

// Color holds an ASS color in BGR or ABGR format.
type Color string

const (
	ColorWhite   Color = "&H00FFFFFF"
	ColorBlack   Color = "&H00000000"
	ColorRed     Color = "&H000000FF"
	ColorGreen   Color = "&H0000FF00"
	ColorBlue    Color = "&H00FF0000"
	ColorYellow  Color = "&H0000FFFF"
	ColorMagenta Color = "&H00FF00FF"
	ColorCyan    Color = "&H00FFFF00"

	ColorGray      Color = "&H00808080"
	ColorDarkGray  Color = "&H00404040"
	ColorLightGray Color = "&H00C0C0C0"

	ColorPurple Color = "&H00800080"
	ColorViolet Color = "&H00EE82EE"
	ColorIndigo Color = "&H004B0082"
	ColorPink   Color = "&H00CBC0FF"
	ColorOrange Color = "&H000080FF"
	ColorBrown  Color = "&H002A2AA5"
	ColorMaroon Color = "&H00000080"
	ColorNavy   Color = "&H00800000"
	ColorOlive  Color = "&H00008080"
	ColorTeal   Color = "&H00808000"

	ColorDeepPink       Color = "&H009314FF"
	ColorSkyBlue        Color = "&H00EBCE87"
	ColorLimeGreen      Color = "&H0032CD32"
	ColorGoldenRod      Color = "&H0020A5DA"
	ColorCornflowerBlue Color = "&H00ED9564"
	ColorBlueViolet     Color = "&H00E22B8A"
	ColorTurquoise      Color = "&H00D0E040"
	ColorSeaGreen       Color = "&H00578B2E"
	ColorHotPink        Color = "&H00B469FF"
	ColorDodgerBlue     Color = "&H00FF901E"
)

// Ptr returns a pointer to a copy of c, for optional Override fields.
func (c Color) Ptr() *Color {
	return &c
}

// Ptr returns a pointer to a copy of a, for optional Override fields.
func (a Alpha) Ptr() *Alpha {
	return &a
}

// Alignment represents text positioning using numpad-style layout
type Alignment int

const (
	AlignmentBottomLeft   Alignment = 1
	AlignmentBottomCenter Alignment = 2
	AlignmentBottomRight  Alignment = 3
	AlignmentMiddleLeft   Alignment = 4
	AlignmentMiddleCenter Alignment = 5
	AlignmentMiddleRight  Alignment = 6
	AlignmentTopLeft      Alignment = 7
	AlignmentTopCenter    Alignment = 8
	AlignmentTopRight     Alignment = 9
)

// Ptr returns a pointer to a copy of a, for optional fields.
func (a Alignment) Ptr() *Alignment {
	return &a
}

// BorderStyle represents the type of border to draw around text
type BorderStyle int

const (
	BorderStyleOutlineDropShadow BorderStyle = 1
	BorderStyleOpaqueBox         BorderStyle = 3
)

// Ptr returns a pointer to a copy of b, for optional fields.
func (b BorderStyle) Ptr() *BorderStyle {
	return &b
}

// Encoding represents character encoding options
type Encoding int

const (
	EncodingANSI       Encoding = 0
	EncodingDefault    Encoding = 1
	EncodingSymbol     Encoding = 2
	EncodingShiftJIS   Encoding = 128
	EncodingHangeul    Encoding = 129
	EncodingGB2312     Encoding = 134
	EncodingChineseBig Encoding = 136
	EncodingMac        Encoding = 77
	EncodingKorean     Encoding = 129
	EncodingJohab      Encoding = 130
	EncodingOEM        Encoding = 255
)

// Ptr returns a pointer to a copy of e, for optional fields.
func (e Encoding) Ptr() *Encoding {
	return &e
}

// BoldWeight represents predefined weights for bold text
type BoldWeight int

const (
	BoldWeightNone   BoldWeight = 0
	BoldWeightNormal BoldWeight = 1
	BoldWeightBold   BoldWeight = 700
	BoldWeightBolder BoldWeight = 900
)

// Ptr returns a pointer to a copy of b, for optional fields.
func (b BoldWeight) Ptr() *BoldWeight {
	return &b
}

// Effect represents predefined special effects
type Effect string

const (
	EffectKaraoke     Effect = "Karaoke"
	EffectScroll      Effect = "Scroll"
	EffectBanner      Effect = "Banner"
	EffectScrollUp    Effect = "Scroll up"
	EffectScrollDown  Effect = "Scroll down"
	EffectScrollLeft  Effect = "Scroll left"
	EffectScrollRight Effect = "Scroll right"
)

// Ptr returns a pointer to a copy of e, for optional fields.
func (e Effect) Ptr() *Effect {
	return &e
}

// WrapStyle represents text wrapping behavior
type WrapStyle int

const (
	// Smart wrapping, top line is wider
	WrapStyleSmartTop WrapStyle = 0
	// End-of-line wrapping, right margin is strictly enforced
	WrapStyleEndOfLine WrapStyle = 1
	// Smart wrapping with a wider bottom line.
	WrapStyleSmartBottom WrapStyle = 3
	// Forced wrapping, only wrap at \n or \N
	WrapStyleForced WrapStyle = 2
	WrapStyleNoWrap WrapStyle = WrapStyleForced
)

// Ptr returns a pointer to a copy of w, for optional fields.
func (w WrapStyle) Ptr() *WrapStyle {
	return &w
}

// Style represents an ASS style definition that can be applied to multiple subtitle events.
type Style struct {
	Name         string      // Unique identifier for the style
	FontName     string      // Name of the font to use
	FontSize     float64     // Font size in ASS script coordinates.
	PrimaryColor Color       // Primary color in ASS format: &HBBGGRR&
	SecondColor  Color       // Secondary color (for karaoke) in ASS format: &HBBGGRR&
	OutlineColor Color       // Outline color in ASS format: &HBBGGRR&
	ShadowColor  Color       // Shadow color in ASS format: &HBBGGRR&
	PrimaryAlpha Alpha       // Primary color alpha
	SecondAlpha  Alpha       // Secondary color alpha
	OutlineAlpha Alpha       // Outline alpha
	ShadowAlpha  Alpha       // Shadow alpha
	Bold         bool        // Whether text should be bold
	Italic       bool        // Whether text should be italic
	Underline    bool        // Whether text should be underlined
	StrikeOut    bool        // Whether text should be struck through
	ScaleX       float64     // Horizontal scaling as percentage (100 = normal)
	ScaleY       float64     // Vertical scaling as percentage (100 = normal)
	Spacing      float64     // Additional space between letters in pixels
	Angle        float64     // Rotation angle in degrees
	BorderStyle  BorderStyle // Style of border around text
	Outline      float64     // Width of the outline in pixels
	Shadow       float64     // Depth of the shadow in pixels
	Alignment    Alignment   // Text alignment using numpad layout
	MarginL      int         // Left margin in pixels
	MarginR      int         // Right margin in pixels
	MarginV      int         // Vertical margin in pixels
	Encoding     Encoding    // Character encoding number
	WrapStyle    WrapStyle   // Text wrapping style
}

// Dialogue represents a single subtitle event (dialogue line) in the ASS file.
// This is a helper struct that is used when constructing an event.
type Dialogue struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

// Event represents a single subtitle event (dialogue line) in the ASS file.
type Event struct {
	Comment  bool          // Emit a Comment event instead of Dialogue.
	Layer    int           // Layer number (higher layers are drawn on top)
	Start    time.Duration // Start time of the subtitle
	End      time.Duration // End time of the subtitle
	Style    string        // Name of the style to use
	Name     string        // Name of the event (actor/character name)
	MarginL  int           // Left margin override in pixels
	MarginR  int           // Right margin override in pixels
	MarginV  int           // Vertical margin override in pixels
	Effect   Effect        // Special effect name
	Text     string        // The actual subtitle text
	Override *Override     // Optional override tags for custom formatting
}

// Override represents text override tags that can be applied to a subtitle event.
type Override struct {
	ResetStyle      *string // Reset overrides, optionally to another named style.
	Spacing         *float64
	Encoding        *Encoding
	BorderX         *float64
	BorderY         *float64
	ShadowX         *float64
	ShadowY         *float64
	BlurEdges       *int
	DrawingMode     *int
	DrawingBaseline *float64
	Karaoke         *Karaoke
	KaraokeStart    *time.Duration
	SimpleFade      *SimpleFade
	VectorClip      *VectorClip
	Position        *Position   // Absolute positioning
	Movement        *Movement   // Movement animation
	Origin          *Position   // Origin point for rotations
	Alignment       *Alignment  // Text alignment override
	Bold            *bool       // Bold override
	BoldWeight      *BoldWeight // Custom bold weight
	Italic          *bool       // Italic override
	Underline       *bool       // Underline override
	StrikeOut       *bool       // Strikeout override
	FontSize        *float64    // Font size override
	FontName        *string     // Font name override
	PrimaryColor    *Color      // Primary color override
	SecondColor     *Color      // Secondary color override
	OutlineColor    *Color      // Outline color override
	ShadowColor     *Color      // Shadow color override
	GlobalAlpha     *Alpha      // Global alpha override
	PrimaryAlpha    *Alpha      // Primary color alpha
	SecondAlpha     *Alpha      // Secondary color alpha
	OutlineAlpha    *Alpha      // Outline alpha
	ShadowAlpha     *Alpha      // Shadow alpha
	Border          *float64    // Border width override
	Shadow          *float64    // Shadow depth override
	Blur            *float64    // Blur effect
	ScaleX          *float64    // Horizontal scaling override
	ScaleY          *float64    // Vertical scaling override
	RotationZ       *float64    // Z-axis rotation
	RotationX       *float64    // X-axis rotation
	RotationY       *float64    // Y-axis rotation
	ShearX          *float64    // X-axis shearing
	ShearY          *float64    // Y-axis shearing
	Fade            *Fade       // Fade effect
	Transform       []Transform // Transform animations
	Clip            *Clip       // Clipping rectangle
	WrapStyle       *WrapStyle  // Wrap style override
}

// Position is a point in script coordinates.
type Position struct {
	X float64
	Y float64
}

// Movement moves text from (X1, Y1) to (X2, Y2). T1 and T2 bound the movement in event time and must be set together.
type Movement struct {
	X1 float64
	Y1 float64
	X2 float64
	Y2 float64
	T1 *time.Duration
	T2 *time.Duration
}

// Fade is the complex \fade tag: alphas A1, A2 and A3 from 0 to 255 change at times T1 to T4.
type Fade struct {
	A1 int
	A2 int
	A3 int
	T1 time.Duration
	T2 time.Duration
	T3 time.Duration
	T4 time.Duration
}

// Transform animates Override from Start to End. Accel zero selects linear timing.
type Transform struct {
	Start    time.Duration
	End      time.Duration
	Accel    float64
	Override *Override
}

// Clip limits drawing to a rectangle, or to everything outside it when Inverse is set.
type Clip struct {
	X1      float64
	Y1      float64
	X2      float64
	Y2      float64
	Inverse bool
}

// Script represents a complete ASS subtitle script including styles and events.
type Script struct {
	Title      string
	Info       map[string]string // Additional script headers; title, type and resolution use their typed fields.
	Resolution struct {
		Width  int
		Height int
	}
	Styles []Style
	Events []Event
}

// KaraokeMode selects the karaoke highlight effect.
type KaraokeMode string

const (
	KaraokeInstant KaraokeMode = "k"
	KaraokeSweep   KaraokeMode = "kf"
	KaraokeOutline KaraokeMode = "ko"
)

// Karaoke highlights the following syllable for Duration. An empty Mode selects KaraokeInstant.
type Karaoke struct {
	Mode     KaraokeMode
	Duration time.Duration
}

// SimpleFade fades in over In and out over Out.
type SimpleFade struct {
	In  time.Duration
	Out time.Duration
}

// VectorClip uses ASS drawing commands. Scale zero selects the default scale of one.
type VectorClip struct {
	Scale   int
	Drawing string
	Inverse bool
}
