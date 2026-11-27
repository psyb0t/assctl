package fonts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	archivePrefix = "pack/"
	archivePath   = "/pack.zip"
	fontPath      = "/font"
	licensePath   = "/license"
	assetPath     = "/asset"

	fontFile    = "font.ttf"
	licenseFile = "LICENSE"
	sansFile    = "sans.ttf"
	sansBold    = "sans-bold.ttf"
	serifFile   = "serif.ttf"

	familySans  = "Test Sans"
	familySerif = "Test Serif"
)

var packFiles = map[string][]byte{
	sansFile:    []byte("regular sans"),
	sansBold:    []byte("bold sans"),
	serifFile:   []byte("serif"),
	licenseFile: []byte("license text"),
}

func digestOf(content []byte) string {
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}

func zipOf(t *testing.T, members map[string][]byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range members {
		file, err := writer.Create(name)
		require.NoError(t, err)
		_, err = file.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	return buffer.Bytes()
}

func packArchive(t *testing.T) []byte {
	t.Helper()

	members := make(map[string][]byte, len(packFiles))
	for name, content := range packFiles {
		members[archivePrefix+name] = content
	}

	return zipOf(t, members)
}

func directAsset(name, url string, content []byte) fontAsset {
	return fontAsset{name: name, url: url, checksum: digestOf(content)}
}

func memberAsset(name, url string, archive []byte, member string, content []byte) fontAsset {
	return fontAsset{
		name: name, url: url, checksum: digestOf(archive),
		member: member, fileChecksum: digestOf(content),
	}
}

func packSource(url string, archive []byte, font string) fontSource {
	return fontSource{
		font:    memberAsset(font, url, archive, archivePrefix+font, packFiles[font]),
		license: memberAsset(licenseFile, url, archive, archivePrefix+licenseFile, packFiles[licenseFile]),
	}
}

// packCatalog serves three fonts and one notice from a single archive.
func packCatalog(url string, archive []byte) catalog {
	return catalog{
		regular: map[string]fontSource{
			familySans:  packSource(url, archive, sansFile),
			familySerif: packSource(url, archive, serifFile),
		},
		bold: map[string]fontSource{
			familySans: packSource(url, archive, sansBold),
		},
	}
}

type route struct {
	status int
	body   []byte
}

type requestLog struct {
	mu    sync.Mutex
	paths []string
}

func (l *requestLog) add(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.paths = append(l.paths, path)
}

func (l *requestLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]string(nil), l.paths...)
}

// serve answers each known path with its route (status 200 when unset), 404
// for other paths, and records every requested path.
func serve(t *testing.T, routes map[string]route) (*httptest.Server, *requestLog) {
	t.Helper()

	requests := &requestLog{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.add(r.URL.Path)
		found, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)

			return
		}
		status := found.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = w.Write(found.body)
	}))
	t.Cleanup(server.Close)

	return server, requests
}

func newPack(t *testing.T) (*httptest.Server, *requestLog, catalog) {
	t.Helper()

	archive := packArchive(t)
	server, requests := serve(t, map[string]route{archivePath: {body: archive}})

	return server, requests, packCatalog(server.URL+archivePath, archive)
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}

func entryNames(t *testing.T, directory string) []string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), directoryMode))
	require.NoError(t, os.WriteFile(path, content, fontMode))
}

