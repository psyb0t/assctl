package burn

import (
	"bytes"
	"context"
	"encoding/json"
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

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	renderTimeout = 2 * time.Minute
	// textPixels is how many bright pixels prove the caption was drawn on a
	// black frame.
	textPixels    = 200
	brightChannel = 0xc000
	chapterFile   = ";FFMETADATA1\n[CHAPTER]\nTIMEBASE=1/1000\nSTART=0\nEND=1000\ntitle=One\n" +
		"[CHAPTER]\nTIMEBASE=1/1000\nSTART=1000\nEND=2000\ntitle=Two\n"
)

func runTool(t *testing.T, name string, args ...string) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), renderTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, name, args...)
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	require.NoError(t, err, "%s %s: %s", name, strings.Join(args, " "), diagnostics.String())

	return output
}

type probedStream struct {
	CodecType      string `json:"codec_type"`
	CodecName      string `json:"codec_name"`
	CodecTag       string `json:"codec_tag_string"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	PixelFormat    string `json:"pix_fmt"`
	FrameRate      string `json:"r_frame_rate"`
	ColorSpace     string `json:"color_space"`
	ColorTransfer  string `json:"color_transfer"`
	ColorPrimaries string `json:"color_primaries"`
	SideData       []struct {
		Rotation int `json:"rotation"`
	} `json:"side_data_list"`
}

type probed struct {
	Streams  []probedStream `json:"streams"`
	Chapters []struct {
		Tags map[string]string `json:"tags"`
	} `json:"chapters"`
	Format struct {
		Tags map[string]string `json:"tags"`
	} `json:"format"`
}

func probeFile(t *testing.T, path string) probed {
	t.Helper()

	var result probed
	require.NoError(t, json.Unmarshal(runTool(t, "ffprobe", "-v", "error",
		"-show_streams", "-show_chapters", "-show_format", "-of", "json", path), &result))

	return result
}

// streamDigest returns the MD5 of one stream's packets, copied without decoding.
func streamDigest(t *testing.T, path, stream string) string {
	t.Helper()

	return string(runTool(t, "ffmpeg", "-v", "error", "-i", path, "-map", stream, "-c", "copy", "-f", "md5", "-"))
}

func brightPixels(t *testing.T, path string) int {
	t.Helper()

	frame := runTool(t, "ffmpeg", "-v", "error", "-ss", "1", "-i", path,
		"-frames:v", "1", "-f", "image2pipe", "-c:v", "png", "-")
	picture, err := png.Decode(bytes.NewReader(frame))
	require.NoError(t, err)
	count := 0
	bounds := picture.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := picture.At(x, y).RGBA()
			if r > brightChannel && g > brightChannel && b > brightChannel {
				count++
			}
		}
	}

	return count
}

func writeCaptions(t *testing.T, directory string, width, height int) string {
	t.Helper()

	preset, err := ass.LookupPreset(ass.PresetBlackGold)
	require.NoError(t, err)
	script := preset.Captions(ass.CaptionSpec{
		Width: width, Height: height,
		Dialogues: []ass.Dialogue{{End: 2 * time.Second, Text: "BURNED IN"}},
	})
	path := filepath.Join(directory, "it's: [captions], v1; final.ass")
	require.NoError(t, script.WriteToFile(path))

	return path
}

func TestRenderBurnKeepsFormatAndStreams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		extension string
		video     []string
		audio     []string
		chapters  bool
		cover     bool
		tag       string
	}{
		{
			name: "h264 with audio, chapters, cover art and colors", extension: ".mp4", chapters: true, cover: true,
			video: []string{
				"-c:v", "libx264", "-pix_fmt", "yuv420p",
				"-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709",
			},
			audio: []string{"-c:a", "aac"},
		},
		{
			name: "10-bit hevc tagged hvc1", extension: ".mp4", tag: tagHVC1,
			video: []string{"-c:v", "libx265", "-pix_fmt", "yuv420p10le", "-tag:v", tagHVC1, "-x265-params", "log-level=error"},
			audio: []string{"-c:a", "aac"},
		},
		{
			name: "vp9 with opus", extension: ".webm",
			video: []string{"-c:v", "libvpx-vp9", "-pix_fmt", "yuv420p", "-b:v", "0", "-crf", "30"},
			audio: []string{"-c:a", "libopus"},
		},
		{
			name: "av1 in matroska", extension: ".mkv",
			video: []string{"-c:v", "libsvtav1", "-preset", "12", "-pix_fmt", "yuv420p"},
			audio: []string{"-c:a", "libopus"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			input := filepath.Join(directory, "input"+tc.extension)
			args := []string{
				"-v", "error", "-y",
				"-f", "lavfi", "-i", "color=c=black:s=320x568:r=24:d=2",
				"-f", "lavfi", "-i", "sine=d=2",
			}
			maps := []string{"-map", "0", "-map", "1"}
			if tc.chapters {
				metadata := filepath.Join(directory, "chapters.txt")
				require.NoError(t, os.WriteFile(metadata, []byte(chapterFile), 0o600))
				args = append(args, "-i", metadata)
				maps = append(maps, "-map_chapters", "2")
			}
			if tc.cover {
				cover := filepath.Join(directory, "cover.png")
				runTool(t, "ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "color=c=red:s=64x64", "-frames:v", "1", cover)
				args = append(args, "-i", cover)
				maps = append(maps, "-map", "3")
			}
			args = append(args, maps...)
			args = append(args, "-metadata", "title=Source clip")
			args = append(args, tc.video...)
			args = append(args, tc.audio...)
			if tc.cover {
				args = append(args, "-c:v:1", "png", "-pix_fmt:v:1", "rgb24", "-disposition:v:1", "attached_pic")
			}
			runTool(t, "ffmpeg", append(args, input)...)
			output := filepath.Join(directory, "output"+tc.extension)

			require.NoError(t, Burn(t.Context(), Options{
				Input: input, Subtitles: writeCaptions(t, directory, 320, 568), Output: output,
			}))

			before, after := probeFile(t, input), probeFile(t, output)
			require.Len(t, after.Streams, len(before.Streams))
			source, burned := before.Streams[0], after.Streams[0]
			assert.Equal(t, source.CodecName, burned.CodecName)
			assert.Equal(t, source.Width, burned.Width)
			assert.Equal(t, source.Height, burned.Height)
			assert.Equal(t, source.PixelFormat, burned.PixelFormat)
			assert.Equal(t, source.FrameRate, burned.FrameRate)
			assert.Equal(t, source.ColorSpace, burned.ColorSpace)
			assert.Equal(t, source.ColorTransfer, burned.ColorTransfer)
			assert.Equal(t, source.ColorPrimaries, burned.ColorPrimaries)
			if tc.tag != "" {
				assert.Equal(t, tc.tag, burned.CodecTag)
			}
			for index := 1; index < len(before.Streams); index++ {
				assert.Equal(t, before.Streams[index].CodecName, after.Streams[index].CodecName)
				stream := fmt.Sprintf("0:%d", index)
				assert.Equal(t, streamDigest(t, input, stream), streamDigest(t, output, stream), "stream %d changed", index)
			}
			assert.Len(t, after.Chapters, len(before.Chapters))
			assert.Equal(t, "Source clip", after.Format.Tags["title"])
			assert.Zero(t, brightPixels(t, input))
			assert.Greater(t, brightPixels(t, output), textPixels)
		})
	}
}

func TestRenderBurnDrawsOnRotatedVideoUpright(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	landscape := filepath.Join(directory, "landscape.mp4")
	rotated := filepath.Join(directory, "rotated.mp4")
	runTool(t, "ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "color=c=black:s=568x320:r=24:d=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", landscape)
	runTool(t, "ffmpeg", "-v", "error", "-y", "-display_rotation", "90", "-i", landscape, "-c", "copy", rotated)
	source := probeFile(t, rotated).Streams[0]
	require.NotEmpty(t, source.SideData, "the source should carry a rotation")
	output := filepath.Join(directory, "output.mp4")

	require.NoError(t, Burn(t.Context(), Options{
		Input: rotated, Subtitles: writeCaptions(t, directory, 320, 568), Output: output,
	}))

	burned := probeFile(t, output).Streams[0]
	assert.Equal(t, 320, burned.Width)
	assert.Equal(t, 568, burned.Height)
	for _, side := range burned.SideData {
		assert.Zero(t, side.Rotation)
	}
	assert.Greater(t, brightPixels(t, output), textPixels)
}
