package ass

import (
	"math"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideSerializesTypographyAndMotion(t *testing.T) {
	t.Parallel()

	override := &Override{
		Bold: new(false), Italic: new(true), Underline: new(true), StrikeOut: new(false),
		BoldWeight: BoldWeightBold.Ptr(), FontName: new(FontDejaVuSans),
		Origin: &Position{X: 10.5, Y: 20}, Alignment: AlignmentTopRight.Ptr(),
		Movement: &Movement{
			X1: 10, Y1: 20, X2: 30, Y2: 40,
			T1: new(100 * time.Millisecond), T2: new(750 * time.Millisecond),
		},
		ShadowColor: ColorBlue.Ptr(), GlobalAlpha: AlphaHalf.Ptr(), ShadowAlpha: AlphaTransparent.Ptr(),
		Clip: &Clip{X1: 4, Y1: 8, X2: 100, Y2: 200},
		Transform: []Transform{
			{End: time.Second, Override: &Override{Blur: new(1.5)}},
			{End: time.Second, Override: &Override{}},
		},
	}

	const want = "{\\move(10,20,30,40,100,750)\\org(10.5,20)\\an9\\b0\\i1\\u1\\s0\\b700\\fnDejaVu Sans" +
		"\\4c&HFF0000&\\alpha&H80&\\4a&HFF&\\t(0,1000,1,\\blur1.5)\\clip(4,8,100,200)}"
	got, err := override.Tags()
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestOverrideRejectsBadValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		override Override
	}{
		{name: "font delimiter", override: Override{FontName: new("Bad}Font")}},
		{name: "bad color", override: Override{SecondColor: Color("zzzzzz").Ptr()}},
		{name: "bad alpha", override: Override{GlobalAlpha: Alpha("0z").Ptr()}},
		{name: "NaN blur", override: Override{Blur: new(math.NaN())}},
		{name: "infinite size", override: Override{FontSize: new(math.Inf(1))}},
		{name: "reversed transform", override: Override{Transform: []Transform{
			{Start: time.Second, Override: &Override{Blur: new(1.0)}},
		}}},
		{name: "negative acceleration", override: Override{Transform: []Transform{
			{End: time.Second, Accel: -1, Override: &Override{Blur: new(1.0)}},
		}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.override.Tags()
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
			assert.Empty(t, got)
		})
	}
}

func TestAllNumericOverridesRejectNonFiniteValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		build func(*float64) Override
	}{
		{name: "font size", build: func(v *float64) Override { return Override{FontSize: v} }},
		{name: "border", build: func(v *float64) Override { return Override{Border: v} }},
		{name: "shadow", build: func(v *float64) Override { return Override{Shadow: v} }},
		{name: "blur", build: func(v *float64) Override { return Override{Blur: v} }},
		{name: "scale x", build: func(v *float64) Override { return Override{ScaleX: v} }},
		{name: "scale y", build: func(v *float64) Override { return Override{ScaleY: v} }},
		{name: "rotation x", build: func(v *float64) Override { return Override{RotationX: v} }},
		{name: "rotation y", build: func(v *float64) Override { return Override{RotationY: v} }},
		{name: "rotation z", build: func(v *float64) Override { return Override{RotationZ: v} }},
		{name: "shear x", build: func(v *float64) Override { return Override{ShearX: v} }},
		{name: "shear y", build: func(v *float64) Override { return Override{ShearY: v} }},
		{name: "spacing", build: func(v *float64) Override { return Override{Spacing: v} }},
		{name: "border x", build: func(v *float64) Override { return Override{BorderX: v} }},
		{name: "border y", build: func(v *float64) Override { return Override{BorderY: v} }},
		{name: "shadow x", build: func(v *float64) Override { return Override{ShadowX: v} }},
		{name: "shadow y", build: func(v *float64) Override { return Override{ShadowY: v} }},
		{name: "drawing baseline", build: func(v *float64) Override { return Override{DrawingBaseline: v} }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
				override := tc.build(&value)
				_, err := override.Tags()
				require.ErrorIs(t, err, commerr.ErrInvalidValue, "accepted %v", value)
			}
		})
	}
}

func TestOverrideRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		override Override
	}{
		{name: "font name", override: Override{FontName: new("\xff")}},
		{name: "reset style", override: Override{ResetStyle: new("\xff")}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.override.Tags()
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
		})
	}
}

func TestOverrideFormatAndColorValidation(t *testing.T) {
	t.Parallel()

	t.Run("parses RGB colors", func(t *testing.T) {
		t.Parallel()

		red, err := ParseRGB("#f00")
		require.NoError(t, err)
		assert.Equal(t, ColorRed, red)

		_, err = ParseRGB("#ff")
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
	})

	t.Run("encodes every tag kind", func(t *testing.T) {
		t.Parallel()

		override := &Override{
			PrimaryColor: ColorRed.Ptr(),
			Movement:     &Movement{X1: 1, Y1: 2, X2: 3, Y2: 4},
			Fade:         &Fade{A1: 255, A2: 0, A3: 255, T2: time.Second, T3: 2 * time.Second, T4: 3 * time.Second},
			Transform: []Transform{{
				End: 2 * time.Second, Accel: 1.5,
				Override: &Override{FontSize: new(25.0)},
			}},
			Clip: &Clip{X1: 0, Y1: 0, X2: 40, Y2: 40, Inverse: true},
		}
		tags, err := override.Tags()
		require.NoError(t, err)

		for _, tag := range []string{
			"\\1c&H0000FF&", "\\move(1,2,3,4)", "\\fade(255,0,255,0,1000,2000,3000)",
			"\\t(0,2000,1.5,\\fs25)", "\\iclip(0,0,40,40)",
		} {
			assert.Contains(t, tags, tag)
		}
	})

	t.Run("stops a transform cycle at the nesting limit", func(t *testing.T) {
		t.Parallel()

		recursive := &Override{}
		recursive.Transform = []Transform{{Override: recursive}}

		_, err := recursive.Tags()
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
	})
}

func TestExtendedOverrideTags(t *testing.T) {
	t.Parallel()

	t.Run("encodes every extended tag", func(t *testing.T) {
		t.Parallel()

		override := Override{
			ResetStyle: new("Default"), Spacing: new(2.5), Encoding: EncodingDefault.Ptr(),
			BorderX: new(3.0), BorderY: new(4.0), ShadowX: new(-2.0), ShadowY: new(2.0), BlurEdges: new(1),
			DrawingMode: new(1), DrawingBaseline: new(-3.0),
			Karaoke:      &Karaoke{Mode: KaraokeSweep, Duration: 350 * time.Millisecond},
			KaraokeStart: new(100 * time.Millisecond),
			SimpleFade:   &SimpleFade{In: 200 * time.Millisecond, Out: 300 * time.Millisecond},
			VectorClip:   &VectorClip{Drawing: "m 0 0 l 100 0 100 100", Inverse: true},
		}
		tags, err := override.Tags()
		require.NoError(t, err)

		for _, want := range []string{
			`\rDefault`, `\fsp2.5`, `\fe1`, `\xbord3`, `\ybord4`, `\xshad-2`, `\yshad2`, `\be1`, `\p1`,
			`\pbo-3`, `\kf35`, `\kt10`, `\fad(200,300)`, `\iclip(1,m 0 0 l 100 0 100 100)`,
		} {
			assert.Contains(t, tags, want)
		}
	})

	t.Run("accepts every karaoke mode", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			mode KaraokeMode
			want string
		}{
			{name: "default", mode: "", want: `\k0`},
			{name: "instant", mode: KaraokeInstant, want: `\k0`},
			{name: "outline", mode: KaraokeOutline, want: `\ko0`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				tags, err := (&Override{Karaoke: &Karaoke{Mode: tc.mode}}).Tags()
				require.NoError(t, err)
				assert.Equal(t, "{"+tc.want+"}", tags)
			})
		}
	})
}
