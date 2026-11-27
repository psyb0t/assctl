package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

const scriptJSON = "{" +
	"\"Title\":\"Test clip\",\"Resolution\":{\"Width\":540,\"Height\":960}," +
	"\"Styles\":[{\"Name\":\"Default\",\"FontName\":\"DejaVu Sans\"}]," +
	"\"Events\":[{\"Style\":\"Default\",\"End\":1000000000,\"Text\":\"Café, world\"}]}"

type brokenOutput struct{ err error }

func (w brokenOutput) Write([]byte) (int, error) { return 0, w.err }

func runCLI(t *testing.T, input string, args ...string) string {
	t.Helper()

	var output, diagnostics bytes.Buffer
	err := run(args, strings.NewReader(input), &output, &diagnostics)
	require.NoError(t, err, "%v stderr: %s", args, diagnostics.String())

	return output.String()
}

func runCLIFailure(t *testing.T, input string, args ...string) error {
	t.Helper()

	var output bytes.Buffer
	err := run(args, strings.NewReader(input), &output, io.Discard)
	require.Error(t, err, "%v", args)
	require.Empty(t, output.String(), "%v wrote output before failing", args)

	return err
}

func TestCLICreateEditAndInspect(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "source.ass")
	runCLI(t, scriptJSON, "create", "--output", path)
	original, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "valid ASS\n", runCLI(t, string(original), "validate", "-"))

	edits := "{\"Info\":{\"Title\":\"Edited\"}," +
		"\"Events\":[{\"Index\":1,\"Override\":{\"Position\":{\"X\":270,\"Y\":800}}}]}"
	updated := runCLI(t, edits, "edit", "--spec", "-", path)
	infoJSON := runCLI(t, updated, "inspect", "-")
	var contents ass.Contents
	require.NoError(t, json.Unmarshal([]byte(infoJSON), &contents))
	assert.Equal(t, "Edited", contents.Info["Title"])
	require.Len(t, contents.Events, 1)
	assert.Equal(t, "{\\pos(270,800)}Café, world", contents.Events[0]["Text"])

	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, saved, "edit changed the input file")
}

func TestCLIHelpPresetsAndVersion(t *testing.T) {
	t.Parallel()

	t.Run("help forms", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			args []string
		}{
			{name: "no arguments"},
			{name: "help command", args: []string{"help"}},
			{name: "short flag", args: []string{"-h"}},
			{name: "long flag", args: []string{"--help"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := runCLI(t, "", tc.args...)
				assert.Contains(t, got, "Usage:")
				assert.Contains(t, got, "Available Commands:")
			})
		}
	})

	t.Run("preset list", func(t *testing.T) {
		t.Parallel()

		want := []string{
			"black-gold", "ivory-navy", "mint-plum", "clean", "tiktok-bold", "reels-bold",
			"shorts-impact", "social-center", "social-box", "social-hook", "the-bold",
		}
		presets := strings.Fields(runCLI(t, "", "presets"))
		assert.True(t, slices.Equal(presets, want), "presets returned %v, want %v", presets, want)
	})

	t.Run("version forms", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			args []string
		}{
			{name: "command", args: []string{"version"}},
			{name: "flag", args: []string{"--version"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, "dev\n", runCLI(t, "", tc.args...))
			})
		}
	})

	t.Run("command help", func(t *testing.T) {
		t.Parallel()

		var output, diagnostics bytes.Buffer
		err := run([]string{"captions", "--help"}, strings.NewReader(""), &output, &diagnostics)
		require.NoError(t, err)
		assert.Empty(t, diagnostics.String())
		assert.Contains(t, output.String(), "--theme-file")
	})
}

