package ass

import (
	"math"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideRejectsInvalidCombinationsAndValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		override Override
	}{
		{name: "position with movement", override: Override{Position: &Position{}, Movement: &Movement{}}},
		{name: "NaN position", override: Override{Position: &Position{X: math.NaN()}}},
		{name: "infinite origin", override: Override{Origin: &Position{Y: math.Inf(1)}}},
		{name: "movement with only a start time", override: Override{Movement: &Movement{T1: new(time.Second)}}},
		{
			name:     "movement ending before it starts",
			override: Override{Movement: &Movement{T1: new(time.Second), T2: new(time.Duration(0))}},
		},
		{name: "infinite movement target", override: Override{Movement: &Movement{X2: math.Inf(1)}}},
		{name: "alignment above range", override: Override{Alignment: new(Alignment(10))}},
		{name: "wrap style above range", override: Override{WrapStyle: new(WrapStyle(4))}},
		{name: "negative border", override: Override{BorderX: new(-1.0)}},
		{name: "negative edge blur", override: Override{BlurEdges: new(-1)}},
		{name: "negative drawing mode", override: Override{DrawingMode: new(-1)}},
		{name: "reset style with delimiter", override: Override{ResetStyle: new("bad}\\pos(0,0)")}},
		{name: "negative karaoke start", override: Override{KaraokeStart: new(-time.Second)}},
		{name: "unknown karaoke mode", override: Override{Karaoke: &Karaoke{Mode: "invalid"}}},
		{name: "negative karaoke duration", override: Override{Karaoke: &Karaoke{Duration: -time.Second}}},
		{name: "negative simple fade", override: Override{SimpleFade: &SimpleFade{In: -time.Second}}},
		{name: "fade alpha above range", override: Override{Fade: &Fade{A1: 256}}},
		{name: "unordered fade times", override: Override{Fade: &Fade{T1: time.Second}}},
		{name: "fade with simple fade", override: Override{Fade: &Fade{}, SimpleFade: &SimpleFade{}}},
		{name: "reversed clip", override: Override{Clip: &Clip{X2: -1}}},
		{name: "NaN clip", override: Override{Clip: &Clip{X2: math.NaN()}}},
		{name: "clip with vector clip", override: Override{Clip: &Clip{}, VectorClip: &VectorClip{}}},
		{name: "negative vector clip scale", override: Override{VectorClip: &VectorClip{Drawing: "m 0 0", Scale: -1}}},
		{name: "non-numeric drawing coordinate", override: Override{VectorClip: &VectorClip{Drawing: "m 0 bad"}}},
		{name: "drawing not starting with move", override: Override{VectorClip: &VectorClip{Drawing: "l 0 0"}}},
		{name: "bezier with too few coordinates", override: Override{VectorClip: &VectorClip{Drawing: "m 0 0 b 1 2"}}},
		{name: "NaN drawing coordinate", override: Override{VectorClip: &VectorClip{Drawing: "m 0 0 l NaN 1"}}},
		{name: "line feed in drawing", override: Override{VectorClip: &VectorClip{Drawing: "m 0 0\nl 100 0 100 100"}}},
		{name: "carriage return in drawing", override: Override{VectorClip: &VectorClip{Drawing: "m 0 0\rl 100 0 100 100"}}},
		{
			name: "transform animating a position",
			override: Override{Transform: []Transform{
				{End: time.Second, Override: &Override{Position: &Position{}}},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tc.override.Tags()
			require.ErrorIs(t, err, commerr.ErrInvalidValue)
		})
	}
}

func TestOverrideAcceptsValidDrawings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		drawing string
	}{
		{name: "bezier", drawing: "m 0 0 b 1 2 3 4 5 6"},
		{name: "spline with point and close", drawing: "n 1 2 s 1 2 3 4 5 6 p 7 8 c"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			override := &Override{VectorClip: &VectorClip{Scale: 2, Drawing: tc.drawing}}
			tags, err := override.Tags()
			require.NoError(t, err)
			assert.Equal(t, "{\\clip(2,"+tc.drawing+")}", tags)
		})
	}
}
