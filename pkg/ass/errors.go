package ass

import "errors"

// Errors reported by the reader and editor.
var (
	ErrInvalidDocument = errors.New("invalid ASS document")
	ErrInvalidValue    = errors.New("invalid ASS value")
	ErrNotFound        = errors.New("ASS entry not found")
	ErrInputTooLarge   = errors.New("ASS input exceeds 64 MiB")
)
