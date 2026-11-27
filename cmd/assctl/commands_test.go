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

	"github.com/psyb0t/assctl/pkg/ass"
)

const scriptJSON = "{" +
	"\"Title\":\"Test clip\",\"Resolution\":{\"Width\":540,\"Height\":960}," +
	"\"Styles\":[{\"Name\":\"Default\",\"FontName\":\"DejaVu Sans\"}]," +
	"\"Events\":[{\"Style\":\"Default\",\"End\":1000000000,\"Text\":\"Café, world\"}]}"

func runCLI(t *testing.T, input string, args ...string) string {
	t.Helper()

	var output, diagnostics bytes.Buffer
	if err := run(args, strings.NewReader(input), &output, &diagnostics); err != nil {
		t.Fatalf("%v returned %v, stderr: %s", args, err, diagnostics.String())
	}
	return output.String()
}

func TestCLICreateEditAndInspect(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "source.ass")
	runCLI(t, scriptJSON, "create", "--output", path)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := runCLI(t, string(original), "validate", "-"); got != "valid ASS\n" {
		t.Fatalf("validate returned %q", got)
	}
	edits := "{\"Info\":{\"Title\":\"Edited\"}," +
		"\"Events\":[{\"Index\":1,\"Override\":{\"Position\":{\"X\":270,\"Y\":800}}}]}"
	updated := runCLI(t, edits, "edit", "--spec", "-", path)
	infoJSON := runCLI(t, updated, "inspect", "-")
	var contents ass.Contents
	if err := json.Unmarshal([]byte(infoJSON), &contents); err != nil {
		t.Fatal(err)
	}
	if contents.Info["Title"] != "Edited" || len(contents.Events) != 1 ||
		contents.Events[0]["Text"] != "{\\pos(270,800)}Café, world" {
		t.Fatalf("unexpected edited contents: %+v", contents)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatalf("edit changed the input file, read error: %v", err)
	}
}

func TestCLIHelpPresetsAndVersion(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"help"}, {"-h"}, {"--help"}} {
		if got := runCLI(t, "", args...); !strings.Contains(got, "Usage:") || !strings.Contains(got, "Available Commands:") {
			t.Fatalf("%v returned %q", args, got)
		}
	}
	presets := strings.Fields(runCLI(t, "", "presets"))
	if want := []string{"black-gold", "ivory-navy", "mint-plum", "clean"}; !slices.Equal(presets, want) {
		t.Fatalf("presets returned %v, want %v", presets, want)
	}
	for _, args := range [][]string{{"version"}, {"--version"}} {
		if got := runCLI(t, "", args...); got != "dev\n" {
			t.Fatalf("version returned %q for local build", got)
		}
	}
	var output, diagnostics bytes.Buffer
	if err := run([]string{"captions", "--help"}, strings.NewReader(""), &output, &diagnostics); err != nil ||
		diagnostics.Len() != 0 || !strings.Contains(output.String(), "--theme-file") {
		t.Fatalf("command help returned %v, stdout %q, stderr %q", err, output.String(), diagnostics.String())
	}
}

