// Package fonts installs the font files required by assctl's built-in presets.
package fonts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/psyb0t/ctxerrors"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	archiveURL      = "https://github.com/dejavu-fonts/dejavu-fonts/releases/download/version_2_37/dejavu-fonts-ttf-2.37.zip"
	archiveSHA      = "7576310b219e04159d35ff61dd4a4ec4cdba4f35c00e002a136f00e96a908b0a"
	archivePrefix   = "dejavu-fonts-ttf-2.37/"
	maxDownload     = 8 << 20
	maxFont         = 2 << 20
	downloadTimeout = 60 * time.Second
	fontMode        = 0o644
	directoryMode   = 0o755
)

var errFont = errors.New("cannot install preset font")

var fileHashes = map[string]string{
	"DejaVuSans.ttf":          "7da195a74c55bef988d0d48f9508bd5d849425c1770dba5d7bfc6ce9ed848954",
	"DejaVuSans-Bold.ttf":     "e6476c1b80502924294eed40894c5b18e06c181444ca953e5334262df9c27724",
	"DejaVuSerif.ttf":         "42d1edeb7952f31b1f96d767ed7030b08a39e0c372b0071641518864e2bffb51",
	"DejaVuSansMono-Bold.ttf": "bce60f1b4421acd9ea51ba6623d7024ecbe6817a953e3654df62a5e6bdf8f769",
	"LICENSE":                 "7a083b136e64d064794c3419751e5c7dd10d2f64c108fe5ba161eae5e5958a93",
}

func presetFile(preset ass.Preset) (string, error) {
	switch preset.Font {
	case "DejaVu Sans":
		if preset.Bold {
			return "DejaVuSans-Bold.ttf", nil
		}
		return "DejaVuSans.ttf", nil
	case "DejaVu Serif":
		return "DejaVuSerif.ttf", nil
	case "DejaVu Sans Mono":
		return "DejaVuSansMono-Bold.ttf", nil
	default:
		return "", ctxerrors.Wrapf(errFont, "no bundled source for font %q", preset.Font)
	}
}

// Install verifies and installs the fonts for the selected presets in the user's font directory.
func Install(ctx context.Context, presets []ass.Preset) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ctxerrors.Wrap(err, "locate home directory")
	}
	directory, err := fontDirectory(runtime.GOOS, home, os.Getenv("XDG_DATA_HOME"), os.Getenv("LOCALAPPDATA"))
	if err != nil {
		return "", ctxerrors.Wrap(err, "locate user fonts")
	}
	client := &http.Client{Timeout: downloadTimeout}
	installer := installer{directory: directory, url: archiveURL, checksum: archiveSHA, client: client, hashes: fileHashes}
	if err := installer.install(ctx, presets); err != nil {
		return "", ctxerrors.Wrap(err, "install preset fonts")
	}
	return directory, nil
}

func fontDirectory(platform, home, data, local string) (string, error) {
	switch platform {
	case "linux", "freebsd":
		if data == "" {
			data = filepath.Join(home, ".local", "share")
		}
		if !filepath.IsAbs(data) {
			return "", ctxerrors.Wrap(errFont, "XDG_DATA_HOME must be absolute")
		}
		return filepath.Join(data, "fonts", "assctl"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Fonts", "assctl"), nil
	case "windows":
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(local, "Microsoft", "Windows", "Fonts"), nil
	default:
		return "", ctxerrors.Wrapf(errFont, "unsupported font directory on %s", platform)
	}
}

type installer struct {
	directory string
	url       string
	checksum  string
	client    *http.Client
	hashes    map[string]string
}

func (i installer) install(ctx context.Context, presets []ass.Preset) error {
	files := []string{"LICENSE"}
	for _, preset := range presets {
		name, err := presetFile(preset)
		if err != nil {
			return ctxerrors.Wrap(err, "select font file")
		}
		if !slices.Contains(files, name) {
			files = append(files, name)
		}
	}
	missing := false
	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(i.directory, name))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(content)) != i.hashes[name] {
			missing = true
		}
	}
	if !missing {
		return nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, i.url, nil)
	if err != nil {
		return ctxerrors.Wrap(err, "prepare font download")
	}
	response, err := i.client.Do(request)
	if err != nil {
		return ctxerrors.Wrap(err, "download preset fonts")
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxDownload+1))
	if err := errors.Join(readErr, response.Body.Close()); err != nil {
		return ctxerrors.Wrap(err, "read font download")
	}
	if response.StatusCode != http.StatusOK || len(data) > maxDownload ||
		fmt.Sprintf("%x", sha256.Sum256(data)) != i.checksum {
		return ctxerrors.Wrap(errFont, "download status, size, or checksum is invalid")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ctxerrors.Wrap(err, "read font archive")
	}
	contents := make(map[string][]byte, len(files))
	for _, name := range files {
		path := archivePrefix + "ttf/" + name
		if name == "LICENSE" {
			path = archivePrefix + name
		}
		file, err := reader.Open(path)
		if err != nil {
			return ctxerrors.Wrap(err, "locate archived font")
		}
		content, readErr := io.ReadAll(io.LimitReader(file, maxFont+1))
		if err := errors.Join(readErr, file.Close()); err != nil {
			return ctxerrors.Wrap(err, "read archived font")
		}
		if len(content) > maxFont || fmt.Sprintf("%x", sha256.Sum256(content)) != i.hashes[name] {
			return ctxerrors.Wrap(errFont, "font size or checksum is invalid")
		}
		contents[name] = content
	}
	if err := os.MkdirAll(i.directory, directoryMode); err != nil {
		return ctxerrors.Wrap(err, "create font directory")
	}
	for _, name := range files {
		if err := writeFont(filepath.Join(i.directory, name), contents[name]); err != nil {
			return ctxerrors.Wrap(err, "save font")
		}
	}
	return nil
}

func writeFont(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".assctl-font-*")
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
