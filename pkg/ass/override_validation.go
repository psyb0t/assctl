package ass

import (
	"math"
	"strconv"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

const (
	maxAlphaValue     = 255
	pointCoordinates  = 2
	bezierCoordinates = 6
	drawCommands      = drawMove + drawMoveOpen + drawLine + drawBezier + drawSpline + drawPoint + drawCloseSpline
	drawMove          = "m"
	drawMoveOpen      = "n"
	drawLine          = "l"
	drawBezier        = "b"
	drawSpline        = "s"
	drawPoint         = "p"
	drawCloseSpline   = "c"
)

func finite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func (o *Override) validate() error {
	if o.Position != nil && o.Movement != nil || o.Clip != nil && o.VectorClip != nil || o.Fade != nil && o.SimpleFade != nil {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "choose one position, clip, and fade form")
	}
	for _, point := range []*Position{o.Position, o.Origin} {
		if point != nil && !finite(point.X, point.Y) {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "non-finite position")
		}
	}
	if m := o.Movement; m != nil {
		if !finite(m.X1, m.Y1, m.X2, m.Y2) || (m.T1 == nil) != (m.T2 == nil) {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "invalid movement coordinates or incomplete times")
		}
		if m.T1 != nil && (*m.T1 < 0 || *m.T2 < *m.T1) {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "invalid movement timing")
		}
	}
	if o.Alignment != nil && (*o.Alignment < AlignmentBottomLeft || *o.Alignment > AlignmentTopRight) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "alignment must be from 1 to 9")
	}
	if o.WrapStyle != nil && (*o.WrapStyle < WrapStyleSmartTop || *o.WrapStyle > WrapStyleSmartBottom) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "wrap style must be from 0 to 3")
	}
	for _, value := range []*float64{o.FontSize, o.Border, o.BorderX, o.BorderY, o.Shadow, o.Blur, o.ScaleX, o.ScaleY} {
		if value != nil && *value < 0 {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "negative size, scale, border, blur, or shadow")
		}
	}
	for _, value := range []*int{o.BlurEdges, o.DrawingMode} {
		if value != nil && *value < 0 {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "negative edge blur or drawing scale")
		}
	}
	if o.ResetStyle != nil && strings.ContainsAny(*o.ResetStyle, tagDelimiterChars) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "style reset contains an ASS delimiter")
	}
	if o.KaraokeStart != nil && *o.KaraokeStart < 0 {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "negative karaoke start")
	}
	if k := o.Karaoke; k != nil {
		if k.Duration < 0 || (k.Mode != "" && k.Mode != KaraokeInstant && k.Mode != KaraokeSweep && k.Mode != KaraokeOutline) {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "invalid karaoke mode or duration")
		}
	}
	if f := o.SimpleFade; f != nil && (f.In < 0 || f.Out < 0) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "negative fade duration")
	}
	if f := o.Fade; f != nil {
		for _, alpha := range []int{f.A1, f.A2, f.A3} {
			if alpha < 0 || alpha > maxAlphaValue {
				return ctxerrors.Wrap(commerr.ErrInvalidValue, "fade alpha must be from 0 to 255")
			}
		}
		if f.T1 < 0 || f.T2 < f.T1 || f.T3 < f.T2 || f.T4 < f.T3 {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "fade times must be nonnegative and ordered")
		}
	}
	if c := o.Clip; c != nil && (!finite(c.X1, c.Y1, c.X2, c.Y2) || c.X2 < c.X1 || c.Y2 < c.Y1) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "invalid rectangular clip")
	}
	if c := o.VectorClip; c != nil {
		if c.Scale < 0 {
			return ctxerrors.Wrap(commerr.ErrInvalidValue, "negative vector clip scale")
		}
		if err := validateDrawing(c.Drawing); err != nil {
			return ctxerrors.Wrap(err, "invalid vector clip drawing")
		}
	}
	return nil
}

func validateDrawing(drawing string) error {
	if strings.ContainsAny(drawing, lineBreakChars) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "drawing must fit one ASS event line")
	}
	fields := strings.Fields(drawing)
	if len(fields) == 0 || (fields[0] != drawMove && fields[0] != drawMoveOpen) {
		return ctxerrors.Wrap(commerr.ErrInvalidValue, "drawing must start with m or n")
	}
	for index := 0; index < len(fields); {
		command := fields[index]
		index++
		count := 0
		for index < len(fields) && !strings.Contains(drawCommands, fields[index]) {
			value, err := strconv.ParseFloat(fields[index], 64)
			if err != nil || !finite(value) {
				return ctxerrors.Wrap(commerr.ErrInvalidValue, "drawing coordinates must be finite numbers")
			}
			count++
			index++
		}
		valid := false
		switch command {
		case drawMove, drawMoveOpen, drawPoint:
			valid = count == pointCoordinates
		case drawLine:
			valid = count >= pointCoordinates && count%pointCoordinates == 0
		case drawBezier:
			valid = count >= bezierCoordinates && count%bezierCoordinates == 0
		case drawSpline:
			valid = count >= bezierCoordinates && count%pointCoordinates == 0
		case drawCloseSpline:
			valid = count == 0
		}
		if !valid {
			return ctxerrors.Wrapf(commerr.ErrInvalidValue, "invalid %s drawing coordinate count", command)
		}
	}
	return nil
}
