package burn

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const h264Probe = `{"streams":[` +
	`{"index":0,"codec_type":"video","codec_name":"png","disposition":{"attached_pic":1}},` +
	`{"index":1,"codec_type":"video","codec_name":"h264","codec_tag_string":"avc1","pix_fmt":"yuv420p",` +
	`"color_range":"tv","color_space":"bt709","color_transfer":"bt709","color_primaries":"unknown","disposition":{"attached_pic":0}},` +
	`{"index":2,"codec_type":"audio","codec_name":"aac"},` +
	`{"index":3,"codec_type":"data","codec_name":"bin_data","codec_tag_string":"text"}]}`

// fakeTool writes an executable shell script and returns its path.
func fakeTool(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700))

	return path
}

// probeTool prints output as ffprobe's JSON.
func probeTool(t *testing.T, output string) string {
	t.Helper()

	data := filepath.Join(t.TempDir(), "probe.json")
	require.NoError(t, os.WriteFile(data, []byte(output), 0o600))

	return fakeTool(t, "ffprobe", "cat '"+data+"'")
}

// recordingFFmpeg saves its arguments, one per line, and creates its last
// argument without the file: prefix.
func recordingFFmpeg(t *testing.T) (string, string) {
	t.Helper()

	record := filepath.Join(t.TempDir(), "args")
	tool := fakeTool(t, "ffmpeg", `for arg in "$@"; do printf '%s\n' "$arg"; done > '`+record+`'
for arg in "$@"; do last="$arg"; done
printf video > "${last#file:}"`)

	return tool, record
}

// files creates an input video and a subtitle file and returns options for
// them with an output path in the same directory.
func files(t *testing.T) Options {
	t.Helper()

	directory := t.TempDir()
	input := filepath.Join(directory, "in.mp4")
	subtitles := filepath.Join(directory, "captions.ass")
	require.NoError(t, os.WriteFile(input, []byte("source"), 0o600))
	require.NoError(t, os.WriteFile(subtitles, []byte("[Script Info]\n"), 0o600))

	return Options{Input: input, Subtitles: subtitles, Output: filepath.Join(directory, "out.mp4")}
}

func TestFilterValueEscapesBothLevels(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path string
		want string
	}{
		{name: "plain", path: "/work/captions.ass", want: "/work/captions.ass"},
		{name: "colon", path: "C:/a.ass", want: `C\\:/a.ass`},
		{name: "quote", path: "it's.ass", want: `it\\\'s.ass`},
		{name: "backslash", path: `a\b.ass`, want: `a\\\\b.ass`},
		{name: "graph separators", path: "[a],b;c.ass", want: `\[a\]\,b\;c.ass`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, filterValue(tc.path))
		})
	}
}

func TestMainVideo(t *testing.T) {
	t.Parallel()

	cover := stream{CodecType: codecTypeVideo, CodecName: "png"}
	cover.Disposition.AttachedPic = 1
	audio := stream{CodecType: "audio", CodecName: "aac"}
	main := stream{CodecType: codecTypeVideo, CodecName: "h264"}

	testCases := []struct {
		name        string
		streams     []stream
		wantOrdinal int
		wantErr     error
	}{
		{name: "first stream", streams: []stream{main, audio}, wantOrdinal: 0},
		{name: "after cover art", streams: []stream{audio, cover, main}, wantOrdinal: 1},
		{name: "audio only", streams: []stream{audio}, wantErr: commerr.ErrNotFound},
		{name: "cover art only", streams: []stream{cover}, wantErr: commerr.ErrNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := mainVideo(tc.streams)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantOrdinal, got.ordinal)
			assert.Equal(t, "h264", got.CodecName)
		})
	}
}