func TestFetchAsset(t *testing.T) {
	t.Parallel()

	font := []byte("font")
	atLimit := bytes.Repeat([]byte("f"), maxFont)
	overLimit := bytes.Repeat([]byte("f"), maxFont+1)
	fontZip := zipOf(t, map[string][]byte{archivePrefix + fontFile: font})
	atLimitZip := zipOf(t, map[string][]byte{archivePrefix + fontFile: atLimit})
	overLimitZip := zipOf(t, map[string][]byte{archivePrefix + fontFile: overLimit})
	member := archivePrefix + fontFile

	testCases := []struct {
		name    string
		status  int
		body    []byte
		asset   fontAsset
		want    []byte
		wantErr error
	}{
		{name: "direct file", body: font, asset: fontAsset{checksum: digestOf(font)}, want: font},
		{
			name: "direct file at size limit", body: atLimit,
			asset: fontAsset{checksum: digestOf(atLimit)}, want: atLimit,
		},
		{
			name: "direct file over size limit", body: overLimit,
			asset: fontAsset{checksum: digestOf(overLimit)}, wantErr: commerr.ErrDownloadFailed,
		},
		{
			name: "download over size limit", body: bytes.Repeat([]byte("a"), maxDownload+1),
			asset:   fontAsset{checksum: digestOf(bytes.Repeat([]byte("a"), maxDownload+1))},
			wantErr: commerr.ErrDownloadFailed,
		},
		{
			name: "HTTP failure", status: http.StatusForbidden, body: font,
			asset: fontAsset{checksum: digestOf(font)}, wantErr: commerr.ErrDownloadFailed,
		},
		{
			name: "wrong download checksum", body: font,
			asset: fontAsset{checksum: "wrong"}, wantErr: commerr.ErrDownloadFailed,
		},
		{
			name: "archive member", body: fontZip,
			asset: fontAsset{checksum: digestOf(fontZip), member: member, fileChecksum: digestOf(font)},
			want:  font,
		},
		{
			name: "archive member at size limit", body: atLimitZip,
			asset: fontAsset{checksum: digestOf(atLimitZip), member: member, fileChecksum: digestOf(atLimit)},
			want:  atLimit,
		},
		{
			name: "archive member over size limit", body: overLimitZip,
			asset: fontAsset{
				checksum: digestOf(overLimitZip), member: member, fileChecksum: digestOf(overLimit),
			},
			wantErr: commerr.ErrDownloadFailed,
		},
		{
			name: "wrong member checksum", body: fontZip,
			asset:   fontAsset{checksum: digestOf(fontZip), member: member, fileChecksum: "wrong"},
			wantErr: commerr.ErrDownloadFailed,
		},
		{
			name: "not a zip archive", body: font,
			asset:   fontAsset{checksum: digestOf(font), member: member, fileChecksum: digestOf(font)},
			wantErr: zip.ErrFormat,
		},
		{
			name: "missing archive member", body: fontZip,
			asset: fontAsset{
				checksum: digestOf(fontZip), member: archivePrefix + "missing.ttf", fileChecksum: digestOf(font),
			},
			wantErr: fs.ErrNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server, _ := serve(t, map[string]route{assetPath: {status: tc.status, body: tc.body}})
			asset := tc.asset
			asset.url = server.URL + assetPath

			got, err := fetchAsset(t.Context(), server.Client(), asset, make(map[string][]byte))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFetchAssetDownloadsSharedURLOnce(t *testing.T) {
	t.Parallel()

	archive := packArchive(t)
	server, requests := serve(t, map[string]route{archivePath: {body: archive}})
	source := server.URL + archivePath
	cache := make(map[string][]byte)

	for _, name := range []string{sansFile, serifFile, licenseFile} {
		asset := memberAsset(name, source, archive, archivePrefix+name, packFiles[name])
		got, err := fetchAsset(t.Context(), server.Client(), asset, cache)
		require.NoError(t, err)
		assert.Equal(t, packFiles[name], got)
	}
	assert.Equal(t, []string{archivePath}, requests.snapshot())

	tampered := memberAsset(sansFile, source, archive, archivePrefix+sansFile, packFiles[sansFile])
	tampered.checksum = "wrong"
	_, err := fetchAsset(t.Context(), server.Client(), tampered, cache)
	require.ErrorIs(t, err, commerr.ErrDownloadFailed)
	assert.Len(t, requests.snapshot(), 1)
}

func TestDownloadTransportFailures(t *testing.T) {
	t.Parallel()

	t.Run("truncated body", func(t *testing.T) {
		t.Parallel()

		body := []byte("short")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)+100))
			_, _ = w.Write(body)
		}))
		t.Cleanup(server.Close)

		_, err := download(t.Context(), server.Client(), server.URL)
		require.Error(t, err)
	})

	t.Run("malformed URL", func(t *testing.T) {
		t.Parallel()

		_, err := download(t.Context(), http.DefaultClient, ":invalid")
		var urlErr *url.Error
		require.ErrorAs(t, err, &urlErr)
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()

		server, requests := serve(t, map[string]route{assetPath: {body: []byte("font")}})

		_, err := download(canceledContext(t), server.Client(), server.URL+assetPath)
		require.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, requests.snapshot())
	})

	t.Run("unreachable server", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.NotFoundHandler())
		address := server.URL
		server.Close()

		_, err := download(t.Context(), http.DefaultClient, address)
		require.Error(t, err)
	})
}

