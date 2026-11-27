package ass

import (
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPointerHelpersCopyValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ColorWhite, *ColorWhite.Ptr())
	assert.Equal(t, AlphaHalf, *AlphaHalf.Ptr())
	assert.Equal(t, BorderStyleOpaqueBox, *BorderStyleOpaqueBox.Ptr())
	assert.Equal(t, EffectBanner, *EffectBanner.Ptr())
	assert.Equal(t, AlignmentTopRight, *AlignmentTopRight.Ptr())
	assert.Equal(t, EncodingShiftJIS, *EncodingShiftJIS.Ptr())
	assert.Equal(t, BoldWeightBold, *BoldWeightBold.Ptr())
	assert.Equal(t, WrapStyleForced, *WrapStyleForced.Ptr())

	override := Override{PrimaryColor: ColorRed.Ptr(), PrimaryAlpha: AlphaHalf.Ptr()}
	tags, err := override.Tags()
	require.NoError(t, err)
	assert.Equal(t, `{\1c&H0000FF&\1a&H80&}`, tags)
}

func TestParseRGB(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    Color
		wantErr error
	}{
		{name: "short with hash", input: "#abc", want: "&H00CCBBAA"},
		{name: "long without hash", input: "abcdef", want: "&H00EFCDAB"},
		{name: "empty", input: "", wantErr: commerr.ErrInvalidValue},
		{name: "two digits", input: "ab", wantErr: commerr.ErrInvalidValue},
		{name: "four digits", input: "abcd", wantErr: commerr.ErrInvalidValue},
		{name: "not hexadecimal", input: "gggggg", wantErr: commerr.ErrInvalidValue},
		{name: "eight digits", input: "#12345678", wantErr: commerr.ErrInvalidValue},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseRGB(tc.input)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
