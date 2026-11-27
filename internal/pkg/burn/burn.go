// Package burn draws an ASS subtitle file into a video with FFmpeg. Only the
// main video stream is re-encoded, with the encoder family of its own codec
// and its pixel format, frame timing and color tags. Every other stream,
// the chapters and the metadata are copied unchanged.
package burn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

const (
	ffmpegName  = "ffmpeg"
	ffprobeName = "ffprobe"

	codecTypeVideo = "video"
	codecTypeData  = "data"
	codecBinData   = "bin_data"
	tagText        = "text"
	tagHVC1        = "hvc1"
	unknownValue   = "unknown"
	tempPattern    = ".assctl-burn-*"
	// fileProtocol keeps FFmpeg from reading a name such as "clip:1.mp4" as a
	// protocol or a leading "-" as an option.
	fileProtocol = "file:"

	// stderrTail is how much of FFmpeg's error output a failure carries.
	stderrTail = 4 << 10
)

// encoder re-encodes one source codec. quality holds the arguments after the
// CRF value, written once per call with the output stream specifier.
type encoder struct {
	name       string
	defaultCRF int
	minCRF     int
	maxCRF     int
	quality    []string
}

// encoders pick a visually lossless default CRF for each supported codec.
var encoders = map[string]encoder{
	"h264": {name: "libx264", defaultCRF: 18, minCRF: 0, maxCRF: 51, quality: []string{"-preset", "medium"}},
	"hevc": {
		name: "libx265", defaultCRF: 20, minCRF: 0, maxCRF: 51,
		quality: []string{"-preset", "medium", "-x265-params", "log-level=error"},
	},
	"vp9": {name: "libvpx-vp9", defaultCRF: 20, minCRF: 0, maxCRF: 63, quality: []string{"-b", "0", "-row-mt", "1"}},
	"av1": {name: "libsvtav1", defaultCRF: 23, minCRF: 1, maxCRF: 63, quality: []string{"-preset", "6"}},
}

// Options selects the files and the optional quality override for Burn.
type Options struct {
	// Input is the source video, Subtitles the ASS file and Output the new
	// video. Output must differ from Input and name its container with its
	// extension.
	Input     string
	Subtitles string
	Output    string
	// CRF overrides the encoder's default quality when it is not nil.
	CRF *int
	// FFmpeg and FFprobe name the executables; empty means the ones on PATH.
	FFmpeg  string
	FFprobe string
	// Diagnostics receives FFmpeg's progress and errors; nil discards them.
	Diagnostics io.Writer
}

type stream struct {
	Index          int    `json:"index"`
	CodecType      string `json:"codec_type"`
	CodecName      string `json:"codec_name"`
	CodecTag       string `json:"codec_tag_string"`
	PixelFormat    string `json:"pix_fmt"`
	ColorRange     string `json:"color_range"`
	ColorSpace     string `json:"color_space"`
	ColorTransfer  string `json:"color_transfer"`
	ColorPrimaries string `json:"color_primaries"`
	Disposition    struct {
		AttachedPic int `json:"attached_pic"`
	} `json:"disposition"`
}

// video is the stream to re-encode and its position among the video streams,
// which is also its output stream specifier because every stream is mapped
// in order.
type video struct {
	stream
	ordinal int
}

// Burn writes Output: Input with Subtitles drawn into its main video stream.
// It writes a temporary file beside Output and renames it, so a failure
// leaves no partial output and an existing Output untouched.
func Burn(ctx context.Context, opts Options) error {
	if err := checkPaths(opts); err != nil {
		return ctxerrors.Wrap(err, "check burn files")
	}
	ffmpeg, err := executable(opts.FFmpeg, ffmpegName)
	if err != nil {
		return ctxerrors.Wrap(err, "find ffmpeg")
	}
	ffprobe, err := executable(opts.FFprobe, ffprobeName)
	if err != nil {
		return ctxerrors.Wrap(err, "find ffprobe")
	}
	streams, err := probe(ctx, ffprobe, opts.Input)
	if err != nil {
		return ctxerrors.Wrap(err, "read video streams")
	}
	main, err := mainVideo(streams)
	if err != nil {
		return ctxerrors.Wrap(err, "choose video stream")
	}
	codec, ok := encoders[main.CodecName]
	if !ok {
		return ctxerrors.Wrapf(commerr.ErrUnsupported, "video codec %q; supported codecs are h264, hevc, vp9 and av1", main.CodecName)
	}
	crf := codec.defaultCRF
	if opts.CRF != nil {
		crf = *opts.CRF
		if crf < codec.minCRF || crf > codec.maxCRF {
			return ctxerrors.Wrapf(commerr.ErrInvalidArgument, "CRF %d for %s; use %d to %d", crf, codec.name, codec.minCRF, codec.maxCRF)
		}
	}
	if err := encode(ctx, ffmpeg, opts, burnArgs(opts.Input, opts.Subtitles, streams, main, codec, crf)); err != nil {
		return ctxerrors.Wrap(err, "burn subtitles")
	}

	return nil
}

func checkPaths(opts Options) error {
	if opts.Input == "" || opts.Subtitles == "" || opts.Output == "" {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "input, subtitles and output are required")
	}
	if filepath.Ext(opts.Output) == "" {
		return ctxerrors.Wrapf(commerr.ErrInvalidArgument, "output %q needs a file extension that names its container", opts.Output)
	}
	input, err := os.Stat(opts.Input)
	if err != nil {
		return ctxerrors.Wrap(err, "open input video")
	}
	if _, err := os.Stat(opts.Subtitles); err != nil {
		return ctxerrors.Wrap(err, "open subtitles")
	}
	output, err := os.Stat(opts.Output)
	if err == nil && os.SameFile(input, output) {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "output must not replace the input video")
	}

	return nil
}

