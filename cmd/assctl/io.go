package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	maxSpecBytes = 64 * 1024 * 1024
	maxJSONDepth = 64
)

type commandOutput struct {
	writer io.Writer
	err    error
}

func (w *commandOutput) Write(data []byte) (int, error) {
	written, err := w.writer.Write(data)
	if err == nil && written != len(data) {
		err = io.ErrShortWrite
	}
	if w.err == nil {
		w.err = err
	}
	if err != nil {
		return written, ctxerrors.Wrap(err, "write command stream")
	}

	return written, nil
}

func openInput(path string, input io.Reader) (io.ReadCloser, error) {
	if path == stream {
		return io.NopCloser(input), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "open input")
	}

	return file, nil
}

func (c cli) read(path string) (*ass.Document, error) {
	reader, err := openInput(path, c.input)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "open ASS input")
	}
	document, readErr := ass.Parse(reader)
	if err := errors.Join(readErr, reader.Close()); err != nil {
		return nil, ctxerrors.Wrap(err, "read ASS input")
	}

	return document, nil
}

func decodeFile[T any](path string, input io.Reader) (T, error) {
	var result T
	reader, err := openInput(path, input)
	if err != nil {
		return result, ctxerrors.Wrap(err, "open JSON input")
	}
	data, readErr := io.ReadAll(io.LimitReader(reader, maxSpecBytes+1))
	if err := errors.Join(readErr, reader.Close()); err != nil {
		return result, ctxerrors.Wrap(err, "read JSON input")
	}
	if len(data) > maxSpecBytes {
		return result, ctxerrors.Wrap(errSpecTooLarge, "read JSON input")
	}
	if err := validateJSONFields(data); err != nil {
		return result, ctxerrors.Wrap(err, "validate JSON fields")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, ctxerrors.Wrap(err, "decode JSON input")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return result, ctxerrors.Wrap(commerr.ErrInvalidArgument, "JSON input must contain one value")
	}

	return result, nil
}

func (c cli) save(path, content string) error {
	if path == stream {
		if err := writeString(c.output, content); err != nil {
			return ctxerrors.Wrap(err, "write ASS to output")
		}

		return nil
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".assctl-*")
	if err != nil {
		return ctxerrors.Wrap(err, "create temporary ASS output")
	}
	if err := errors.Join(writeString(temp, content), temp.Close()); err != nil {
		return ctxerrors.Wrap(errors.Join(err, os.Remove(temp.Name())), "write ASS output")
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return ctxerrors.Wrap(errors.Join(err, os.Remove(temp.Name())), "replace ASS output")
	}

	return nil
}

func writeString(writer io.Writer, value string) error {
	written, err := io.WriteString(writer, value)
	if err != nil {
		return ctxerrors.Wrap(err, "write command output")
	}
	if written != len(value) {
		return ctxerrors.Wrap(io.ErrShortWrite, "write complete command output")
	}

	return nil
}

func validateJSONFields(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "JSON spec must be an object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeJSONValue(decoder, 0); err != nil {
		return ctxerrors.Wrap(err, "read JSON object")
	}

	return nil
}

func consumeJSONValue(decoder *json.Decoder, depth int) error {
	if depth > maxJSONDepth {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "JSON nesting exceeds the supported depth")
	}
	token, err := decoder.Token()
	if err != nil {
		return ctxerrors.Wrap(err, "read JSON value")
	}
	delimiter, isContainer := token.(json.Delim)
	if !isContainer {
		return nil
	}
	seen := make(map[string]bool)
	for decoder.More() {
		if delimiter == '{' {
			keyToken, err := decoder.Token()
			if err != nil {
				return ctxerrors.Wrap(err, "read JSON field")
			}
			key, ok := keyToken.(string)
			if !ok {
				return ctxerrors.Wrap(commerr.ErrInvalidArgument, "JSON field name must be a string")
			}
			key = strings.ToLower(key)
			if seen[key] {
				return ctxerrors.Wrapf(commerr.ErrInvalidArgument, "duplicate JSON field %q", key)
			}
			seen[key] = true
		}
		if err := consumeJSONValue(decoder, depth+1); err != nil {
			return ctxerrors.Wrap(err, "read nested JSON value")
		}
	}
	if _, err := decoder.Token(); err != nil {
		return ctxerrors.Wrap(err, "close JSON container")
	}

	return nil
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return ctxerrors.Wrap(err, "write JSON output")
	}

	return nil
}
