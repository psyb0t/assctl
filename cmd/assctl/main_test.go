package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

func TestCommandLifecycle(t *testing.T) {
	t.Parallel()

	const source = "{\"Title\":\"Clip\",\"Width\":1080,\"Height\":1920,\"Dialogues\":[{\"Start\":0,\"End\":2000000000,\"Text\":\"hello, world\"}]}"
	file := filepath.Join(t.TempDir(), "captions.ass")

	runCLI(t, source, "captions", "--preset", "black-gold", "--output", file)
	assert.Equal(t, "valid ASS\n", runCLI(t, source, "validate", file))
	assert.Contains(t, runCLI(t, source, "inspect", file), "hello, world")

	runCLI(t, source, "theme", "--preset", "mint-plum", "--style", "Text", "--output", file, file)
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(data), "DejaVu Sans Mono")
	assert.Contains(t, string(data), "hello, world")

	assert.Contains(t, runCLI(t, source, "presets", "--json"), "\"ivory-navy\"")

	preset, err := ass.LookupPreset("clean")
	require.NoError(t, err)
	preset.Font = "Custom Font"
	custom, err := json.Marshal(preset)
	require.NoError(t, err)
	themeFile := filepath.Join(t.TempDir(), "theme.json")
	require.NoError(t, os.WriteFile(themeFile, custom, 0o600))
	assert.Contains(t, runCLI(t, source, "captions", "--theme-file", themeFile), "Custom Font")
}

func TestMalformedInputs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want error
	}{
		{name: "unknown JSON field", args: []string{"create"}},
		{name: "edit reading stdin twice", args: []string{"edit", "--spec", "-", "-"}, want: commerr.ErrInvalidArgument},
		{name: "unknown caption preset", args: []string{"captions", "--preset", "missing"}, want: commerr.ErrNotFound},
		{
			name: "caption preset with stdin theme", args: []string{"captions", "--preset", "clean", "--theme-file", "-"},
			want: commerr.ErrInvalidArgument,
		},
		{name: "theme source is not ASS", args: []string{"theme", "--preset", "mint-plum", "-"}, want: ass.ErrInvalidDocument},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			err := run(tc.args, strings.NewReader("{\"Unknown\": true}"), &output, &output)
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
		})
	}
}

// TestCLIProcessHelper is the child entry point for TestCLIRealProcessExitStatus.
// It stays serial because it rewrites os.Args and exits the process.
func TestCLIProcessHelper(t *testing.T) {
	if os.Getenv("ASSCTL_TEST_PROCESS") != "1" {
		return
	}
	for index, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"assctl"}, os.Args[index+1:]...)
			main()
			os.Exit(0)
		}
	}
	require.Fail(t, "missing helper argument separator")
}

func TestCLIRealProcessExitStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		args         []string
		input        string
		wantError    bool
		wantValidASS bool
	}{
		{name: "version", args: []string{"version"}},
		{name: "creation", args: []string{"create"}, input: scriptJSON, wantValidASS: true},
		{name: "bad command", args: []string{"does-not-exist"}, wantError: true},
		{name: "bad JSON", args: []string{"create"}, input: `{`, wantError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			executable, err := os.Executable()
			require.NoError(t, err)
			args := append([]string{"-test.run=^TestCLIProcessHelper$", "--"}, tc.args...)
			command := exec.CommandContext(t.Context(), executable, args...)
			command.Env = append(os.Environ(), "ASSCTL_TEST_PROCESS=1")
			command.Stdin = strings.NewReader(tc.input)
			var output, diagnostics bytes.Buffer
			command.Stdout, command.Stderr = &output, &diagnostics
			err = command.Run()

			if tc.wantError {
				var status *exec.ExitError
				require.ErrorAs(t, err, &status)
				assert.Equal(t, 1, status.ExitCode())
				assert.NotEmpty(t, diagnostics.String())
				assert.Empty(t, output.String())

				return
			}
			require.NoError(t, err, "stderr: %s", diagnostics.String())
			require.NotEmpty(t, output.String())
			if !tc.wantValidASS {
				return
			}
			document, err := ass.Parse(&output)
			require.NoError(t, err)
			require.NoError(t, document.Validate())
		})
	}
}
