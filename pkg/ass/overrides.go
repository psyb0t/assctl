package ass

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

const (
	scaleResetTag       = "\\fsc"
	maxTransformDepth   = 16
	defaultClipScale    = 1
	defaultAcceleration = 1
)

// Tags encodes the override as a leading ASS tag block.
func (o *Override) Tags() (string, error) {
	payload, err := o.tags(0)
	if err != nil {
		return "", ctxerrors.Wrap(err, "encode ASS override")
	}
	if payload == "" {
		return "", nil
	}
	if !utf8.ValidString(payload) {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "ASS override must be valid UTF-8")
	}

	return "{" + payload + "}", nil
}

func (o *Override) tags(depth int) (string, error) {
	if o == nil {
		return "", nil
	}
	if depth >= maxTransformDepth {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "override has too many nested transforms")
	}
	if err := o.validate(); err != nil {
		return "", ctxerrors.Wrap(err, "validate ASS override")
	}
	if o.FontName != nil && strings.ContainsAny(*o.FontName, tagDelimiterChars) {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "font name contains an ASS delimiter")
	}

	colors, err := o.colorTags()
	if err != nil {
		return "", ctxerrors.Wrap(err, "format override colors")
	}
	numeric, err := o.numberTags()
	if err != nil {
		return "", ctxerrors.Wrap(err, "format numeric overrides")
	}

	parts := appendValue([]string{}, "r", o.ResetStyle)
	if o.ResetScale {
		parts = append(parts, scaleResetTag)
	}
	parts = append(parts, o.positionTags(), o.fontTags(), colors, numeric)
	parts = appendValue(parts, "fe", o.Encoding)
	parts = appendValue(parts, "be", o.BlurEdges)
	parts = appendValue(parts, "p", o.DrawingMode)
	if o.SimpleFade != nil {
		parts = append(parts, fmt.Sprintf("\\fad(%d,%d)", o.SimpleFade.In.Milliseconds(), o.SimpleFade.Out.Milliseconds()))
	}
	if o.KaraokeStart != nil {
		parts = append(parts, fmt.Sprintf("\\kt%d", *o.KaraokeStart/centisecond))
	}
	if o.Karaoke != nil {
		mode := o.Karaoke.Mode
		if mode == "" {
			mode = KaraokeInstant
		}
		parts = append(parts, fmt.Sprintf("\\%s%d", mode, o.Karaoke.Duration/centisecond))
	}
	if clip := o.VectorClip; clip != nil {
		scale := clip.Scale
		if scale == 0 {
			scale = defaultClipScale
		}
		parts = append(parts, fmt.Sprintf("\\%s(%d,%s)", clipName(clip.Inverse), scale, clip.Drawing))
	}
	if o.Fade != nil {
		fade := o.Fade
		parts = append(parts, fmt.Sprintf("\\fade(%d,%d,%d,%d,%d,%d,%d)",
			fade.A1, fade.A2, fade.A3, fade.T1.Milliseconds(), fade.T2.Milliseconds(),
			fade.T3.Milliseconds(), fade.T4.Milliseconds()))
	}
	for _, animation := range o.Transform {
		part, err := animation.tags(depth + 1)
		if err != nil {
			return "", ctxerrors.Wrap(err, "format transform")
		}
		parts = append(parts, part)
	}
	if clip := o.Clip; clip != nil {
		parts = append(parts, fmt.Sprintf("\\%s(%s,%s,%s,%s)", clipName(clip.Inverse),
			number(clip.X1), number(clip.Y1), number(clip.X2), number(clip.Y2)))
	}
	parts = appendValue(parts, "q", o.WrapStyle)

	return strings.Join(parts, ""), nil
}

func clipName(inverse bool) string {
	if inverse {
		return inverseClipTag
	}

	return clipTag
}

func (o *Override) positionTags() string {
	var parts []string
	if at := o.Position; at != nil {
		parts = append(parts, "\\pos("+number(at.X)+","+number(at.Y)+")")
	}
	if move := o.Movement; move != nil {
		fields := []string{number(move.X1), number(move.Y1), number(move.X2), number(move.Y2)}
		if move.T1 != nil && move.T2 != nil {
			fields = append(fields, integer(int(move.T1.Milliseconds())), integer(int(move.T2.Milliseconds())))
		}
		parts = append(parts, "\\move("+strings.Join(fields, ",")+")")
	}
	if origin := o.Origin; origin != nil {
		parts = append(parts, "\\org("+number(origin.X)+","+number(origin.Y)+")")
	}
	parts = appendValue(parts, "an", o.Alignment)

	return strings.Join(parts, "")
}

func appendValue[T ~int | ~string](parts []string, tag string, value *T) []string {
	if value == nil {
		return parts
	}

	return append(parts, fmt.Sprintf("\\%s%v", tag, *value))
}

