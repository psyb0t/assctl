package main

import (
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	captionsJSON = `{"Width":540,"Height":960,"Dialogues":[{"End":1000000000,"Text":"Keep these words"}]}`

	fontsHeader      = "[Fonts]"
	fontNameLine     = "fontname: "
	embedFontFlag    = "--embed-font"
	embedPresetFonts = "--embed-preset-fonts"
	unsupportedFont  = "No Bundled Source"
)

// writeFontFile writes a small file named name inside directory and returns its path.
func writeFontFile(t *testing.T, directory, name string) string {
	t.Helper()

	require.NoError(t, os.MkdirAll(directory, 0o700))
	path := filepath.Join(directory, name)
	require.NoError(t, os.WriteFile(path, []byte("font bytes of "+name), 0o600))

	return path
}

// embeddedNames returns the names in the [Fonts] section of an ASS text.
func embeddedNames(text string) []string {
	var names []string
	for _, line := range strings.Split(text, "\n") {
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), fontNameLine); ok {
			names = append(names, name)
		}
	}

	return names
}

func assertValidScript(t *testing.T, text string) {
	t.Helper()

	assert.Equal(t, "valid ASS\n", runCLI(t, text, "validate", "-"))
	var contents ass.Contents
	require.NoError(t, json.Unmarshal([]byte(runCLI(t, text, "inspect", "-")), &contents))
	assert.NotEmpty(t, contents.Styles)
}

type embedCommand struct {
	name  string
	input string
	args  func(t *testing.T, fonts ...string) []string
}

func embedCommands(t *testing.T) []embedCommand {
	t.Helper()

	source := filepath.Join(t.TempDir(), "source.ass")
	runCLI(t, scriptJSON, "create", "--output", source)

	withFonts := func(args []string, fonts []string) []string {
		for _, font := range fonts {
			args = append(args, embedFontFlag, font)
		}

		return args
	}

	return []embedCommand{
		{
			name: "create", input: scriptJSON,
			args: func(_ *testing.T, fonts ...string) []string { return withFonts([]string{"create"}, fonts) },
		},
		{
			name: "captions", input: captionsJSON,
			args: func(_ *testing.T, fonts ...string) []string { return withFonts([]string{"captions"}, fonts) },
		},
		{
			name: "theme",
			args: func(_ *testing.T, fonts ...string) []string {
				return withFonts([]string{"theme", "--preset", "clean", source}, fonts)
			},
		},
	}
}

func TestCLIEmbedFont(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	regular := writeFontFile(t, directory, "Regular.ttf")
	italic := writeFontFile(t, directory, "Italic.otf")

	for _, command := range embedCommands(t) {
		t.Run(command.name, func(t *testing.T) {
			t.Parallel()

			t.Run("one font", func(t *testing.T) {
				t.Parallel()

				text := runCLI(t, command.input, command.args(t, regular)...)
				assert.Contains(t, text, fontsHeader)
				assert.Equal(t, []string{"Regular.ttf"}, embeddedNames(text))
				assertValidScript(t, text)
			})

			t.Run("repeated flag keeps order", func(t *testing.T) {
				t.Parallel()

				text := runCLI(t, command.input, command.args(t, italic, regular)...)
				assert.Equal(t, []string{"Italic.otf", "Regular.ttf"}, embeddedNames(text))
				assertValidScript(t, text)
			})

			t.Run("without the flag", func(t *testing.T) {
				t.Parallel()

				text := runCLI(t, command.input, command.args(t)...)
				assert.NotContains(t, text, fontsHeader)
				assert.Empty(t, embeddedNames(text))
			})

			t.Run("written to a file", func(t *testing.T) {
				t.Parallel()

				output := filepath.Join(t.TempDir(), "output.ass")
				runCLI(t, command.input, append(command.args(t, regular), "--output", output)...)
				data, err := os.ReadFile(output)
				require.NoError(t, err)
				assert.Equal(t, []string{"Regular.ttf"}, embeddedNames(string(data)))
				assert.Equal(t, "valid ASS\n", runCLI(t, "", "validate", output))
			})
		})
	}
}

