package fonts

import (
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

func TestBuiltInFontsSelection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		preset   ass.Preset
		wantFont string
		wantErr  error
	}{
		{name: "regular DejaVu Sans", preset: ass.Preset{Font: ass.FontDejaVuSans}, wantFont: "DejaVuSans.ttf"},
		{
			name:     "bold DejaVu Sans",
			preset:   ass.Preset{Font: ass.FontDejaVuSans, Bold: true},
			wantFont: "DejaVuSans-Bold.ttf",
		},
		{name: "regular DejaVu Serif", preset: ass.Preset{Font: ass.FontDejaVuSerif}, wantFont: "DejaVuSerif.ttf"},
		{
			name:     "bold DejaVu Serif has no bold file",
			preset:   ass.Preset{Font: ass.FontDejaVuSerif, Bold: true},
			wantFont: "DejaVuSerif.ttf",
		},
		{
			name:     "regular DejaVu Sans Mono",
			preset:   ass.Preset{Font: ass.FontDejaVuSansMono},
			wantFont: "DejaVuSansMono-Bold.ttf",
		},
		{
			name:     "bold DejaVu Sans Mono",
			preset:   ass.Preset{Font: ass.FontDejaVuSansMono, Bold: true},
			wantFont: "DejaVuSansMono-Bold.ttf",
		},
		{name: "Anton", preset: ass.Preset{Font: ass.FontAnton}, wantFont: "Anton-Regular.ttf"},
		{
			name:     "bold Anton has no bold file",
			preset:   ass.Preset{Font: ass.FontAnton, Bold: true},
			wantFont: "Anton-Regular.ttf",
		},
		{
			name:     "Barlow Condensed",
			preset:   ass.Preset{Font: ass.FontBarlowCondensedExtraBold},
			wantFont: "BarlowCondensed-ExtraBold.ttf",
		},
		{
			name:     "TikTok Sans",
			preset:   ass.Preset{Font: ass.FontTikTokSans, Bold: true},
			wantFont: "TikTokSans36pt-Bold.ttf",
		},
		{
			name:     "The Bold Font",
			preset:   ass.Preset{Font: ass.FontTheBold, Bold: true},
			wantFont: "THEBOLDFONT-FREEVERSION.ttf",
		},
		{name: "unknown family", preset: ass.Preset{Font: "Unknown"}, wantErr: commerr.ErrUnsupported},
		{name: "unknown bold family", preset: ass.Preset{Font: "Unknown", Bold: true}, wantErr: commerr.ErrUnsupported},
		{name: "empty family", preset: ass.Preset{}, wantErr: commerr.ErrUnsupported},
		{
			name:    "family names are case sensitive",
			preset:  ass.Preset{Font: strings.ToLower(ass.FontDejaVuSans)},
			wantErr: commerr.ErrUnsupported,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := builtInFonts.source(tc.preset)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantFont, got.font.name)
			assert.NotEmpty(t, got.license.name)
		})
	}
}

func TestCatalogSource(t *testing.T) {
	t.Parallel()

	regular := fontSource{font: fontAsset{name: "regular.ttf"}}
	bold := fontSource{font: fontAsset{name: "bold.ttf"}}
	boldOnly := fontSource{font: fontAsset{name: "bold-only.ttf"}}
	fonts := catalog{
		regular: map[string]fontSource{"Both": regular},
		bold:    map[string]fontSource{"Both": bold, "Bold Only": boldOnly},
	}

	testCases := []struct {
		name    string
		preset  ass.Preset
		want    fontSource
		wantErr error
	}{
		{name: "regular preset", preset: ass.Preset{Font: "Both"}, want: regular},
		{name: "bold preset", preset: ass.Preset{Font: "Both", Bold: true}, want: bold},
		{name: "bold preset without regular entry", preset: ass.Preset{Font: "Bold Only", Bold: true}, want: boldOnly},
		{name: "regular preset without regular entry", preset: ass.Preset{Font: "Bold Only"}, wantErr: commerr.ErrUnsupported},
		{name: "unknown family", preset: ass.Preset{Font: "Missing"}, wantErr: commerr.ErrUnsupported},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := fonts.source(tc.preset)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("empty catalog", func(t *testing.T) {
		t.Parallel()

		_, err := catalog{}.source(ass.Preset{Font: "Both"})
		require.ErrorIs(t, err, commerr.ErrUnsupported)
	})
}

func TestBuiltInFontsCoverEveryPreset(t *testing.T) {
	t.Parallel()

	for _, preset := range ass.Presets() {
		t.Run(preset.Name, func(t *testing.T) {
			t.Parallel()

			got, err := builtInFonts.source(preset)
			require.NoError(t, err)
			assert.Equal(t, ".ttf", filepath.Ext(got.font.name))
			assert.NotEmpty(t, got.license.name)
		})
	}
}

func TestBuiltInFontAssetsAreWellFormed(t *testing.T) {
	t.Parallel()

	var sources []fontSource
	for _, group := range []map[string]fontSource{builtInFonts.regular, builtInFonts.bold} {
		for _, source := range group {
			sources = append(sources, source)
		}
	}
	require.NotEmpty(t, sources)

	installed := make(map[string]fontAsset)
	archives := make(map[string]string)
	for _, source := range sources {
		assert.NotEqual(t, source.font.name, source.license.name)
		assert.Equal(t, ".ttf", filepath.Ext(source.font.name), source.font.name)

		for _, asset := range []fontAsset{source.font, source.license} {
			assertWellFormedAsset(t, asset)

			if previous, ok := installed[asset.name]; ok {
				assert.Equal(t, previous, asset, "conflicting definitions of %s", asset.name)
			}
			installed[asset.name] = asset

			if checksum, ok := archives[asset.url]; ok {
				assert.Equal(t, checksum, asset.checksum, "one URL, two checksums: %s", asset.url)
			}
			archives[asset.url] = asset.checksum
		}
	}
}

func assertWellFormedAsset(t *testing.T, asset fontAsset) {
	t.Helper()

	assert.NotEmpty(t, asset.name)
	assert.Equal(t, filepath.Base(asset.name), asset.name, "name must be a plain file name")
	assert.True(t, strings.HasPrefix(asset.url, "https://"), "URL %s must use HTTPS", asset.url)
	assertSHA256Hex(t, asset.checksum)
	assertSHA256Hex(t, asset.digest())
	assert.Equal(t, asset.member == "", asset.fileChecksum == "", "member and file checksum go together: %s", asset.name)
}

func assertSHA256Hex(t *testing.T, value string) {
	t.Helper()

	decoded, err := hex.DecodeString(value)
	require.NoError(t, err, value)
	assert.Len(t, decoded, 32, value)
	assert.Equal(t, strings.ToLower(value), value)
}

func TestFontAssetDigest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		asset fontAsset
		want  string
	}{
		{name: "direct file", asset: fontAsset{checksum: "download"}, want: "download"},
		{name: "archive member", asset: fontAsset{checksum: "archive", fileChecksum: "member"}, want: "member"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.asset.digest())
		})
	}
}
