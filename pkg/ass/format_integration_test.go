package ass

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func renderFrame(t *testing.T, script Script, at time.Duration, width, height int) image.Image {
	t.Helper()

	return renderFrameWithEnv(t, script, at, width, height, nil)
}

// renderFrameWithEnv renders like renderFrame with extra KEY=value entries added
// to the ffmpeg environment.
func renderFrameWithEnv(t *testing.T, script Script, at time.Duration, width, height int, env []string) image.Image {
	t.Helper()

	file := filepath.Join(t.TempDir(), "captions.ass")
	require.NoError(t, script.WriteToFile(file))

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)

	command := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error", "-threads", "1", "-filter_threads", "1",
		"-f", "lavfi", "-i", fmt.Sprintf("color=c=black:s=%dx%d:r=20:d=3", width, height),
		"-vf", "ass="+file, "-ss", fmt.Sprintf("%.2f", at.Seconds()),
		"-frames:v", "1", "-threads", "1", "-f", "image2pipe", "-c:v", "png", "-",
	)
	if len(env) > 0 {
		command.Env = append(os.Environ(), env...)
	}
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics

	data, err := command.Output()
	require.NoError(t, err, diagnostics.String())

	frame, err := png.Decode(bytes.NewReader(data))
	require.NoError(t, err)

	return frame
}

func visibleBounds(frame image.Image) image.Rectangle {
	box := image.Rectangle{}
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
			r, g, b, _ := frame.At(x, y).RGBA()
			if r>>8+g>>8+b>>8 < 90 {
				continue
			}
			pixel := image.Rect(x, y, x+1, y+1)
			box = box.Union(pixel)
		}
	}

	return box
}

func drawingScript(override *Override) Script {
	script := NewScript("Drawing", 640, 360)
	script.AddStyle(Style{
		Name: "Default", FontName: FontDejaVuSans, FontSize: 64, ScaleX: 100, ScaleY: 100,
		Alignment: AlignmentTopLeft, PrimaryColor: ColorRed,
	})
	script.AddEvent(Event{
		Style: "Default", End: 2 * time.Second, Override: override,
		Text: "m 0 0 l 100 0 100 100 0 100",
	})

	return *script
}