func TestCLIEmbedFontKeepsFontDataLines(t *testing.T) {
	t.Parallel()

	path := writeFontFile(t, t.TempDir(), "Data.ttf")

	text := runCLI(t, scriptJSON, "create", "--embed-font", path)
	lines := strings.Split(text, "\n")
	index := 0
	for index < len(lines) && lines[index] != fontNameLine+"Data.ttf" {
		index++
	}
	require.Less(t, index+1, len(lines), "fontname line is the last line")
	assert.NotEmpty(t, lines[index+1], "encoded font data must follow the fontname line")
	assert.False(t, strings.HasPrefix(lines[index+1], "["), "section header found where font data belongs")
}

func TestCLIEmbedFontOnThemedFileWithFonts(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	first := writeFontFile(t, directory, "First.ttf")
	second := writeFontFile(t, directory, "Second.ttf")
	source := filepath.Join(directory, "source.ass")
	runCLI(t, scriptJSON, "create", "--embed-font", first, "--output", source)

	t.Run("adds to the existing section", func(t *testing.T) {
		t.Parallel()

		text := runCLI(t, "", "theme", "--preset", "mint-plum", "--embed-font", second, source)
		assert.Equal(t, []string{"First.ttf", "Second.ttf"}, embeddedNames(text))
		assert.Equal(t, 1, strings.Count(text, fontsHeader))
		assertValidScript(t, text)
	})

	t.Run("rejects a font that is already embedded", func(t *testing.T) {
		t.Parallel()

		err := runCLIFailure(t, "", "theme", "--embed-font", first, source)
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
	})

	t.Run("rejects the same name in another case", func(t *testing.T) {
		t.Parallel()

		other := writeFontFile(t, filepath.Join(directory, "other"), "FIRST.TTF")
		err := runCLIFailure(t, "", "theme", "--embed-font", other, source)
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
	})
}

func TestCLIEmbedFontFailures(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	first := writeFontFile(t, filepath.Join(directory, "a"), "Same.ttf")
	second := writeFontFile(t, filepath.Join(directory, "b"), "Same.ttf")
	differentCase := writeFontFile(t, filepath.Join(directory, "c"), "SAME.TTF")
	missing := filepath.Join(directory, "absent.ttf")
	empty := filepath.Join(directory, "empty.ttf")
	require.NoError(t, os.WriteFile(empty, nil, 0o600))
	oversized := filepath.Join(directory, "huge.ttf")
	require.NoError(t, os.WriteFile(oversized, nil, 0o600))
	require.NoError(t, os.Truncate(oversized, maxSpecBytes+1))
	folder := filepath.Join(directory, "folder.ttf")
	require.NoError(t, os.Mkdir(folder, 0o700))

	testCases := []struct {
		name  string
		fonts []string
		want  error
	}{
		{name: "missing file", fonts: []string{missing}, want: fs.ErrNotExist},
		{name: "missing file after a good one", fonts: []string{first, missing}, want: fs.ErrNotExist},
		{name: "duplicate name", fonts: []string{first, second}, want: commerr.ErrInvalidValue},
		{name: "duplicate name in another case", fonts: []string{first, differentCase}, want: commerr.ErrInvalidValue},
		{name: "same file twice", fonts: []string{first, first}, want: commerr.ErrInvalidValue},
		{name: "empty file", fonts: []string{empty}, want: commerr.ErrInvalidValue},
		{name: "file over the size limit", fonts: []string{oversized}, want: errFontTooLarge},
		{name: "directory instead of a file", fonts: []string{folder}},
		{name: "empty path", fonts: []string{""}},
	}

	for _, command := range embedCommands(t) {
		for _, tc := range testCases {
			t.Run(command.name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				output := filepath.Join(t.TempDir(), "output.ass")
				require.NoError(t, os.WriteFile(output, []byte("keep"), 0o600))
				args := append(command.args(t, tc.fonts...), "--output", output)

				err := runCLIFailure(t, command.input, args...)
				if tc.want != nil {
					require.ErrorIs(t, err, tc.want)
				}
				data, readErr := os.ReadFile(output)
				require.NoError(t, readErr)
				assert.Equal(t, "keep", string(data), "failed command changed the output file")
			})
		}
	}
}

