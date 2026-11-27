package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/psyb0t/ctxerrors"

	"github.com/psyb0t/assctl/pkg/ass"
)

const maxSpecBytes = 64 * 1024 * 1024

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
		return result, ctxerrors.Wrap(ass.ErrInputTooLarge, "read JSON input")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, ctxerrors.Wrap(err, "decode JSON input")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return result, ctxerrors.Wrap(errUsage, "JSON input must contain one value")
	}

	return result, nil
}

func (c cli) save(path, content string) error {
	if path == stream {
		return writeString(c.output, content)
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
	if _, err := io.WriteString(writer, value); err != nil {
		return ctxerrors.Wrap(err, "write command output")
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