func TestInstallVerifiedFontsAndReuseOffline(t *testing.T) {
	t.Parallel()

	server, _, fonts := newPack(t)
	directory := filepath.Join(t.TempDir(), "fonts", "assctl")
	presets := []ass.Preset{
		{Font: familySans},
		{Font: familySans, Bold: true},
		{Font: familySerif},
	}

	require.NoError(t, install(t.Context(), directory, presets, server.Client(), fonts))
	for name, want := range packFiles {
		got, err := os.ReadFile(filepath.Join(directory, name))
		require.NoError(t, err)
		assert.Equal(t, want, got, name)
	}

	server.Close()
	require.NoError(t, install(canceledContext(t), directory, presets, server.Client(), fonts))
}

func TestInstallDownloadsSharedArchiveOnce(t *testing.T) {
	t.Parallel()

	server, requests, fonts := newPack(t)
	presets := []ass.Preset{
		{Font: familySans},
		{Font: familySans, Bold: true},
		{Font: familySerif},
		{Font: familySerif},
	}

	require.NoError(t, install(t.Context(), t.TempDir(), presets, server.Client(), fonts))
	assert.Equal(t, []string{archivePath}, requests.snapshot())
}

func TestInstallIgnoresUnrelatedArchiveMembers(t *testing.T) {
	t.Parallel()

	members := map[string][]byte{"../../escape.ttf": []byte("unrelated")}
	for name, content := range packFiles {
		members[archivePrefix+name] = content
	}
	archive := zipOf(t, members)
	server, _ := serve(t, map[string]route{archivePath: {body: archive}})
	root := t.TempDir()
	directory := filepath.Join(root, "nested", "fonts")

	err := install(
		t.Context(), directory, []ass.Preset{{Font: familySans}},
		server.Client(), packCatalog(server.URL+archivePath, archive),
	)
	require.NoError(t, err)
	assert.Equal(t, []string{licenseFile, sansFile}, entryNames(t, directory))
	assert.NoFileExists(t, filepath.Join(root, "escape.ttf"))
}

func TestInstallDownloadsOnlyMissingAssets(t *testing.T) {
	t.Parallel()

	fontContent := []byte("font data")
	licenseContent := []byte("license data")

	testCases := []struct {
		name         string
		installed    map[string][]byte
		wantRequests []string
	}{
		{
			name:         "nothing installed",
			wantRequests: []string{fontPath, licensePath},
		},
		{
			name:         "font installed",
			installed:    map[string][]byte{fontFile: fontContent},
			wantRequests: []string{licensePath},
		},
		{
			name:         "license installed",
			installed:    map[string][]byte{licenseFile: licenseContent},
			wantRequests: []string{fontPath},
		},
		{
			name:         "corrupt font",
			installed:    map[string][]byte{fontFile: []byte("broken"), licenseFile: licenseContent},
			wantRequests: []string{fontPath},
		},
		{
			name:      "everything installed",
			installed: map[string][]byte{fontFile: fontContent, licenseFile: licenseContent},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server, requests := serve(t, map[string]route{
				fontPath:    {body: fontContent},
				licensePath: {body: licenseContent},
			})
			fonts := catalog{regular: map[string]fontSource{familySans: {
				font:    directAsset(fontFile, server.URL+fontPath, fontContent),
				license: directAsset(licenseFile, server.URL+licensePath, licenseContent),
			}}}
			directory := t.TempDir()
			for name, content := range tc.installed {
				writeFile(t, filepath.Join(directory, name), content)
			}

			err := install(t.Context(), directory, []ass.Preset{{Font: familySans}}, server.Client(), fonts)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantRequests, requests.snapshot())
			for name, want := range map[string][]byte{fontFile: fontContent, licenseFile: licenseContent} {
				got, err := os.ReadFile(filepath.Join(directory, name))
				require.NoError(t, err)
				assert.Equal(t, want, got, name)
			}
		})
	}
}

func TestInstallRejectsDownloadsBeforeChangingFonts(t *testing.T) {
	t.Parallel()

	fontContent := []byte("font data")
	licenseContent := []byte("license data")

	testCases := []struct {
		name  string
		route route
	}{
		{name: "HTTP failure", route: route{status: http.StatusServiceUnavailable, body: fontContent}},
		{name: "missing file", route: route{status: http.StatusNotFound}},
		{name: "wrong checksum", route: route{body: []byte("tampered")}},
		{name: "oversized response", route: route{body: bytes.Repeat([]byte("x"), maxDownload+1)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server, _ := serve(t, map[string]route{
				fontPath:    tc.route,
				licensePath: {body: licenseContent},
			})
			fonts := catalog{regular: map[string]fontSource{familySans: {
				font:    directAsset(fontFile, server.URL+fontPath, fontContent),
				license: directAsset(licenseFile, server.URL+licensePath, licenseContent),
			}}}
			directory := t.TempDir()
			installed := filepath.Join(directory, fontFile)
			writeFile(t, installed, []byte("original"))

			err := install(t.Context(), directory, []ass.Preset{{Font: familySans}}, server.Client(), fonts)
			require.ErrorIs(t, err, commerr.ErrDownloadFailed)

			got, err := os.ReadFile(installed)
			require.NoError(t, err)
			assert.Equal(t, []byte("original"), got)
			assert.Equal(t, []string{fontFile}, entryNames(t, directory))
		})
	}
}