func TestCLIEmbedFontFlagsOnCommands(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		command         string
		wantFont        bool
		wantPresetFonts bool
	}{
		{command: "create", wantFont: true},
		{command: "captions", wantFont: true, wantPresetFonts: true},
		{command: "theme", wantFont: true, wantPresetFonts: true},
		{command: "edit"},
		{command: "position"},
		{command: "inspect"},
		{command: "validate"},
		{command: "safe-area"},
		{command: "fonts-install"},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()

			help := runCLI(t, "", tc.command, "--help")
			assert.Equal(t, tc.wantFont, strings.Contains(help, embedFontFlag+" "), "%s %s", tc.command, embedFontFlag)
			assert.Equal(t, tc.wantPresetFonts, strings.Contains(help, embedPresetFonts), "%s %s", tc.command, embedPresetFonts)
		})
	}

	t.Run("create rejects the preset fonts flag", func(t *testing.T) {
		t.Parallel()

		_ = runCLIFailure(t, scriptJSON, "create", embedPresetFonts)
	})
}

// The preset font tests change HOME and the proxy settings, so they run serially.
// HOME and XDG_DATA_HOME point at an empty directory and the proxy at a closed
// port so that no test installs fonts or reaches the network.
func TestCLIEmbedPresetFontsFailures(t *testing.T) {
	data := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", data)
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		t.Setenv(name, "http://127.0.0.1:1")
	}
	t.Setenv("NO_PROXY", "")

	custom := writeThemeFile(t, unsupportedFont)
	source := filepath.Join(t.TempDir(), "source.ass")
	runCLI(t, scriptJSON, "create", "--output", source)

	testCases := []struct {
		name  string
		args  []string
		input string
		want  func(t *testing.T, err error)
	}{
		{
			name: "captions with a font that has no bundled source",
			args: []string{"captions", "--theme-file", custom, embedPresetFonts}, input: captionsJSON,
			want: func(t *testing.T, err error) { require.ErrorIs(t, err, commerr.ErrUnsupported) },
		},
		{
			name: "theme with a font that has no bundled source",
			args: []string{"theme", "--theme-file", custom, embedPresetFonts, source},
			want: func(t *testing.T, err error) { require.ErrorIs(t, err, commerr.ErrUnsupported) },
		},
		{
			name: "captions cannot download", args: []string{"captions", embedPresetFonts}, input: captionsJSON,
			want: func(t *testing.T, err error) {
				var failure *url.Error
				require.ErrorAs(t, err, &failure)
			},
		},
		{
			name: "theme cannot download", args: []string{"theme", "--preset", "clean", embedPresetFonts, source},
			want: func(t *testing.T, err error) {
				var failure *url.Error
				require.ErrorAs(t, err, &failure)
			},
		},
		{
			name:  "download failure comes before a missing font file",
			args:  []string{"captions", embedPresetFonts, embedFontFlag, filepath.Join(data, "absent.ttf")},
			input: captionsJSON,
			want: func(t *testing.T, err error) {
				var failure *url.Error
				require.ErrorAs(t, err, &failure)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "output.ass")
			require.NoError(t, os.WriteFile(output, []byte("keep"), 0o600))
			args := append(append([]string{}, tc.args...), "--output", output)

			tc.want(t, runCLIFailure(t, tc.input, args...))
			kept, err := os.ReadFile(output)
			require.NoError(t, err)
			assert.Equal(t, "keep", string(kept), "failed command changed the output file")
			assert.NoDirExists(t, filepath.Join(data, "fonts"), "failed command installed fonts")
		})
	}
}

// writeThemeFile writes a custom theme that uses the given font family.
func writeThemeFile(t *testing.T, family string) string {
	t.Helper()

	preset, err := ass.LookupPreset("clean")
	require.NoError(t, err)
	preset.Font = family
	data, err := json.Marshal(preset)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "theme.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	return path
}
