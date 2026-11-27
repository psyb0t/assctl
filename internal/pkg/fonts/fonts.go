// Package fonts installs the font files required by assctl's built-in presets.
package fonts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gonfiguration"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	maxDownload      = 16 << 20
	maxFont          = 2 << 20
	downloadTimeout  = 60 * time.Second
	fontMode         = 0o644
	directoryMode    = 0o755
	fontSubdirectory = "assctl"
	tempFontPattern  = ".assctl-font-*"

	platformLinux   = "linux"
	platformFreeBSD = "freebsd"
	platformDarwin  = "darwin"
	platformWindows = "windows"
)

// environment holds the platform directory settings that override the defaults.
type environment struct {
	DataHome     string `env:"XDG_DATA_HOME"`
	LocalAppData string `env:"LOCALAPPDATA"`
}

// Install verifies and installs the fonts for the selected presets in the
// user's font directory and returns that directory. Files that are already
// installed with the expected checksum are not downloaded again. On Windows
// each font is also registered for the current user.
func Install(ctx context.Context, presets []ass.Preset) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ctxerrors.Wrap(err, "locate home directory")
	}
	var env environment
	if err := gonfiguration.Parse(&env); err != nil {
		return "", ctxerrors.Wrap(err, "read font directory settings")
	}
	directory, err := fontDirectory(runtime.GOOS, home, env)
	if err != nil {
		return "", ctxerrors.Wrap(err, "locate user fonts")
	}
	client := &http.Client{Timeout: downloadTimeout}
	if err := install(ctx, directory, presets, client, builtInFonts); err != nil {
		return "", ctxerrors.Wrap(err, "install preset fonts")
	}

	return directory, nil
}

func fontDirectory(platform, home string, env environment) (string, error) {
	switch platform {
	case platformLinux, platformFreeBSD:
		data := env.DataHome
		if data == "" {
			data = filepath.Join(home, ".local", "share")
		}
		if !filepath.IsAbs(data) {
			return "", ctxerrors.Wrap(commerr.ErrInvalidValue, "XDG_DATA_HOME must be absolute")
		}

		return filepath.Join(data, "fonts", fontSubdirectory), nil
	case platformDarwin:
		return filepath.Join(home, "Library", "Fonts", fontSubdirectory), nil
	case platformWindows:
		local := env.LocalAppData
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}

		return filepath.Join(local, "Microsoft", "Windows", "Fonts"), nil
	default:
		return "", ctxerrors.Wrapf(commerr.ErrUnsupported, "font directory on %s", platform)
	}
}

// install downloads every missing or changed file before writing any of them,
// so a failed download leaves the installed fonts untouched.
func install(
	ctx context.Context,
	directory string,
	presets []ass.Preset,
	client *http.Client,
	fonts catalog,
) error {
	assets := make(map[string]fontAsset)
	var fontFiles []string
	for _, preset := range presets {
		source, err := fonts.source(preset)
		if err != nil {
			return ctxerrors.Wrapf(err, "select font for preset %q", preset.Name)
		}
		if _, seen := assets[source.font.name]; !seen {
			fontFiles = append(fontFiles, source.font.name)
		}
		assets[source.font.name] = source.font
		assets[source.license.name] = source.license
	}

	contents := make(map[string][]byte)
	downloads := make(map[string][]byte)
	for _, name := range slices.Sorted(maps.Keys(assets)) {
		asset := assets[name]
		if isInstalled(directory, asset) {
			continue
		}
		content, err := fetchAsset(ctx, client, asset, downloads)
		if err != nil {
			return ctxerrors.Wrapf(err, "download %s", name)
		}
		contents[name] = content
	}

	if len(contents) > 0 {
		if err := os.MkdirAll(directory, directoryMode); err != nil {
			return ctxerrors.Wrap(err, "create font directory")
		}
	}
	for _, name := range slices.Sorted(maps.Keys(contents)) {
		if err := writeFont(filepath.Join(directory, name), contents[name]); err != nil {
			return ctxerrors.Wrapf(err, "save %s", name)
		}
	}
	for _, name := range fontFiles {
		if err := registerFont(filepath.Join(directory, name)); err != nil {
			return ctxerrors.Wrapf(err, "register %s", name)
		}
	}

	return nil
}

func isInstalled(directory string, asset fontAsset) bool {
	content, err := os.ReadFile(filepath.Join(directory, asset.name))

	return err == nil && sha256Hex(content) == asset.digest()
}

// fetchAsset downloads asset.url once per install, using cache for archives
// that hold several assets, and returns the verified file.
func fetchAsset(ctx context.Context, client *http.Client, asset fontAsset, cache map[string][]byte) ([]byte, error) {
	data, ok := cache[asset.url]
	if !ok {
		downloaded, err := download(ctx, client, asset.url)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "download font source")
		}
		data = downloaded
		cache[asset.url] = data
	}
	if sha256Hex(data) != asset.checksum {
		return nil, ctxerrors.Wrap(commerr.ErrDownloadFailed, "font download checksum is invalid")
	}
	if asset.member != "" {
		member, err := readMember(data, asset.member)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "extract font archive member")
		}
		data = member
	}
	if len(data) > maxFont || sha256Hex(data) != asset.digest() {
		return nil, ctxerrors.Wrap(commerr.ErrDownloadFailed, "font file size or checksum is invalid")
	}

	return data, nil
}

func download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "prepare font request")
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "request font")
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxDownload+1))
	if err := errors.Join(readErr, response.Body.Close()); err != nil {
		return nil, ctxerrors.Wrap(err, "read font response")
	}
	if response.StatusCode != http.StatusOK || len(data) > maxDownload {
		return nil, ctxerrors.Wrapf(commerr.ErrDownloadFailed, "status %d or size %d is invalid", response.StatusCode, len(data))
	}

	return data, nil
}

func readMember(archive []byte, name string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, ctxerrors.Wrap(err, "open font archive")
	}
	file, err := reader.Open(name)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "find font archive member")
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxFont+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, ctxerrors.Wrap(err, "read font archive member")
	}

	return data, nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func writeFont(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), tempFontPattern)
	if err != nil {
		return ctxerrors.Wrap(err, "create font temporary file")
	}
	_, writeErr := temp.Write(data)
	if err := errors.Join(writeErr, temp.Chmod(fontMode), temp.Close()); err != nil {
		return ctxerrors.Wrap(errors.Join(err, os.Remove(temp.Name())), "write font")
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return ctxerrors.Wrap(errors.Join(err, os.Remove(temp.Name())), "replace font")
	}

	return nil
}