func TestInstallRejectsArchiveWithoutNotice(t *testing.T) {
	t.Parallel()

	archive := zipOf(t, map[string][]byte{archivePrefix + sansFile: packFiles[sansFile]})
	server, _ := serve(t, map[string]route{archivePath: {body: archive}})
	directory := filepath.Join(t.TempDir(), "fonts")

	err := install(
		t.Context(), directory, []ass.Preset{{Font: familySans}},
		server.Client(), packCatalog(server.URL+archivePath, archive),
	)
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.NoDirExists(t, directory)
}

func TestInstallWithoutPresetsChangesNothing(t *testing.T) {
	t.Parallel()

	server, requests, fonts := newPack(t)
	directory := filepath.Join(t.TempDir(), "fonts")

	require.NoError(t, install(t.Context(), directory, nil, server.Client(), fonts))
	assert.Empty(t, requests.snapshot())
	assert.NoDirExists(t, directory)
}

func TestInstallRepairsCorruptedFonts(t *testing.T) {
	t.Parallel()

	server, requests, fonts := newPack(t)
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, sansFile), []byte("broken"))
	writeFile(t, filepath.Join(directory, licenseFile), []byte("broken"))

	require.NoError(t, install(t.Context(), directory, []ass.Preset{{Font: familySans}}, server.Client(), fonts))
	assert.Len(t, requests.snapshot(), 1)
	for _, name := range []string{sansFile, licenseFile} {
		got, err := os.ReadFile(filepath.Join(directory, name))
		require.NoError(t, err)
		assert.Equal(t, packFiles[name], got, name)
	}
	assert.Equal(t, []string{licenseFile, sansFile}, entryNames(t, directory))
}

func TestInstallFailures(t *testing.T) {
	t.Parallel()

	presets := []ass.Preset{{Font: familySans}}

	t.Run("unknown font family", func(t *testing.T) {
		t.Parallel()

		server, requests, fonts := newPack(t)
		directory := filepath.Join(t.TempDir(), "fonts")

		err := install(t.Context(), directory, []ass.Preset{{Font: "Missing"}}, server.Client(), fonts)
		require.ErrorIs(t, err, commerr.ErrUnsupported)
		assert.Empty(t, requests.snapshot())
		assert.NoDirExists(t, directory)
	})

	t.Run("directory under a file", func(t *testing.T) {
		t.Parallel()

		server, _, fonts := newPack(t)
		blocker := filepath.Join(t.TempDir(), "blocker")
		writeFile(t, blocker, []byte("file"))

		err := install(t.Context(), filepath.Join(blocker, "fonts"), presets, server.Client(), fonts)
		require.Error(t, err)
	})

	t.Run("font path is a directory", func(t *testing.T) {
		t.Parallel()

		server, _, fonts := newPack(t)
		directory := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(directory, sansFile), directoryMode))

		err := install(t.Context(), directory, presets, server.Client(), fonts)
		require.Error(t, err)
		assert.Equal(t, []string{licenseFile, sansFile}, entryNames(t, directory))
	})

	t.Run("malformed source URL keeps installed fonts", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		installed := filepath.Join(directory, sansFile)
		writeFile(t, installed, []byte("original"))
		fonts := packCatalog(":invalid", packArchive(t))

		err := install(t.Context(), directory, presets, http.DefaultClient, fonts)
		require.Error(t, err)
		got, err := os.ReadFile(installed)
		require.NoError(t, err)
		assert.Equal(t, []byte("original"), got)
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()

		server, requests, fonts := newPack(t)
		directory := filepath.Join(t.TempDir(), "fonts")

		err := install(canceledContext(t), directory, presets, server.Client(), fonts)
		require.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, requests.snapshot())
		assert.NoDirExists(t, directory)
	})
}

