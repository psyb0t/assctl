//go:build real

package fonts

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	realTestTimeout = 3 * time.Minute
	minFontBytes    = 1000
	renderWidth     = 540
	renderHeight    = 960
)

// TestRealPresetFontDownloadsAndRendering downloads every preset font from
// its upstream source and renders a caption with each one. It changes HOME and
// XDG_DATA_HOME, so it runs serially.
func TestRealPresetFontDownloadsAndRendering(t *testing.T) {
	if os.Getenv("ASSCTL_TEST_REAL_FONTS") != "1" {
		t.Skip("run make test-real for upstream font downloads")
	}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ctx, cancel := context.WithTimeout(t.Context(), realTestTimeout)
	t.Cleanup(cancel)

	directory, err := Install(ctx, ass.Presets())
	require.NoError(t, err)

	// Every verified file is already present, so a canceled context must not matter.
	offline, stop := context.WithCancel(t.Context())
	stop()
	_, err = Install(offline, ass.Presets())
	require.NoError(t, err)

	for _, preset := range ass.Presets() {
		t.Run(preset.Name, func(t *testing.T) {
			t.Parallel()

			source, err := builtInFonts.source(preset)
			require.NoError(t, err)
			path := filepath.Join(directory, source.font.name)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Greater(t, len(data), minFontBytes, path)

			postscript, err := exec.CommandContext(ctx, "fc-scan", "--format", "%{postscriptname}", path).Output()
			require.NoError(t, err)
			require.NotEmpty(t, postscript)

			script := preset.Captions(ass.CaptionSpec{
				Width:  renderWidth,
				Height: renderHeight,
				Dialogues: []ass.Dialogue{
					{End: 2 * time.Second, Text: "REAL PRESET FONT"},
				},
			})
			assFile := filepath.Join(t.TempDir(), "caption.ass")
			require.NoError(t, script.WriteToFile(assFile))

			command := exec.CommandContext(
				ctx, "ffmpeg", "-hide_banner", "-loglevel", "info", "-filter_threads", "1",
				"-f", "lavfi", "-i", "color=c=black:s=540x960:d=1",
				"-vf", "ass="+assFile+":fontsdir="+directory,
				"-frames:v", "1", "-threads", "1", "-f", "image2pipe", "-c:v", "png", "-",
			)
			var diagnostics bytes.Buffer
			command.Stderr = &diagnostics
			frame, err := command.Output()
			require.NoError(t, err, diagnostics.String())
			require.Greater(t, len(frame), minFontBytes)

			wanted := strings.TrimSpace(string(postscript))
			selected := false
			for line := range strings.SplitSeq(diagnostics.String(), "\n") {
				if strings.Contains(line, "fontselect:") && strings.Contains(line, wanted) {
					selected = true
				}
			}
			assert.True(t, selected, "renderer used a fallback instead of %s:\n%s", wanted, diagnostics.String())
		})
	}
}