func TestCLIFlagsAfterInputAndCompletion(t *testing.T) {
	t.Parallel()

	t.Run("theme flags after input", func(t *testing.T) {
		t.Parallel()

		source := filepath.Join(t.TempDir(), "source.ass")
		runCLI(t, scriptJSON, "create", "--output", source)
		themed := runCLI(t, "", "theme", source, "--preset", "mint-plum", "--style", "Default")
		document, err := ass.Parse(strings.NewReader(themed))
		require.NoError(t, err)
		styles := document.Contents().Styles
		require.NotEmpty(t, styles)
		assert.Equal(t, "DejaVu Sans Mono", styles[0]["Fontname"])
	})

	t.Run("bash completion", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, runCLI(t, "", "completion", "bash"), "__start_assctl")
	})

	t.Run("unknown completion shell", func(t *testing.T) {
		t.Parallel()

		var output bytes.Buffer
		err := run([]string{"completion", "unknown-shell"}, strings.NewReader(""), &output, io.Discard)
		require.Error(t, err)
		assert.Empty(t, output.String())
	})
}

func TestCLIRejectsInvalidArgumentsAndFiles(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")
	testCases := []struct {
		name  string
		args  []string
		input string
		want  error
	}{
		{name: "unknown command", args: []string{"unknown"}},
		{
			name: "unknown font preset", args: []string{"fonts-install", "--preset", "missing"},
			want: commerr.ErrNotFound,
		},
		{
			name: "extra font argument", args: []string{"fonts-install", "extra"},
			want: commerr.ErrInvalidArgument,
		},
		{
			name: "extra version argument", args: []string{"version", "extra"},
			want: commerr.ErrInvalidArgument,
		},
		{name: "unrecognized flag", args: []string{"presets", "--bad"}},
		{
			name: "two JSON values", args: []string{"create"}, input: scriptJSON + " {}",
			want: commerr.ErrInvalidArgument,
		},
		{name: "unknown JSON field", args: []string{"create"}, input: "{\"Unexpected\":true}"},
		{name: "malformed JSON", args: []string{"create"}, input: "{"},
		{
			name: "invalid script JSON", args: []string{"create"}, input: "{}",
			want: commerr.ErrInvalidValue,
		},
		{name: "missing ASS file", args: []string{"inspect", missing}, want: fs.ErrNotExist},
		{name: "missing JSON file", args: []string{"create", "--spec", missing}, want: fs.ErrNotExist},
		{name: "missing edits file", args: []string{"edit", "--spec", missing, "-"}, want: fs.ErrNotExist},
		{
			name: "invalid edit source", args: []string{"edit", "--spec", "-", missing}, input: "{}",
			want: fs.ErrNotExist,
		},
		{
			name: "missing theme file", args: []string{"theme", "--theme-file", missing, "-"},
			want: fs.ErrNotExist,
		},
		{
			name: "invalid theme source", args: []string{"theme", "-"}, input: "bad source",
			want: ass.ErrInvalidDocument,
		},
		{
			name: "invalid validation source", args: []string{"validate", "-"},
			input: "[Script Info]\n[Events]\nFormat: Style, Start, End, Text\nDialogue: Missing,0:00:00.00,0:00:01.00,Hello\n",
			want:  ass.ErrInvalidDocument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := runCLIFailure(t, tc.input, tc.args...)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestCLIPropagatesStreamFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("stream disconnected")

	t.Run("input", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			args []string
		}{
			{name: "create", args: []string{"create"}},
			{name: "inspect", args: []string{"inspect", "-"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				err := run(tc.args, iotest.ErrReader(failure), io.Discard, io.Discard)
				require.ErrorIs(t, err, failure)
			})
		}
	})

	t.Run("output", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			args []string
		}{
			{name: "presets", args: []string{"presets"}},
			{name: "presets JSON", args: []string{"presets", "--json"}},
			{name: "create", args: []string{"create"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				err := run(tc.args, strings.NewReader(scriptJSON), brokenOutput{failure}, io.Discard)
				require.ErrorIs(t, err, failure)
			})
		}
	})
}

func TestCLIFailedEditPreservesExistingOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "source.ass")
	runCLI(t, scriptJSON, "create", "--output", path)
	original, err := os.ReadFile(path)
	require.NoError(t, err)

	const spec = "{\"Info\":{\"Title\":\"Edited\"},\"Events\":[{\"Index\":2}]}"
	args := []string{"edit", "--spec", "-", "--output", path, path}
	err = run(args, strings.NewReader(spec), io.Discard, io.Discard)
	require.ErrorIs(t, err, commerr.ErrNotFound)

	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, saved, "failed edit changed output")
}