func (o *Override) fontTags() string {
	var parts []string
	for _, item := range []struct {
		tag string
		val *bool
	}{
		{"b", o.Bold}, {"i", o.Italic}, {"u", o.Underline}, {"s", o.StrikeOut},
	} {
		if item.val == nil {
			continue
		}
		toggle := overrideFalse
		if *item.val {
			toggle = overrideTrue
		}
		parts = append(parts, "\\"+item.tag+toggle)
	}
	parts = appendValue(parts, "b", o.BoldWeight)
	parts = appendValue(parts, "fn", o.FontName)

	return strings.Join(parts, "")
}

func (o *Override) numberTags() (string, error) {
	var parts []string
	for _, item := range []struct {
		tag string
		val *float64
	}{
		{"fs", o.FontSize},
		{"bord", o.Border},
		{"shad", o.Shadow},
		{"blur", o.Blur},
		{"fscx", o.ScaleX},
		{"fscy", o.ScaleY},
		{"frz", o.RotationZ},
		{"frx", o.RotationX},
		{"fry", o.RotationY},
		{"fax", o.ShearX},
		{"fay", o.ShearY},
		{"fsp", o.Spacing},
		{"xbord", o.BorderX},
		{"ybord", o.BorderY},
		{"xshad", o.ShadowX},
		{"yshad", o.ShadowY},
		{"pbo", o.DrawingBaseline},
	} {
		if item.val == nil {
			continue
		}
		if math.IsNaN(*item.val) || math.IsInf(*item.val, 0) {
			return "", ctxerrors.Wrapf(commerr.ErrInvalidValue, "non-finite %s override", item.tag)
		}
		parts = append(parts, "\\"+item.tag+number(*item.val))
	}

	return strings.Join(parts, ""), nil
}

func (o *Override) colorTags() (string, error) {
	var parts []string
	for _, item := range []struct {
		tag string
		val *Color
	}{
		{"1c", o.PrimaryColor}, {"2c", o.SecondColor}, {"3c", o.OutlineColor}, {"4c", o.ShadowColor},
	} {
		if item.val == nil {
			continue
		}
		color, err := colorValue(*item.val, "", ColorWhite)
		if err != nil {
			return "", ctxerrors.Wrapf(err, "color tag %s", item.tag)
		}
		parts = append(parts, "\\"+item.tag+hexPrefix+color[len(hexPrefix)+alphaHexLen:]+hexSuffix)
	}

	for _, item := range []struct {
		tag string
		val *Alpha
	}{
		{"alpha", o.GlobalAlpha},
		{"1a", o.PrimaryAlpha},
		{"2a", o.SecondAlpha},
		{"3a", o.OutlineAlpha},
		{"4a", o.ShadowAlpha},
	} {
		if item.val == nil {
			continue
		}
		color, err := colorValue(ColorWhite, *item.val, ColorWhite)
		if err != nil {
			return "", ctxerrors.Wrapf(err, "alpha tag %s", item.tag)
		}
		parts = append(parts, "\\"+item.tag+hexPrefix+color[len(hexPrefix):len(hexPrefix)+alphaHexLen]+hexSuffix)
	}

	return strings.Join(parts, ""), nil
}

func (a Transform) tags(depth int) (string, error) {
	if o := a.Override; o != nil && (o.Position != nil || o.Movement != nil || o.Origin != nil ||
		o.Alignment != nil || o.ResetStyle != nil || o.FontName != nil || o.Encoding != nil ||
		o.Bold != nil || o.BoldWeight != nil || o.Italic != nil || o.Underline != nil || o.StrikeOut != nil ||
		o.Fade != nil || o.SimpleFade != nil || o.Karaoke != nil || o.KaraokeStart != nil ||
		o.WrapStyle != nil || o.DrawingMode != nil || o.DrawingBaseline != nil || o.VectorClip != nil ||
		o.ResetScale || len(o.Transform) > 0) {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "transform contains a tag ASS cannot animate")
	}
	payload, err := a.Override.tags(depth)
	if err != nil {
		return "", ctxerrors.Wrap(err, "format transform overrides")
	}
	if payload == "" {
		return "", nil
	}
	// An End of zero runs the transform until the event ends.
	if a.Start < 0 || (a.End != 0 && a.End < a.Start) || math.IsNaN(a.Accel) || math.IsInf(a.Accel, 0) || a.Accel < 0 {
		return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "invalid transform timing or acceleration")
	}

	acceleration := a.Accel
	if acceleration == 0 {
		acceleration = defaultAcceleration
	}

	return fmt.Sprintf("\\t(%d,%d,%s,%s)", a.Start/time.Millisecond, a.End/time.Millisecond,
		number(acceleration), payload), nil
}
