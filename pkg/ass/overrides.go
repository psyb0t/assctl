package ass

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/psyb0t/ctxerrors"
)

const maxTransformDepth = 16

// Tags encodes the override as a leading ASS tag block.
func (o *Override) Tags() (string, error) {
	payload, err := o.tags(0)
	if err != nil {
		return "", ctxerrors.Wrap(err, "encode ASS override")
	}
	if payload == "" {
		return "", nil
	}

	return "{" + payload + "}", nil
}

func (o *Override) tags(depth int) (string, error) {
	if o == nil {
		return "", nil
	}
	if depth >= maxTransformDepth {
		return "", ctxerrors.Wrap(ErrInvalidValue, "override has too many nested transforms")
	}
	if o.FontName != nil && strings.ContainsAny(*o.FontName, "{}\\\r\n\x00") {
		return "", ctxerrors.Wrap(ErrInvalidValue, "font name contains an ASS delimiter")
	}

	colors, err := o.colorTags()
	if err != nil {
		return "", ctxerrors.Wrap(err, "format override colors")
	}
	numeric, err := o.numberTags()
	if err != nil {
		return "", ctxerrors.Wrap(err, "format numeric overrides")
	}

	parts := []string{o.positionTags(), o.fontTags(), colors, numeric}
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
		name := "clip"
		if clip.Inverse {
			name = "iclip"
		}
		parts = append(parts, fmt.Sprintf("\\%s(%s,%s,%s,%s)", name,
			number(clip.X1), number(clip.Y1), number(clip.X2), number(clip.Y2)))
	}
	parts = appendValue(parts, "q", o.WrapStyle)

	return strings.Join(parts, ""), nil
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
	if value != nil {
		return append(parts, fmt.Sprintf("\\%s%v", tag, *value))
	}
	return parts
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
		toggle := "0"
		if *item.val {
			toggle = "1"
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
	} {
		if item.val == nil {
			continue
		}
		if math.IsNaN(*item.val) || math.IsInf(*item.val, 0) {
			return "", ctxerrors.Wrapf(ErrInvalidValue, "non-finite %s override", item.tag)
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
		parts = append(parts, "\\"+item.tag+"&H"+color[4:]+"&")
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
		parts = append(parts, "\\"+item.tag+"&H"+color[2:4]+"&")
	}

	return strings.Join(parts, ""), nil
}

func (a Transform) tags(depth int) (string, error) {
	payload, err := a.Override.tags(depth)
	if err != nil {
		return "", ctxerrors.Wrap(err, "format transform overrides")
	}
	if payload == "" {
		return "", nil
	}
	if a.Start < 0 || a.End < a.Start || math.IsNaN(a.Accel) || math.IsInf(a.Accel, 0) || a.Accel < 0 {
		return "", ctxerrors.Wrap(ErrInvalidValue, "invalid transform timing or acceleration")
	}

	acceleration := a.Accel
	if acceleration == 0 {
		acceleration = 1
	}

	return fmt.Sprintf("\\t(%d,%d,%s,%s)", a.Start/time.Millisecond, a.End/time.Millisecond,
		number(acceleration), payload), nil
}