func TestCLICleansTemporaryFilesAfterFailedOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	targetDirectory := filepath.Join(directory, "target")
	require.NoError(t, os.Mkdir(targetDirectory, 0o700))
	missingParent := filepath.Join(directory, "missing", "file.ass")

	for _, path := range []string{targetDirectory, missingParent} {
		err := run([]string{"create", "--output", path}, strings.NewReader(scriptJSON), io.Discard, io.Discard)
		require.Error(t, err, "output to %q unexpectedly succeeded", path)
	}

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "target", entries[0].Name())
	assert.True(t, entries[0].IsDir())
}

func TestCLIPositionAndTagFailures(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	file := filepath.Join(directory, "source.ass")
	runCLI(t, scriptJSON, "create", "--output", file)

	testCases := []struct {
		name  string
		args  []string
		input string
	}{
		{name: "missing position source", args: []string{"position", "--text-position", "portrait-upper", filepath.Join(directory, "absent.ass")}},
		{name: "missing position style", args: []string{"position", "--text-position", "portrait-upper", "--style", "absent", file}},
		{name: "missing position option", args: []string{"position", file}},
		{name: "unknown theme position", args: []string{"theme", "--text-position", "absent", file}},
		{name: "unknown theme", args: []string{"theme", "--preset", "absent", file}},
		{name: "invalid canvas", args: []string{"create", "--text-position", "portrait-upper"}, input: strings.ReplaceAll(scriptJSON, `"Width":540`, `"Width":0`)},
		{name: "malformed tags", args: []string{"tags"}, input: `{`},
		{name: "unknown tag field", args: []string{"tags"}, input: `{"Spacingg":2}`},
		{name: "negative border", args: []string{"tags"}, input: `{"Border":-1}`},
		{name: "multiple JSON roots", args: []string{"tags"}, input: `{} {}`},
		{name: "trailing garbage", args: []string{"tags"}, input: `{} broken`},
		{name: "empty tag spec", args: []string{"tags"}},
		{name: "extra positional argument", args: []string{"positions", "extra"}},
		{name: "unknown font preset", args: []string{"fonts-install", "--preset", "absent"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_ = runCLIFailure(t, tc.input, tc.args...)
		})
	}
}

