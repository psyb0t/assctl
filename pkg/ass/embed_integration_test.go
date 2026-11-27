package ass

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const installedFontFile = "/usr/share/fonts/dejavu/DejaVuSans.ttf"

// emptyFontconfigEnv points fontconfig at a configuration without any font
// directory, so libass finds no installed font and can only use embedded ones.
func emptyFontconfigEnv(t *testing.T) []string {
	t.Helper()

	dir := t.TempDir()
	config := filepath.Join(dir, "fonts.conf")
	content := fmt.Sprintf("<?xml version=\"1.0\"?>\n<fontconfig><cachedir>%s</cachedir></fontconfig>\n", dir)
	require.NoError(t, os.WriteFile(config, []byte(content), 0o600))

	return []string{"FONTCONFIG_FILE=" + config, "XDG_CACHE_HOME=" + dir}
}

func TestRenderEmbeddedFontNeedsNoInstalledFont(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(installedFontFile)
	require.NoError(t, err)

	script := NewScript("Embedded", 640, 360)
	script.AddStyle(Style{
		Name: "Default", FontName: FontDejaVuSans, FontSize: 64, ScaleX: 100, ScaleY: 100,
		Alignment: AlignmentTopLeft, PrimaryColor: ColorWhite,
	})
	script.AddEvent(Event{Style: "Default", End: 2 * time.Second, Text: "EMBEDDED"})

	env := emptyFontconfigEnv(t)
	bare := visibleBounds(renderFrameWithEnv(t, *script, time.Second, 640, 360, env))
	assert.True(t, bare.Empty(), "text drawn without any font: %v", bare)

	script.Fonts = []EmbeddedFont{{Name: "DejaVuSans.ttf", Data: data}}
	embedded := visibleBounds(renderFrameWithEnv(t, *script, time.Second, 640, 360, env))
	require.False(t, embedded.Empty(), "embedded font drew nothing")

	script.Fonts = nil
	installed := visibleBounds(renderFrame(t, *script, time.Second, 640, 360))
	assert.Equal(t, installed, embedded, "embedded copy drew different glyphs than the installed font")
}
