//go:build real

package main

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/internal/pkg/fonts"
	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	realEmbedTimeout  = 3 * time.Minute
	realRenderWidth   = 540
	realRenderHeight  = 960
	brightChannelSum  = 3 * 0x7fff
	minBrightPixels   = 50
	realCaptionPeriod = 2 * time.Second
)

// TestRealEmbedPresetFontsRenderWithoutInstalledFonts downloads every preset's
// fonts through captions --embed-preset-fonts, then renders each file with a
// fontconfig that has no font directory, so text can only come from the
// embedded fonts. It changes HOME and XDG_DATA_HOME, so it runs serially.
func TestRealEmbedPresetFontsRenderWithoutInstalledFonts(t *testing.T) {
	if os.Getenv("ASSCTL_TEST_REAL_FONTS") != "1" {
		t.Skip("run make test-real for upstream font downloads")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ctx, cancel := context.WithTimeout(t.Context(), realEmbedTimeout)
	t.Cleanup(cancel)

	directory := t.TempDir()
	spec := filepath.Join(directory, "spec.json")
	require.NoError(t, os.WriteFile(spec, []byte(fmt.Sprintf(
		`{"Width":%d,"Height":%d,"Dialogues":[{"Start":0,"End":%d,"Text":"EMBEDDED FONT"}]}`,
		realRenderWidth, realRenderHeight, realCaptionPeriod.Nanoseconds(),
	)), 0o600))
	fontconfig := filepath.Join(directory, "fonts.conf")
	require.NoError(t, os.WriteFile(fontconfig, []byte(fmt.Sprintf(
		"<?xml version=\"1.0\"?>\n<fontconfig><cachedir>%s</cachedir></fontconfig>\n", directory,
	)), 0o600))

	for _, preset := range ass.Presets() {
		output := filepath.Join(directory, preset.Name+".ass")
		err := run(
			[]string{"captions", "--preset", preset.Name, "--spec", spec, "--embed-preset-fonts", "--output", output},
			strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{},
		)
		require.NoError(t, err, preset.Name)

		content, err := os.ReadFile(output)
		require.NoError(t, err)
		names, err := fonts.FontFiles([]ass.Preset{preset})
		require.NoError(t, err)
		for _, name := range names {
			assert.Contains(t, string(content), "\nfontname: "+name+"\n", preset.Name)
		}

		command := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
			"-f", "lavfi", "-i", fmt.Sprintf("color=c=black:s=%dx%d:d=1", realRenderWidth, realRenderHeight),
			"-vf", "ass="+output, "-frames:v", "1", "-f", "image2pipe", "-c:v", "png", "-")
		command.Env = append(os.Environ(), "FONTCONFIG_FILE="+fontconfig, "XDG_CACHE_HOME="+directory)
		var diagnostics bytes.Buffer
		command.Stderr = &diagnostics
		frame, err := command.Output()
		require.NoError(t, err, diagnostics.String())

		image, err := png.Decode(bytes.NewReader(frame))
		require.NoError(t, err)
		bright := 0
		bounds := image.Bounds()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, _ := image.At(x, y).RGBA()
				if r+g+b > brightChannelSum {
					bright++
				}
			}
		}
		assert.Greater(t, bright, minBrightPixels, "%s drew no text from its embedded fonts", preset.Name)
	}
}
