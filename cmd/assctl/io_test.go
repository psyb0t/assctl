package main

import (
	"io"
	"strings"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/require"
)

type shortOutput struct{}

func (shortOutput) Write(data []byte) (int, error) {
	return len(data) / 2, nil
}

type errorOutput struct{}

func (errorOutput) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestCLIReportsShortWrite(t *testing.T) {
	t.Parallel()

	err := run([]string{"create"}, strings.NewReader(scriptJSON), shortOutput{}, io.Discard)
	require.ErrorIs(t, err, io.ErrShortWrite)
}

func TestCLIOutputFailureForEveryCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		args  []string
		input string
	}{
		{name: "positions", args: []string{"positions"}},
		{name: "positions JSON", args: []string{"positions", "--json"}},
		{name: "presets", args: []string{"presets"}},
		{name: "presets JSON", args: []string{"presets", "--json"}},
		{name: "tags", args: []string{"tags"}, input: `{"Spacing":2}`},
		{name: "version", args: []string{"version"}},
		{name: "help", args: []string{"--help"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run(tc.args, strings.NewReader(tc.input), errorOutput{}, io.Discard)
			require.ErrorIs(t, err, io.ErrClosedPipe)
		})
	}
}

func TestJSONInputSizeBoundary(t *testing.T) {
	t.Parallel()

	input := "{}" + strings.Repeat(" ", maxSpecBytes-1)

	t.Run("at the limit", func(t *testing.T) {
		t.Parallel()

		err := run([]string{"tags"}, strings.NewReader(input[:maxSpecBytes]), io.Discard, io.Discard)
		require.NoError(t, err)
	})

	t.Run("over the limit", func(t *testing.T) {
		t.Parallel()

		err := run([]string{"tags"}, strings.NewReader(input[:maxSpecBytes+1]), io.Discard, io.Discard)
		require.ErrorIs(t, err, errSpecTooLarge)
	})
}

func TestJSONNestingAndMalformedContainers(t *testing.T) {
	t.Parallel()

	// wantErr is nil for JSON syntax errors, which carry no sentinel.
	testCases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "array root", input: `[]`, wantErr: commerr.ErrInvalidArgument},
		{name: "truncated field value", input: `{"Karaoke":`},
		{name: "missing nested value", input: `{"SimpleFade":{"In":}}`},
		{name: "unterminated nested object", input: `{"SimpleFade":{`},
		{name: "malformed array", input: `{"Transform":[}`},
		{name: "unterminated array", input: `{"Transform":[{"End":1`},
		{name: "second root value", input: `{"Spacing":1} {}`, wantErr: commerr.ErrInvalidArgument},
		{name: "trailing garbage", input: `{"Spacing":1}x`, wantErr: commerr.ErrInvalidArgument},
		{
			name:    "excessive nesting",
			input:   `{"Spacing":` + strings.Repeat("[", maxJSONDepth+1) + `0` + strings.Repeat("]", maxJSONDepth+1) + `}`,
			wantErr: commerr.ErrInvalidArgument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run([]string{"tags"}, strings.NewReader(tc.input), io.Discard, io.Discard)
			require.Error(t, err)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}
