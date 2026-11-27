package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const burnRenderTimeout = 2 * time.Minute

func TestCLIBurnRejectsMissingFiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
	}{
		{name: "no subtitles", args: []string{"burn", "in.mp4", "--output", "out.mp4"}},
		{name: "no output", args: []string{"burn", "in.mp4", "--subtitles", "a.ass"}},
		{name: "stdout output", args: []string{"burn", "in.mp4", "--subtitles", "a.ass", "--output", "-"}},
		{name: "stdin subtitles", args: []string{"burn", "in.mp4", "--subtitles", "-", "--output", "out.mp4"}},
		{name: "stdin video", args: []string{"burn", "-", "--subtitles", "a.ass", "--output", "out.mp4"}},
		{name: "no input", args: []string{"burn", "--subtitles", "a.ass", "--output", "out.mp4"}},
		{name: "two inputs", args: []string{"burn", "a.mp4", "b.mp4", "--subtitles", "a.ass", "--output", "out.mp4"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, runCLIFailure(t, "", tc.args...), commerr.ErrInvalidArgument)
		})
	}
}

func TestRenderBurnCommand(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	input := filepath.Join(directory, "clip.mp4")
	ctx, cancel := context.WithTimeout(t.Context(), burnRenderTimeout)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=320x568:r=24:d=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", input)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	spec := filepath.Join(directory, "spec.json")
	require.NoError(t, os.WriteFile(spec,
		[]byte(`{"Width":320,"Height":568,"Dialogues":[{"Start":0,"End":1000000000,"Text":"BURN"}]}`), 0o600))
	captions := filepath.Join(directory, "captions.ass")
	runCLI(t, "", "captions", "--spec", spec, "--output", captions)
	burned := filepath.Join(directory, "burned.mp4")

	t.Run("writes the video", func(t *testing.T) {
		t.Parallel()

		runCLI(t, "", "burn", input, "--subtitles", captions, "--output", burned, "--crf", "23")
		info, err := os.Stat(burned)
		require.NoError(t, err)
		assert.Positive(t, info.Size())
	})

	t.Run("rejects a CRF outside the encoder range", func(t *testing.T) {
		t.Parallel()

		err := runCLIFailure(t, "", "burn", input, "--subtitles", captions,
			"--output", filepath.Join(directory, "never.mp4"), "--crf", "99")
		require.ErrorIs(t, err, commerr.ErrInvalidArgument)
		assert.NoFileExists(t, filepath.Join(directory, "never.mp4"))
	})
}
