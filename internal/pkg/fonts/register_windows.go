//go:build windows

package fonts

import (
	"errors"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/psyb0t/ctxerrors"
)

const (
	userFontsKey   = `Software\Microsoft\Windows NT\CurrentVersion\Fonts`
	trueTypeSuffix = " (TrueType)"
)

// registerFont adds an installed font to the current user's font registry, so
// Windows applications find it without opening the file and choosing Install.
func registerFont(path string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, userFontsKey, registry.SET_VALUE)
	if err != nil {
		return ctxerrors.Wrap(err, "open user font registry")
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + trueTypeSuffix
	if err := errors.Join(key.SetStringValue(name, path), key.Close()); err != nil {
		return ctxerrors.Wrap(err, "write user font registry")
	}

	return nil
}
