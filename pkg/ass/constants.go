package ass

import "time"

// Built-in preset fonts. Custom presets may name any installed font family.
const (
	FontDejaVuSans               = "DejaVu Sans"
	FontDejaVuSerif              = "DejaVu Serif"
	FontDejaVuSansMono           = "DejaVu Sans Mono"
	FontTikTokSans               = "TikTok Sans"
	FontBarlowCondensedExtraBold = "Barlow Condensed ExtraBold"
	FontAnton                    = "Anton"
	FontTheBold                  = "THE BOLD FONT (FREE VERSION)"
)

// Section headers written in new scripts. Readers match them without regard to case.
const (
	headerScriptInfo = "[Script Info]"
	headerStyles     = "[V4+ Styles]"
	headerEvents     = "[Events]"
)

// Record types that start a line inside a section.
const (
	recordFormat   = "Format"
	recordStyle    = "Style"
	recordDialogue = "Dialogue"
	recordComment  = "Comment"
)

// Script Info keys with typed fields or generated defaults.
const (
	infoTitle                 = "Title"
	infoScriptType            = "ScriptType"
	infoPlayResX              = "PlayResX"
	infoPlayResY              = "PlayResY"
	infoScaledBorderAndShadow = "ScaledBorderAndShadow"
	infoYCbCrMatrix           = "YCbCr Matrix"

	scriptTypeV4Plus = "v4.00+"
	scaledBorderOn   = "yes"
	ycbcrMatrixNone  = "None"
)

// Style and event Format columns.
const (
	columnName            = "Name"
	columnFontname        = "Fontname"
	columnFontsize        = "Fontsize"
	columnPrimaryColour   = "PrimaryColour"
	columnSecondaryColour = "SecondaryColour"
	columnOutlineColour   = "OutlineColour"
	columnBackColour      = "BackColour"
	columnBold            = "Bold"
	columnItalic          = "Italic"
	columnUnderline       = "Underline"
	columnStrikeOut       = "StrikeOut"
	columnScaleX          = "ScaleX"
	columnScaleY          = "ScaleY"
	columnSpacing         = "Spacing"
	columnAngle           = "Angle"
	columnBorderStyle     = "BorderStyle"
	columnOutline         = "Outline"
	columnShadow          = "Shadow"
	columnAlignment       = "Alignment"
	columnMarginL         = "MarginL"
	columnMarginR         = "MarginR"
	columnMarginV         = "MarginV"
	columnEncoding        = "Encoding"
	columnLayer           = "Layer"
	columnStart           = "Start"
	columnEnd             = "End"
	columnStyle           = "Style"
	columnEffect          = "Effect"
	columnText            = "Text"
)

// ASS value encodings.
const (
	hexPrefix        = "&H"
	hexSuffix        = "&"
	opaqueAlphaHex   = "00"
	styleTrue        = "-1"
	styleFalse       = "0"
	overrideTrue     = "1"
	overrideFalse    = "0"
	clipTag          = "clip"
	inverseClipTag   = "iclip"
	centisecond      = 10 * time.Millisecond
	minutesPerHour   = 60
	secondsPerMinute = 60
	centisPerSecond  = 100
	maxClockMinute   = minutesPerHour - 1
	maxClockSecond   = secondsPerMinute - 1
	maxClockCenti    = centisPerSecond - 1
)

// Placement ratios shared by the social presets and the named text positions.
const (
	socialSideMarginRatio  = 0.18
	socialUpperMarginRatio = 0.18
	socialLowerMarginRatio = 0.4
)

// Characters that would end an ASS line or field early.
const (
	lineBreakChars    = "\r\n\x00"
	tagDelimiterChars = "{}\\" + lineBreakChars
)