func TestFontDirectory(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	data := t.TempDir()
	local := t.TempDir()

	testCases := []struct {
		name     string
		platform string
		env      environment
		want     string
		wantErr  error
	}{
		{
			name: "linux default", platform: "linux",
			want: filepath.Join(home, ".local", "share", "fonts", "assctl"),
		},
		{
			name: "linux XDG_DATA_HOME", platform: "linux", env: environment{DataHome: data},
			want: filepath.Join(data, "fonts", "assctl"),
		},
		{
			name: "freebsd default", platform: "freebsd",
			want: filepath.Join(home, ".local", "share", "fonts", "assctl"),
		},
		{
			name: "freebsd XDG_DATA_HOME", platform: "freebsd", env: environment{DataHome: data},
			want: filepath.Join(data, "fonts", "assctl"),
		},
		{
			name: "darwin", platform: "darwin",
			want: filepath.Join(home, "Library", "Fonts", "assctl"),
		},
		{
			name: "darwin ignores XDG_DATA_HOME", platform: "darwin", env: environment{DataHome: "relative"},
			want: filepath.Join(home, "Library", "Fonts", "assctl"),
		},
		{
			name: "windows default", platform: "windows",
			want: filepath.Join(home, "AppData", "Local", "Microsoft", "Windows", "Fonts"),
		},
		{
			name: "windows LOCALAPPDATA", platform: "windows", env: environment{LocalAppData: local},
			want: filepath.Join(local, "Microsoft", "Windows", "Fonts"),
		},
		{
			name: "linux relative XDG_DATA_HOME", platform: "linux", env: environment{DataHome: "relative"},
			wantErr: commerr.ErrInvalidValue,
		},
		{name: "unsupported platform", platform: "plan9", wantErr: commerr.ErrUnsupported},
		{name: "empty platform", wantErr: commerr.ErrUnsupported},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := fontDirectory(tc.platform, home, tc.env)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, got)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// The Install tests change HOME and XDG_DATA_HOME, so they run serially.
func TestInstallConfiguration(t *testing.T) {
	if runtime.GOOS != platformLinux {
		t.Skip("HOME and XDG_DATA_HOME select the font directory on Linux only")
	}

	t.Run("empty home directory", func(t *testing.T) {
		t.Setenv("HOME", "")
		t.Setenv("XDG_DATA_HOME", "")

		directory, err := Install(t.Context(), nil)
		require.Error(t, err)
		assert.Empty(t, directory)
	})

	t.Run("relative XDG_DATA_HOME", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", "relative")

		directory, err := Install(t.Context(), nil)
		require.ErrorIs(t, err, commerr.ErrInvalidValue)
		assert.Empty(t, directory)
	})

	t.Run("default directory under home", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_DATA_HOME", "")

		directory, err := Install(t.Context(), nil)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".local", "share", "fonts", "assctl"), directory)
	})

	t.Run("XDG_DATA_HOME directory", func(t *testing.T) {
		data := t.TempDir()
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", data)

		directory, err := Install(t.Context(), nil)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(data, "fonts", "assctl"), directory)
	})

	t.Run("unknown font family", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", t.TempDir())

		directory, err := Install(t.Context(), []ass.Preset{{Font: "Missing"}})
		require.ErrorIs(t, err, commerr.ErrUnsupported)
		assert.Empty(t, directory)
	})

	t.Run("canceled context", func(t *testing.T) {
		data := t.TempDir()
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_DATA_HOME", data)

		directory, err := Install(canceledContext(t), ass.Presets())
		require.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, directory)
		assert.NoDirExists(t, filepath.Join(data, "fonts"))
	})
}

func TestWriteFont(t *testing.T) {
	t.Parallel()

	t.Run("creates file with font mode", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), fontFile)

		require.NoError(t, writeFont(path, []byte("font")))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, []byte("font"), got)
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(fontMode), info.Mode().Perm())
	})

	t.Run("replaces existing file without leftovers", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		path := filepath.Join(directory, fontFile)
		writeFile(t, path, []byte("old"))

		require.NoError(t, writeFont(path, []byte("new")))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, []byte("new"), got)
		assert.Equal(t, []string{fontFile}, entryNames(t, directory))
	})

	t.Run("directory target removes temporary file", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		target := filepath.Join(directory, fontFile)
		require.NoError(t, os.Mkdir(target, directoryMode))

		require.Error(t, writeFont(target, []byte("font")))
		assert.Equal(t, []string{fontFile}, entryNames(t, directory))
	})

	t.Run("missing parent directory", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()

		require.ErrorIs(t, writeFont(filepath.Join(directory, "missing", fontFile), []byte("font")), fs.ErrNotExist)
		assert.Empty(t, entryNames(t, directory))
	})
}
