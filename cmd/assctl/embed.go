package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/psyb0t/ctxerrors"

	"github.com/psyb0t/assctl/internal/pkg/fonts"
	"github.com/psyb0t/assctl/pkg/ass"
)

// fontOptions are the font files to embed in an output script.
type fontOptions struct {
	files       []string
	presetFonts bool
}

func (o fontOptions) enabled() bool { return len(o.files) > 0 || o.presetFonts }

// load reads the requested font files and, with presetFonts, installs the
// preset's verified fonts and reads them from the user's font directory.
func (o fontOptions) load(ctx context.Context, preset ass.Preset) ([]ass.EmbeddedFont, error) {
	paths := append([]string{}, o.files...)
	if o.presetFonts {
		directory, err := fonts.Install(ctx, []ass.Preset{preset})
		if err != nil {
			return nil, ctxerrors.Wrap(err, "install preset fonts")
		}
		names, err := fonts.FontFiles([]ass.Preset{preset})
		if err != nil {
			return nil, ctxerrors.Wrap(err, "list preset fonts")
		}
		for _, name := range names {
			paths = append(paths, filepath.Join(directory, name))
		}
	}

	embedded := make([]ass.EmbeddedFont, 0, len(paths))
	for _, path := range paths {
		data, err := readLimited(path)
		if err != nil {
			return nil, ctxerrors.Wrapf(err, "read font %s", path)
		}
		embedded = append(embedded, ass.EmbeddedFont{Name: filepath.Base(path), Data: data})
	}

	return embedded, nil
}

func readLimited(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "inspect file")
	}
	if info.Size() > maxSpecBytes {
		return nil, ctxerrors.Wrap(errFontTooLarge, "read font file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "read file")
	}

	return data, nil
}