func executable(path, name string) (string, error) {
	if path == "" {
		path = name
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return "", ctxerrors.Wrapf(errors.Join(commerr.ErrNotFound, err), "%s executable %q; install FFmpeg", name, path)
	}

	return resolved, nil
}

func probe(ctx context.Context, ffprobe, input string) ([]stream, error) {
	command := exec.CommandContext(ctx, ffprobe, "-v", "error",
		"-show_entries", "stream=index,codec_type,codec_name,codec_tag_string,pix_fmt,"+
			"color_range,color_space,color_transfer,color_primaries:stream_disposition=attached_pic",
		"-of", "json", "-i", fileProtocol+input)
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	data, err := command.Output()
	if err != nil {
		return nil, ctxerrors.Wrapf(err, "run ffprobe: %s", strings.TrimSpace(diagnostics.String()))
	}
	var result struct {
		Streams []stream `json:"streams"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, ctxerrors.Wrap(errors.Join(commerr.ErrInvalidValue, err), "decode ffprobe output")
	}

	return result.Streams, nil
}

// mainVideo returns the first video stream that is not cover art.
func mainVideo(streams []stream) (video, error) {
	ordinal := 0
	for _, candidate := range streams {
		if candidate.CodecType != codecTypeVideo {
			continue
		}
		if candidate.Disposition.AttachedPic == 0 {
			return video{stream: candidate, ordinal: ordinal}, nil
		}
		ordinal++
	}

	return video{}, ctxerrors.Wrap(commerr.ErrNotFound, "input has no video stream to draw on")
}

// burnArgs copies every stream and re-encodes only the main video stream.
// FFmpeg turns a rotated video upright before the filter, so the subtitles
// are drawn in the orientation the video is shown in.
func burnArgs(input, subtitles string, streams []stream, main video, codec encoder, crf int) []string {
	spec := ":v:" + strconv.Itoa(main.ordinal)
	args := []string{
		"-nostdin", "-hide_banner", "-loglevel", "error", "-stats", "-y",
		"-i", fileProtocol + input, "-map", "0",
	}
	// An MP4 or QuickTime file stores its chapters a second time as a text
	// data track. The muxer writes that track again from the chapters, so
	// copying it as well would leave two chapter tracks.
	for _, candidate := range streams {
		if candidate.CodecType == codecTypeData && candidate.CodecName == codecBinData && candidate.CodecTag == tagText {
			args = append(args, "-map", "-0:"+strconv.Itoa(candidate.Index))
		}
	}
	args = append(args,
		"-map_metadata", "0", "-map_chapters", "0", "-c", "copy",
		"-filter"+spec, "ass=filename="+filterValue(subtitles),
		"-c"+spec, codec.name, "-crf"+spec, strconv.Itoa(crf),
	)
	for index := 0; index+1 < len(codec.quality); index += 2 {
		args = append(args, codec.quality[index]+spec, codec.quality[index+1])
	}
	args = append(args, "-fps_mode"+spec, "passthrough")
	for _, option := range []struct{ name, value string }{
		{"-pix_fmt", main.PixelFormat},
		{"-color_range", main.ColorRange},
		{"-colorspace", main.ColorSpace},
		{"-color_trc", main.ColorTransfer},
		{"-color_primaries", main.ColorPrimaries},
	} {
		if option.value != "" && option.value != unknownValue {
			args = append(args, option.name+spec, option.value)
		}
	}
	if main.CodecTag == tagHVC1 {
		args = append(args, "-tag"+spec, tagHVC1)
	}

	return args
}

// filterValue escapes a path for a filter option and then for the filter
// graph, the two levels FFmpeg unescapes when it reads -filter.
func filterValue(path string) string {
	option := escape(path, `\':`)

	return escape(option, `\'[],;`)
}

func escape(text, special string) string {
	var escaped strings.Builder
	for _, character := range text {
		if strings.ContainsRune(special, character) {
			escaped.WriteByte('\\')
		}
		escaped.WriteRune(character)
	}

	return escaped.String()
}

func encode(ctx context.Context, ffmpeg string, opts Options, args []string) error {
	temp, err := os.CreateTemp(filepath.Dir(opts.Output), tempPattern+filepath.Ext(opts.Output))
	if err != nil {
		return ctxerrors.Wrap(err, "create temporary output")
	}
	name := temp.Name()
	if err := temp.Close(); err != nil {
		return ctxerrors.Wrap(errors.Join(err, os.Remove(name)), "close temporary output")
	}
	tail := &tailBuffer{limit: stderrTail}
	diagnostics := io.Writer(tail)
	if opts.Diagnostics != nil {
		diagnostics = io.MultiWriter(opts.Diagnostics, tail)
	}
	command := exec.CommandContext(ctx, ffmpeg, append(args, fileProtocol+name)...)
	command.Stderr = diagnostics
	if err := command.Run(); err != nil {
		return ctxerrors.Wrapf(errors.Join(err, os.Remove(name)), "run ffmpeg: %s", strings.TrimSpace(tail.String()))
	}
	if err := os.Rename(name, opts.Output); err != nil {
		return ctxerrors.Wrap(errors.Join(err, os.Remove(name)), "replace output")
	}

	return nil
}

// tailBuffer keeps the last limit bytes written to it.
type tailBuffer struct {
	limit int
	data  []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	if extra := len(b.data) - b.limit; extra > 0 {
		b.data = b.data[extra:]
	}

	return len(p), nil
}

func (b *tailBuffer) String() string { return string(b.data) }
