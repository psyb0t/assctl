package fonts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/psyb0t/assctl/pkg/ass"
)

func archive(t *testing.T, members map[string][]byte) ([]byte, map[string]string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	hashes := make(map[string]string, len(members))
	for name, content := range members {
		path := archivePrefix + "ttf/" + name
		if name == "LICENSE" {
			path = archivePrefix + name
		}
		file, err := writer.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(content); err != nil {
			t.Fatal(err)
		}
		hashes[name] = fmt.Sprintf("%x", sha256.Sum256(content))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes(), hashes
}

func TestInstallVerifiedFontsAndReuseOffline(t *testing.T) {
	t.Parallel()
	files := map[string][]byte{
		"LICENSE": []byte("test license"), "DejaVuSans.ttf": []byte("test regular font"),
		"DejaVuSans-Bold.ttf": []byte("test bold font"), "DejaVuSerif.ttf": []byte("test serif"),
		"DejaVuSansMono-Bold.ttf": []byte("test monospace"),
	}
	data, hashes := archive(t, files)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	directory := filepath.Join(t.TempDir(), "fonts", "assctl")
	i := installer{
		directory: directory, url: server.URL, checksum: fmt.Sprintf("%x", sha256.Sum256(data)),
		client: server.Client(), hashes: hashes,
	}
	if err := i.install(context.Background(), ass.Presets()); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("%s = %q, %v", name, got, err)
		}
	}
	server.Close()
	if err := i.install(context.Background(), ass.Presets()); err != nil {
		t.Fatalf("offline reinstallation failed: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("download count = %d", requests.Load())
	}
}

func TestInstallRejectsDownloadsBeforeChangingFonts(t *testing.T) {
	t.Parallel()
	valid, hashes := archive(t, map[string][]byte{"LICENSE": []byte("license"), "DejaVuSans.ttf": []byte("font")})
	preset, err := ass.LookupPreset("clean")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name     string
		status   int
		data     []byte
		checksum string
	}{
		{name: "HTTP failure", status: http.StatusServiceUnavailable, data: valid},
		{name: "wrong checksum", status: http.StatusOK, data: valid, checksum: "incorrect"},
		{name: "truncated ZIP", status: http.StatusOK, data: []byte("not a zip")},
		{name: "missing font", status: http.StatusOK, data: nil},
		{name: "oversized response", status: http.StatusOK, data: bytes.Repeat([]byte("x"), maxDownload+1)},
	}
	empty, _ := archive(t, map[string][]byte{"LICENSE": []byte("license")})
	testCases[3].data = empty
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.data)
			}))
			t.Cleanup(server.Close)
			directory := t.TempDir()
			path := filepath.Join(directory, "DejaVuSans.ttf")
			if err := os.WriteFile(path, []byte("original"), fontMode); err != nil {
				t.Fatal(err)
			}
			checksum := tc.checksum
			if checksum == "" {
				checksum = fmt.Sprintf("%x", sha256.Sum256(tc.data))
			}
			i := installer{directory: directory, url: server.URL, checksum: checksum, hashes: hashes, client: server.Client()}
			if err := i.install(context.Background(), []ass.Preset{preset}); err == nil {
				t.Fatal("invalid download accepted")
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "original" {
				t.Fatalf("existing font changed to %q, %v", got, err)
			}
		})
	}
}

func TestFontDirectoriesAndPresetSelection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ platform, data, local, want string }{
		{"linux", "", "", "/home/example/.local/share/fonts/assctl"},
		{"linux", "/data", "", "/data/fonts/assctl"},
		{"darwin", "", "", "/home/example/Library/Fonts/assctl"},
		{"windows", "", "/local", "/local/Microsoft/Windows/Fonts"},
		{"windows", "", "", "/home/example/AppData/Local/Microsoft/Windows/Fonts"},
		{"unknown", "", "", ""},
		{"linux", "relative", "", ""},
	} {
		got, err := fontDirectory(tc.platform, "/home/example", tc.data, tc.local)
		if tc.want == "" {
			if err == nil {
				t.Fatalf("accepted %s %s", tc.platform, tc.data)
			}
		} else if err != nil || got != tc.want {
			t.Fatalf("directory = %q, %v, want %q", got, err, tc.want)
		}
	}
	if _, err := presetFile(ass.Preset{Font: "Unknown"}); err == nil {
		t.Fatal("unknown font accepted")
	}
}

func TestInstallConfigurationAndCancellation(t *testing.T) {
	t.Setenv("HOME", "")
	if _, err := Install(context.Background(), ass.Presets()); err == nil {
		t.Fatal("missing home directory accepted")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "relative")
	if _, err := Install(context.Background(), ass.Presets()); err == nil {
		t.Fatal("relative data directory accepted")
	}
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Install(ctx, ass.Presets()); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled install returned %v", err)
	}
}

func TestFontWriteFailureCleansTemporaryFile(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	target := filepath.Join(directory, "font.ttf")
	if err := os.Mkdir(target, directoryMode); err != nil {
		t.Fatal(err)
	}
	if err := writeFont(target, []byte("font")); err == nil {
		t.Fatal("replaced a directory with a font")
	}
	if err := writeFont(filepath.Join(directory, "missing", "font.ttf"), []byte("font")); err == nil {
		t.Fatal("accepted missing parent directory")
	}
	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 1 || files[0].Name() != "font.ttf" {
		t.Fatalf("temporary file left behind: %v, %v", files, err)
	}
}

func TestInstallRepairsCorruptionAndHandlesWriteFailure(t *testing.T) {
	t.Parallel()
	data, hashes := archive(t, map[string][]byte{"LICENSE": []byte("license"), "DejaVuSans.ttf": []byte("font")})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
	t.Cleanup(server.Close)
	preset, err := ass.LookupPreset("clean")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	i := installer{
		directory: directory, url: server.URL, checksum: fmt.Sprintf("%x", sha256.Sum256(data)),
		hashes: hashes, client: server.Client(),
	}
	if err := os.WriteFile(filepath.Join(directory, "DejaVuSans.ttf"), []byte("broken"), fontMode); err != nil {
		t.Fatal(err)
	}
	if err := i.install(context.Background(), []ass.Preset{preset}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(directory, "DejaVuSans.ttf"))
	if err != nil || string(got) != "font" {
		t.Fatalf("repaired font = %q, %v", got, err)
	}
	i.directory = filepath.Join(directory, "DejaVuSans.ttf", "invalid")
	if err := i.install(context.Background(), []ass.Preset{preset}); err == nil {
		t.Fatal("invalid output directory accepted")
	}
	i.url = ":invalid"
	if err := i.install(context.Background(), []ass.Preset{preset}); err == nil {
		t.Fatal("malformed URL accepted")
	}
}
