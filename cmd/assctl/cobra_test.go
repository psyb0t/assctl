package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

func TestCLIPositionsAndTags(t *testing.T) {
	t.Parallel()

	t.Run("positions JSON", func(t *testing.T) {
		t.Parallel()

		var positions []ass.TextPosition
		require.NoError(t, json.Unmarshal([]byte(runCLI(t, "", "positions", "--json")), &positions))
		assert.Len(t, positions, 3)
	})

	t.Run("position names", func(t *testing.T) {
		t.Parallel()

		assert.Len(t, strings.Fields(runCLI(t, "", "positions")), 3)
	})

	t.Run("tags", func(t *testing.T) {
		t.Parallel()

		tags := runCLI(t, `{"Karaoke":{"Mode":"kf","Duration":350000000},"Spacing":2}`, "tags")
		assert.Equal(t, "{\\fsp2\\kf35}\n", tags)
	})

	source := filepath.Join(t.TempDir(), "source.ass")
	runCLI(t, scriptJSON, "create", "--output", source)

	t.Run("applied position", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			args []string
		}{
			{name: "position command", args: []string{"position", "--text-position", "portrait-upper", source}},
			{name: "theme command", args: []string{"theme", "--preset", "clean", "--text-position", "portrait-upper", source}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				text := runCLI(t, "", tc.args...)
				var contents ass.Contents
				require.NoError(t, json.Unmarshal([]byte(runCLI(t, text, "inspect", "-")), &contents))
				require.NotEmpty(t, contents.Styles)
				assert.Equal(t, "8", contents.Styles[0]["Alignment"])
				assert.Equal(t, "172", contents.Styles[0]["MarginV"])
			})
		}
	})

	t.Run("rejected position keeps output", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			args []string
			want error
		}{
			{
				name: "unknown position", args: []string{"position", "--text-position", "missing", source},
				want: commerr.ErrNotFound,
			},
			{
				name: "empty theme position", args: []string{"theme", "--text-position=", source},
				want: commerr.ErrInvalidArgument,
			},
			{
				name: "empty create position", args: []string{"create", "--text-position=", "--spec", "-"},
				want: commerr.ErrInvalidArgument,
			},
			{
				name: "unknown create position", args: []string{"create", "--text-position", "missing", "--spec", "-"},
				want: commerr.ErrNotFound,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				output := filepath.Join(t.TempDir(), "output.ass")
				require.NoError(t, os.WriteFile(output, []byte("keep"), 0o600))
				args := slices.Concat(tc.args, []string{"--output", output})

				err := runCLIFailure(t, scriptJSON, args...)
				require.ErrorIs(t, err, tc.want)
				data, readErr := os.ReadFile(output)
				require.NoError(t, readErr)
				assert.Equal(t, "keep", string(data))
			})
		}
	})
}

func TestCLICreationWithIndependentPosition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
	}{
		{name: "create", input: scriptJSON},
		{
			name:  "captions",
			input: `{"Width":540,"Height":960,"Dialogues":[{"End":1000000000,"Text":"Keep these words"}]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, position := range ass.TextPositions() {
				output := runCLI(t, tc.input, tc.name, "--text-position", position.Name)
				document, err := ass.Parse(strings.NewReader(output))
				require.NoError(t, err)
				require.NoError(t, document.Validate())

				for _, style := range document.Contents().Styles {
					assert.Equal(t, "97", style["MarginL"], "%s/%s left margin", tc.name, position.Name)
					assert.Equal(t, "97", style["MarginR"], "%s/%s right margin", tc.name, position.Name)
					if position.Name == "portrait-center" {
						assert.Equal(t, "0", style["MarginV"], "center retained a vertical margin")
					}
				}
			}
		})
	}
}
