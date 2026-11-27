package ass_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/psyb0t/assctl/pkg/ass"
)

func TestOverrideSerializesTypographyAndMotion(t *testing.T) {
	t.Parallel()

	override := &ass.Override{
		Bold: new(false), Italic: new(true), Underline: new(true), StrikeOut: new(false),
		BoldWeight: ass.BoldWeightBold.Ptr(), FontName: new("DejaVu Sans"),
		Origin: &ass.Position{X: 10.5, Y: 20}, Alignment: ass.AlignmentTopRight.Ptr(),
		Movement: &ass.Movement{
			X1: 10, Y1: 20, X2: 30, Y2: 40,
			T1: new(100 * time.Millisecond), T2: new(750 * time.Millisecond),
		},
		ShadowColor: new(ass.ColorBlue), GlobalAlpha: new(ass.AlphaHalf), ShadowAlpha: new(ass.AlphaTransparent),
		Clip: &ass.Clip{X1: 4, Y1: 8, X2: 100, Y2: 200},
		Transform: []ass.Transform{
			{End: time.Second, Override: &ass.Override{Blur: new(1.5)}},
			{End: time.Second, Override: &ass.Override{}},
		},
	}
	got, err := override.Tags()
	const want = "{\\move(10,20,30,40,100,750)\\org(10.5,20)\\an9\\b0\\i1\\u1\\s0\\b700\\fnDejaVu Sans" +
		"\\4c&HFF0000&\\alpha&H80&\\4a&HFF&\\t(0,1000,1,\\blur1.5)\\clip(4,8,100,200)}"
	if err != nil || got != want {
		t.Fatalf("Tags returned %q, %v, want %q", got, err, want)
	}
}

func TestOverrideRejectsBadValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		override ass.Override
	}{
		{name: "font delimiter", override: ass.Override{FontName: new("Bad}Font")}},
		{name: "bad color", override: ass.Override{SecondColor: new(ass.Color("zzzzzz"))}},
		{name: "bad alpha", override: ass.Override{GlobalAlpha: new(ass.Alpha("0z"))}},
		{name: "NaN blur", override: ass.Override{Blur: new(math.NaN())}},
		{name: "infinite size", override: ass.Override{FontSize: new(math.Inf(1))}},
		{name: "reversed transform", override: ass.Override{Transform: []ass.Transform{
			{Start: time.Second, Override: &ass.Override{Blur: new(1.0)}},
		}}},
		{name: "negative acceleration", override: ass.Override{Transform: []ass.Transform{
			{End: time.Second, Accel: -1, Override: &ass.Override{Blur: new(1.0)}},
		}}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.override.Tags()
			if got != "" || !errors.Is(err, ass.ErrInvalidValue) {
				t.Fatalf("Tags returned %q, %v", got, err)
			}
		})
	}
}