func TestCLIJSONRejectsAmbiguousObjects(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
	}{
		{name: "null", input: `null`},
		{name: "duplicate field", input: `{"Spacing":1,"Spacing":2}`},
		{name: "duplicate field differing in case", input: `{"Spacing":1,"spacing":2}`},
		{name: "duplicate nested field differing in case", input: `{"SimpleFade":{"In":0,"in":1000000}}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_ = runCLIFailure(t, tc.input, "tags")
		})
	}
}

const (
	safeAreaWidth  = 1080
	safeAreaHeight = 1920

	platformTikTok = "tiktok"
	platformShorts = "shorts"
	platformReels  = "reels"
)

var allPlatforms = []string{ass.SafeAreaAll, platformTikTok, platformShorts, platformReels}

type safeAreaMargins struct{ left, right, vertical int }

// scriptSpec drops ass.Script's text marshaling so it encodes as the JSON spec object.
type scriptSpec ass.Script

func runCLIOutput(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()

	var output bytes.Buffer
	err := run(args, strings.NewReader(input), &output, io.Discard)

	return output.String(), err
}

// writeSafeAreaScript writes a 1080 x 1920 script with one style and returns its path.
func writeSafeAreaScript(
	t *testing.T,
	directory, name string,
	alignment ass.Alignment,
	margins safeAreaMargins,
	events ...ass.Event,
) string {
	t.Helper()

	script := ass.Script{
		Title: "Safe area",
		Styles: []ass.Style{{
			Name: "Default", FontName: "DejaVu Sans", Alignment: alignment,
			MarginL: margins.left, MarginR: margins.right, MarginV: margins.vertical,
		}},
		Events: events,
	}
	script.Resolution.Width, script.Resolution.Height = safeAreaWidth, safeAreaHeight
	if len(script.Events) == 0 {
		script.Events = []ass.Event{{Style: "Default", End: time.Second, Text: "Hello"}}
	}
	spec, err := json.Marshal(scriptSpec(script))
	require.NoError(t, err)

	path := filepath.Join(directory, name+".ass")
	runCLI(t, string(spec), "create", "--output", path)

	return path
}

func decodeIssues(t *testing.T, output string) []ass.SafeAreaIssue {
	t.Helper()

	var issues []ass.SafeAreaIssue
	require.NoError(t, json.Unmarshal([]byte(output), &issues), output)

	return issues
}

func TestCLISafeAreaPlatforms(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	testCases := []struct {
		name      string
		margins   safeAreaMargins
		wantClean []string
	}{
		{name: "clears every platform", margins: safeAreaMargins{120, 300, 700}, wantClean: allPlatforms},
		{name: "clears TikTok only", margins: safeAreaMargins{120, 300, 660}, wantClean: []string{platformTikTok}},
		{name: "clears Shorts only", margins: safeAreaMargins{48, 192, 700}, wantClean: []string{platformShorts}},
		{name: "clears Reels only", margins: safeAreaMargins{65, 65, 700}, wantClean: []string{platformReels}},
		{name: "no margins"},
	}

	for _, tc := range testCases {
		path := writeSafeAreaScript(t, directory, tc.name, ass.AlignmentBottomCenter, tc.margins)
		for _, platform := range allPlatforms {
			t.Run(tc.name+"/"+platform, func(t *testing.T) {
				t.Parallel()

				output, err := runCLIOutput(t, "", "safe-area", "--platform", platform, path)
				if slices.Contains(tc.wantClean, platform) {
					require.NoError(t, err)
					assert.Equal(t, "[]\n", output)

					return
				}
				require.ErrorIs(t, err, commerr.ErrValidationFailed)
				issues := decodeIssues(t, output)
				require.NotEmpty(t, issues)
				for _, issue := range issues {
					assert.Equal(t, "Default", issue.Style)
					assert.Zero(t, issue.Event)
					assert.Less(t, issue.Margin, issue.Required)
				}
			})
		}
	}
}

func TestCLISafeAreaDefaultsToAll(t *testing.T) {
	t.Parallel()

	path := writeSafeAreaScript(t, t.TempDir(), "defaults", ass.AlignmentBottomCenter, safeAreaMargins{})

	implicit, implicitErr := runCLIOutput(t, "", "safe-area", path)
	explicit, explicitErr := runCLIOutput(t, "", "safe-area", "--platform", ass.SafeAreaAll, path)
	require.ErrorIs(t, implicitErr, commerr.ErrValidationFailed)
	require.ErrorIs(t, explicitErr, commerr.ErrValidationFailed)
	assert.Equal(t, explicit, implicit)
}

func TestCLISafeAreaIssues(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	testCases := []struct {
		name      string
		alignment ass.Alignment
		platform  string
		want      []ass.SafeAreaIssue
	}{
		{
			name: "bottom all", alignment: ass.AlignmentBottomCenter, platform: ass.SafeAreaAll,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeBottom, Required: 672},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 120},
				{Style: "Default", Edge: ass.EdgeRight, Required: 300},
			},
		},
		{
			name: "bottom tiktok", alignment: ass.AlignmentBottomCenter, platform: platformTikTok,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeBottom, Required: 660},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 120},
				{Style: "Default", Edge: ass.EdgeRight, Required: 300},
			},
		},
		{
			name: "bottom shorts", alignment: ass.AlignmentBottomCenter, platform: platformShorts,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeBottom, Required: 672},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 48},
				{Style: "Default", Edge: ass.EdgeRight, Required: 192},
			},
		},
		{
			name: "bottom reels", alignment: ass.AlignmentBottomCenter, platform: platformReels,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeBottom, Required: 672},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 65},
				{Style: "Default", Edge: ass.EdgeRight, Required: 65},
			},
		},
		{
			name: "top all", alignment: ass.AlignmentTopCenter, platform: ass.SafeAreaAll,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeTop, Required: 288},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 120},
				{Style: "Default", Edge: ass.EdgeRight, Required: 192},
			},
		},
		{
			name: "top tiktok", alignment: ass.AlignmentTopCenter, platform: platformTikTok,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeTop, Required: 240},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 120},
				{Style: "Default", Edge: ass.EdgeRight, Required: 120},
			},
		},
		{
			name: "top shorts", alignment: ass.AlignmentTopCenter, platform: platformShorts,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeTop, Required: 288},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 48},
				{Style: "Default", Edge: ass.EdgeRight, Required: 192},
			},
		},
		{
			name: "top reels", alignment: ass.AlignmentTopCenter, platform: platformReels,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeTop, Required: 269},
				{Style: "Default", Edge: ass.EdgeLeft, Required: 65},
				{Style: "Default", Edge: ass.EdgeRight, Required: 65},
			},
		},
		{
			name: "middle all uses the lower right edge", alignment: ass.AlignmentMiddleCenter, platform: ass.SafeAreaAll,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeLeft, Required: 120},
				{Style: "Default", Edge: ass.EdgeRight, Required: 300},
			},
		},
		{
			name: "middle shorts keeps the upper right edge", alignment: ass.AlignmentMiddleCenter, platform: platformShorts,
			want: []ass.SafeAreaIssue{
				{Style: "Default", Edge: ass.EdgeLeft, Required: 48},
				{Style: "Default", Edge: ass.EdgeRight, Required: 192},
			},
		},
	}

	for _, tc := range testCases {
		path := writeSafeAreaScript(t, directory, tc.name, tc.alignment, safeAreaMargins{})
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			output, err := runCLIOutput(t, "", "safe-area", "--platform", tc.platform, path)
			require.ErrorIs(t, err, commerr.ErrValidationFailed)
			assert.Equal(t, tc.want, decodeIssues(t, output))
		})
	}
}

func TestCLISafeAreaJSONShape(t *testing.T) {
	t.Parallel()

	path := writeSafeAreaScript(t, t.TempDir(), "shape", ass.AlignmentBottomCenter, safeAreaMargins{120, 300, 100})

	output, err := runCLIOutput(t, "", "safe-area", path)
	require.ErrorIs(t, err, commerr.ErrValidationFailed)
	assert.Contains(t, err.Error(), "1 placement(s)")

	var raw []map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &raw))
	require.Len(t, raw, 1)
	assert.Equal(t, map[string]any{
		"Style": "Default", "Event": float64(0), "Edge": "bottom", "Margin": float64(100), "Required": float64(672),
	}, raw[0])
}

func TestCLISafeAreaEventMargins(t *testing.T) {
	t.Parallel()

	path := writeSafeAreaScript(
		t, t.TempDir(), "events", ass.AlignmentBottomCenter, safeAreaMargins{120, 300, 700},
		ass.Event{Style: "Default", End: time.Second, Text: "inherits the style"},
		ass.Event{Style: "Default", End: time.Second, MarginL: 10, Text: "narrow left"},
		ass.Event{Style: "Default", End: time.Second, MarginV: 100, Text: "low"},
		ass.Event{Style: "Default", End: time.Second, MarginL: 200, MarginR: 400, MarginV: 800, Text: "safe"},
	)

	output, err := runCLIOutput(t, "", "safe-area", path)
	require.ErrorIs(t, err, commerr.ErrValidationFailed)
	assert.Equal(t, []ass.SafeAreaIssue{
		{Style: "Default", Event: 2, Edge: ass.EdgeLeft, Margin: 10, Required: 120},
		{Style: "Default", Event: 3, Edge: ass.EdgeBottom, Margin: 100, Required: 672},
	}, decodeIssues(t, output))
}

func TestCLISafeAreaReadsStdin(t *testing.T) {
	t.Parallel()

	path := writeSafeAreaScript(t, t.TempDir(), "stdin", ass.AlignmentBottomCenter, safeAreaMargins{120, 300, 700})
	text, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "[]\n", runCLI(t, string(text), "safe-area", "-"))
}

func TestCLITextPositionsClearSafeArea(t *testing.T) {
	t.Parallel()

	for _, position := range ass.TextPositions() {
		t.Run(position.Name, func(t *testing.T) {
			t.Parallel()

			spec := `{"Width":1080,"Height":1920,"Dialogues":[{"End":1000000000,"Text":"Hello"}]}`
			text := runCLI(t, spec, "captions", "--text-position", position.Name)
			assert.Equal(t, "[]\n", runCLI(t, text, "safe-area", "-"))
		})
	}
}

func TestCLISafeAreaFailures(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	valid := writeSafeAreaScript(t, directory, "valid", ass.AlignmentBottomCenter, safeAreaMargins{120, 300, 700})
	text, err := os.ReadFile(valid)
	require.NoError(t, err)
	badMargin := strings.Replace(string(text), ",120,300,700,", ",abc,300,700,", 1)
	require.NotEqual(t, string(text), badMargin, "fixture margin was not replaced")

	testCases := []struct {
		name  string
		args  []string
		input string
		want  error
	}{
		{name: "unknown platform", args: []string{"safe-area", "--platform", "missing", valid}, want: commerr.ErrNotFound},
		{name: "unknown platform before input", args: []string{"safe-area", "--platform", "missing", "-"}, want: commerr.ErrNotFound},
		{name: "platform names are case sensitive", args: []string{"safe-area", "--platform", "TikTok", valid}, want: commerr.ErrNotFound},
		{name: "empty platform", args: []string{"safe-area", "--platform", "", valid}, want: commerr.ErrNotFound},
		{name: "missing input", args: []string{"safe-area", filepath.Join(directory, "absent.ass")}, want: fs.ErrNotExist},
		{name: "no input argument", args: []string{"safe-area"}, want: commerr.ErrInvalidArgument},
		{name: "extra input argument", args: []string{"safe-area", valid, valid}, want: commerr.ErrInvalidArgument},
		{name: "input is not ASS", args: []string{"safe-area", "-"}, input: "bad source", want: ass.ErrInvalidDocument},
		{name: "margin is not a number", args: []string{"safe-area", "-"}, input: badMargin, want: commerr.ErrInvalidValue},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := runCLIFailure(t, tc.input, tc.args...)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCLISafeAreaOutputFailure(t *testing.T) {
	t.Parallel()

	path := writeSafeAreaScript(t, t.TempDir(), "clean", ass.AlignmentBottomCenter, safeAreaMargins{120, 300, 700})
	broken := writeSafeAreaScript(t, t.TempDir(), "broken", ass.AlignmentBottomCenter, safeAreaMargins{})

	testCases := []struct {
		name string
		path string
	}{
		{name: "clean file", path: path},
		{name: "failing file", path: broken},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run([]string{"safe-area", tc.path}, strings.NewReader(""), errorOutput{}, io.Discard)
			require.ErrorIs(t, err, io.ErrClosedPipe)
		})
	}
}

func TestCLISafeAreas(t *testing.T) {
	t.Parallel()

	t.Run("names", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, allPlatforms, strings.Fields(runCLI(t, "", "safe-areas")))
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		var areas []ass.SafeArea
		require.NoError(t, json.Unmarshal([]byte(runCLI(t, "", "safe-areas", "--json")), &areas))
		assert.Equal(t, ass.SafeAreas(), areas)
		require.Len(t, areas, len(allPlatforms))
		for _, area := range areas {
			assert.Positive(t, area.TopRatio, area.Name)
			assert.Positive(t, area.BottomRatio, area.Name)
			assert.Positive(t, area.LeftRatio, area.Name)
			assert.Positive(t, area.RightRatio, area.Name)
		}
	})

	t.Run("extra argument", func(t *testing.T) {
		t.Parallel()

		err := runCLIFailure(t, "", "safe-areas", "extra")
		require.ErrorIs(t, err, commerr.ErrInvalidArgument)
	})
}
