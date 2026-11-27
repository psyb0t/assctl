package ass

import "errors"

// Errors specific to ASS documents. Missing entries wrap commerr.ErrNotFound and
// rejected values wrap commerr.ErrInvalidValue from github.com/psyb0t/ctxerrors/commerr.
var (
	ErrInvalidDocument = errors.New("invalid ASS document")
	ErrInputTooLarge   = errors.New("ASS input exceeds 64 MiB")
)