func TestBurnArgs(t *testing.T) {
	t.Parallel()

	chapters := stream{Index: 3, CodecType: codecTypeData, CodecName: codecBinData, CodecTag: tagText}
	timecode := stream{Index: 4, CodecType: codecTypeData, CodecName: codecBinData, CodecTag: "tmcd"}

	t.Run("h264 keeps format and skips the chapter track", func(t *testing.T) {
		t.Parallel()

		main := video{ordinal: 1, stream: stream{
			CodecName: "h264", PixelFormat: "yuv420p", ColorRange: "tv", ColorSpace: "bt709",
			ColorTransfer: "bt709", ColorPrimaries: unknownValue,
		}}
		got := burnArgs("in.mp4", "a.ass", []stream{chapters, timecode}, main, encoders["h264"], 18)
		assert.Equal(t, []string{
			"-nostdin", "-hide_banner", "-loglevel", "error", "-stats", "-y",
			"-i", "file:in.mp4", "-map", "0", "-map", "-0:3",
			"-map_metadata", "0", "-map_chapters", "0", "-c", "copy",
			"-filter:v:1", "ass=filename=a.ass",
			"-c:v:1", "libx264", "-crf:v:1", "18", "-preset:v:1", "medium",
			"-fps_mode:v:1", "passthrough",
			"-pix_fmt:v:1", "yuv420p", "-color_range:v:1", "tv", "-colorspace:v:1", "bt709", "-color_trc:v:1", "bt709",
		}, got)
	})

	t.Run("hevc keeps the hvc1 tag", func(t *testing.T) {
		t.Parallel()

		main := video{stream: stream{CodecName: "hevc", CodecTag: tagHVC1, PixelFormat: "yuv420p10le"}}
		got := burnArgs("in.mp4", "a.ass", nil, main, encoders["hevc"], 20)
		assert.Equal(t, []string{"-tag:v:0", "hvc1"}, got[len(got)-2:])
		assert.Contains(t, strings.Join(got, " "), "-x265-params:v:0 log-level=error")
		assert.Contains(t, strings.Join(got, " "), "-pix_fmt:v:0 yuv420p10le")
	})

	t.Run("vp9 uses constant quality", func(t *testing.T) {
		t.Parallel()

		got := burnArgs("in.webm", "a.ass", nil, video{stream: stream{CodecName: "vp9"}}, encoders["vp9"], 20)
		assert.Contains(t, strings.Join(got, " "), "-c:v:0 libvpx-vp9 -crf:v:0 20 -b:v:0 0 -row-mt:v:0 1")
	})
}

func TestBurnWritesOutputThroughFFmpeg(t *testing.T) {
	t.Parallel()

	opts := files(t)
	ffmpeg, record := recordingFFmpeg(t)
	opts.FFmpeg = ffmpeg
	opts.FFprobe = probeTool(t, h264Probe)
	crf := 12
	opts.CRF = &crf
	var diagnostics bytes.Buffer
	opts.Diagnostics = &diagnostics

	require.NoError(t, Burn(t.Context(), opts))

	content, err := os.ReadFile(opts.Output)
	require.NoError(t, err)
	assert.Equal(t, "video", string(content))
	args, err := os.ReadFile(record)
	require.NoError(t, err)
	assert.Contains(t, string(args), "\n-crf:v:1\n12\n")
	assert.Contains(t, string(args), "\n-map\n-0:3\n")
	assert.Contains(t, string(args), "\nfile:"+opts.Input+"\n")
	entries, err := os.ReadDir(filepath.Dir(opts.Output))
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasPrefix(entry.Name(), ".assctl-burn-"), "temporary file %s left behind", entry.Name())
	}
}