func TestCLIFlagsAfterInputAndCompletion(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "source.ass")
	runCLI(t, scriptJSON, "create", "--output", source)
	themed := runCLI(t, "", "theme", source, "--preset", "mint-plum", "--style", "Default")
	document, err := ass.Parse(strings.NewReader(themed))
	if err != nil {
		t.Fatal(err)
	}
	if got := document.Contents().Styles[0]["Fontname"]; got != "DejaVu Sans Mono" {
		t.Fatalf("theme flag after input selected font %q", got)
	}
	completion := runCLI(t, "", "completion", "bash")
	if !strings.Contains(completion, "__start_assctl") {
		t.Fatal("bash completion lacks the assctl entry point")
	}
	var output bytes.Buffer
	err = run([]string{"completion", "unknown-shell"}, strings.NewReader(""), &output, io.Discard)
	if err == nil || output.Len() != 0 {
		t.Fatalf("invalid completion shell returned %v, output %q", err, output.String())
	}
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
		{name: "extra version argument", args: []string{"version", "extra"}, want: errUsage},
		{name: "unrecognized flag", args: []string{"presets", "--bad"}},
		{name: "two JSON values", args: []string{"create"}, input: scriptJSON + " {}", want: errUsage},
		{name: "unknown JSON field", args: []string{"create"}, input: "{\"Unexpected\":true}"},
		{name: "malformed JSON", args: []string{"create"}, input: "{"},
		{name: "invalid script JSON", args: []string{"create"}, input: "{}", want: ass.ErrInvalidValue},
		{name: "missing ASS file", args: []string{"inspect", missing}, want: fs.ErrNotExist},
		{name: "missing JSON file", args: []string{"create", "--spec", missing}, want: fs.ErrNotExist},
		{name: "missing edits file", args: []string{"edit", "--spec", missing, "-"}, want: fs.ErrNotExist},
		{name: "invalid edit source", args: []string{"edit", "--spec", "-", missing}, input: "{}", want: fs.ErrNotExist},
		{name: "missing theme file", args: []string{"theme", "--theme-file", missing, "-"}, want: fs.ErrNotExist},
		{name: "invalid theme source", args: []string{"theme", "-"}, input: "bad source", want: ass.ErrInvalidDocument},
		{
			name: "invalid validation source", args: []string{"validate", "-"},
			input: "[Script Info]\n[Events]\nFormat: Style, Start, End, Text\nDialogue: Missing,0:00:00.00,0:00:01.00,Hello\n",
			want:  ass.ErrInvalidDocument,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			err := run(tc.args, strings.NewReader(tc.input), &output, io.Discard)
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) || output.Len() != 0 {
				t.Fatalf("%v returned %v, want %v, output %q", tc.args, err, tc.want, output.String())
			}
		})
	}
}

type brokenOutput struct{ err error }

func (w brokenOutput) Write([]byte) (int, error) { return 0, w.err }

func TestCLIPropagatesStreamFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("stream disconnected")
	for _, args := range [][]string{{"create"}, {"inspect", "-"}} {
		err := run(args, iotest.ErrReader(failure), io.Discard, io.Discard)
		if !errors.Is(err, failure) {
			t.Fatalf("%v returned %v", args, err)
		}
	}
	for _, args := range [][]string{{"presets"}, {"presets", "--json"}, {"create"}} {
		err := run(args, strings.NewReader(scriptJSON), brokenOutput{failure}, io.Discard)
		if !errors.Is(err, failure) {
			t.Fatalf("%v returned %v", args, err)
		}
	}
}

func TestCLIFailedEditPreservesExistingOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "source.ass")
	runCLI(t, scriptJSON, "create", "--output", path)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const spec = "{\"Info\":{\"Title\":\"Edited\"},\"Events\":[{\"Index\":2}]}"
	args := []string{"edit", "--spec", "-", "--output", path, path}
	err = run(args, strings.NewReader(spec), io.Discard, io.Discard)
	if !errors.Is(err, ass.ErrNotFound) {
		t.Fatalf("edit returned %v", err)
	}
	saved, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatalf("failed edit changed output, read error: %v", err)
	}
}

func TestCLICleansTemporaryFilesAfterFailedOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	targetDirectory := filepath.Join(directory, "target")
	if err := os.Mkdir(targetDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	missingParent := filepath.Join(directory, "missing", "file.ass")
	for _, path := range []string{targetDirectory, missingParent} {
		err := run([]string{"create", "--output", path}, strings.NewReader(scriptJSON), io.Discard, io.Discard)
		if err == nil {
			t.Fatalf("output to %q unexpectedly succeeded", path)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "target" || !entries[0].IsDir() {
		t.Fatalf("unexpected directory entries after failed output: %v", entries)
	}
}
