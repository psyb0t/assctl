package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/psyb0t/assctl/pkg/ass"
)

func TestCommandLifecycle(t *testing.T) {
	const source = "{\"Title\":\"Clip\",\"Width\":1080,\"Height\":1920,\"Dialogues\":[{\"Start\":0,\"End\":2000000000,\"Text\":\"hello, world\"}]}"
	var output, diagnostics bytes.Buffer
	file := filepath.Join(t.TempDir(), "captions.ass")
	runCommand := func(args ...string) string {
		t.Helper()
		output.Reset()
		if err := run(args, strings.NewReader(source), &output, &diagnostics); err != nil {
			t.Fatalf("%v: %v, stderr=%s", args, err, diagnostics.String())
		}
		return output.String()
	}

	runCommand("captions", "--preset", "black-gold", "--output", file)
	if result := runCommand("validate", file); result != "valid ASS\n" {
		t.Fatalf("validate output: %q", result)
	}
	if result := runCommand("inspect", file); !strings.Contains(result, "hello, world") {
		t.Fatalf("inspect output: %q", result)
	}
	runCommand("theme", "--preset", "mint-plum", "--style", "Text", "--output", file, file)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "DejaVu Sans Mono") || !strings.Contains(string(data), "hello, world") {
		t.Fatalf("themed file: %q", data)
	}
	if result := runCommand("presets", "--json"); !strings.Contains(result, "\"ivory-navy\"") {
		t.Fatalf("preset list: %q", result)
	}
	preset, err := ass.LookupPreset("clean")
	if err != nil {
		t.Fatal(err)
	}
	preset.Font = "Custom Font"
	custom, err := json.Marshal(preset)
	if err != nil {
		t.Fatal(err)
	}
	themeFile := filepath.Join(t.TempDir(), "theme.json")
	if err := os.WriteFile(themeFile, custom, 0o600); err != nil {
		t.Fatal(err)
	}
	if result := runCommand("captions", "--theme-file", themeFile); !strings.Contains(result, "Custom Font") {
		t.Fatalf("custom caption output: %q", result)
	}
}

func TestMalformedInputs(t *testing.T) {
	for _, command := range [][]string{
		{"create"},
		{"edit", "--spec", "-", "-"},
		{"captions", "--preset", "missing"},
		{"captions", "--preset", "clean", "--theme-file", "-"},
		{"theme", "--preset", "mint-plum", "-"},
	} {
		var output bytes.Buffer
		err := run(command, strings.NewReader("{\"Unknown\": true}"), &output, &output)
		if err == nil {
			t.Fatalf("accepted malformed input for %v", command)
		}
	}
}