func TestBurnRejects(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")
	testCases := []struct {
		name    string
		change  func(t *testing.T, opts *Options)
		wantErr error
		wantMsg string
	}{
		{name: "missing input path", change: func(_ *testing.T, opts *Options) { opts.Input = "" }, wantErr: commerr.ErrInvalidArgument},
		{name: "missing subtitles path", change: func(_ *testing.T, opts *Options) { opts.Subtitles = "" }, wantErr: commerr.ErrInvalidArgument},
		{name: "missing output path", change: func(_ *testing.T, opts *Options) { opts.Output = "" }, wantErr: commerr.ErrInvalidArgument},
		{name: "output without extension", change: func(_ *testing.T, opts *Options) { opts.Output += ".d/out" }, wantErr: commerr.ErrInvalidArgument},
		{name: "input does not exist", change: func(_ *testing.T, opts *Options) { opts.Input = missing }, wantErr: os.ErrNotExist},
		{name: "subtitles do not exist", change: func(_ *testing.T, opts *Options) { opts.Subtitles = missing }, wantErr: os.ErrNotExist},
		{name: "output is the input", change: func(_ *testing.T, opts *Options) { opts.Output = opts.Input }, wantErr: commerr.ErrInvalidArgument},
		{name: "ffmpeg not found", change: func(_ *testing.T, opts *Options) { opts.FFmpeg = missing }, wantErr: commerr.ErrNotFound},
		{name: "ffprobe not found", change: func(_ *testing.T, opts *Options) { opts.FFprobe = missing }, wantErr: commerr.ErrNotFound},
		{
			name: "ffprobe fails",
			change: func(t *testing.T, opts *Options) {
				opts.FFprobe = fakeTool(t, "ffprobe", "echo broken file >&2; exit 1")
			},
			wantMsg: "broken file",
		},
		{
			name:    "ffprobe prints no JSON",
			change:  func(t *testing.T, opts *Options) { opts.FFprobe = probeTool(t, "not json") },
			wantErr: commerr.ErrInvalidValue,
		},
		{
			name:    "no video stream",
			change:  func(t *testing.T, opts *Options) { opts.FFprobe = probeTool(t, `{"streams":[{"codec_type":"audio"}]}`) },
			wantErr: commerr.ErrNotFound,
		},
		{
			name: "unsupported codec",
			change: func(t *testing.T, opts *Options) {
				opts.FFprobe = probeTool(t, `{"streams":[{"codec_type":"video","codec_name":"prores"}]}`)
			},
			wantErr: commerr.ErrUnsupported,
		},
		{
			name: "CRF below the encoder range",
			change: func(_ *testing.T, opts *Options) {
				crf := -1
				opts.CRF = &crf
			},
			wantErr: commerr.ErrInvalidArgument,
		},
		{
			name: "CRF above the encoder range",
			change: func(_ *testing.T, opts *Options) {
				crf := 52
				opts.CRF = &crf
			},
			wantErr: commerr.ErrInvalidArgument,
		},
		{
			name: "ffmpeg fails",
			change: func(t *testing.T, opts *Options) {
				opts.FFmpeg = fakeTool(t, "ffmpeg", "echo encoder exploded >&2; exit 1")
			},
			wantMsg: "encoder exploded",
		},
		{
			name: "output directory is missing",
			change: func(_ *testing.T, opts *Options) {
				opts.Output = filepath.Join(filepath.Dir(opts.Output), "missing", "out.mp4")
			},
			wantErr: os.ErrNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := files(t)
			opts.FFmpeg, _ = recordingFFmpeg(t)
			opts.FFprobe = probeTool(t, h264Probe)
			require.NoError(t, os.WriteFile(opts.Output, []byte("existing"), 0o600))
			tc.change(t, &opts)

			err := Burn(t.Context(), opts)
			require.Error(t, err)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			}
			assert.Contains(t, err.Error(), tc.wantMsg)

			if content, readErr := os.ReadFile(filepath.Join(filepath.Dir(opts.Input), "out.mp4")); readErr == nil {
				assert.Equal(t, "existing", string(content), "a failed burn changed the existing output")
			}
		})
	}
}

func TestBurnRemovesTemporaryOutputWhenFFmpegFails(t *testing.T) {
	t.Parallel()

	opts := files(t)
	opts.FFprobe = probeTool(t, h264Probe)
	opts.FFmpeg = fakeTool(t, "ffmpeg", `for arg in "$@"; do last="$arg"; done
printf partial > "${last#file:}"; exit 1`)

	require.Error(t, Burn(t.Context(), opts))

	entries, err := os.ReadDir(filepath.Dir(opts.Output))
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.ElementsMatch(t, []string{"in.mp4", "captions.ass"}, names)
}

func TestTailBufferKeepsTheEnd(t *testing.T) {
	t.Parallel()

	buffer := &tailBuffer{limit: 4}
	written, err := buffer.Write([]byte("abc"))
	require.NoError(t, err)
	assert.Equal(t, 3, written)
	_, err = buffer.Write([]byte("defg"))
	require.NoError(t, err)
	assert.Equal(t, "defg", buffer.String())
}
