package main

import (
	"context"
	"io"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"

	"github.com/psyb0t/assctl/internal/pkg/burn"
	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	stream       = "-"
	validMessage = "valid ASS\n"
)

type cli struct {
	input       io.Reader
	output      io.Writer
	diagnostics io.Writer
}

// createOptions are the inputs of the create and captions commands.
type createOptions struct {
	spec, target, preset, theme, position string
	captions                              bool
	fonts                                 fontOptions
}

func (c cli) create(ctx context.Context, opts createOptions) error {
	script, preset, err := c.readScript(opts.spec, opts.captions, opts.preset, opts.theme)
	if err != nil {
		return ctxerrors.Wrap(err, "prepare script")
	}
	if opts.fonts.enabled() {
		embedded, err := opts.fonts.load(ctx, preset)
		if err != nil {
			return ctxerrors.Wrap(err, "load fonts to embed")
		}
		script.Fonts = append(script.Fonts, embedded...)
	}
	if opts.position != "" {
		position, err := ass.LookupTextPosition(opts.position)
		if err != nil {
			return ctxerrors.Wrap(err, "choose text position")
		}
		script, err = script.WithTextPosition(position)
		if err != nil {
			return ctxerrors.Wrap(err, "position script")
		}
	}
	data, err := script.MarshalText()
	if err != nil {
		return ctxerrors.Wrap(err, "format script")
	}
	if err := c.save(opts.target, string(data)); err != nil {
		return ctxerrors.Wrap(err, "save script")
	}

	return nil
}

// readScript returns the script to write and, for captions, the preset that styled it.
func (c cli) readScript(path string, captions bool, presetName, themePath string) (ass.Script, ass.Preset, error) {
	if !captions {
		script, err := decodeFile[ass.Script](path, c.input)
		if err != nil {
			return ass.Script{}, ass.Preset{}, ctxerrors.Wrap(err, "read script spec")
		}

		return script, ass.Preset{}, nil
	}
	if (presetName != "" && themePath != "") || (themePath == stream && path == stream) {
		return ass.Script{}, ass.Preset{}, ctxerrors.Wrap(commerr.ErrInvalidArgument, "choose one of --preset and --theme-file and use stdin once")
	}
	preset, err := c.loadPreset(presetName, themePath)
	if err != nil {
		return ass.Script{}, ass.Preset{}, ctxerrors.Wrap(err, "choose caption preset")
	}
	spec, err := decodeFile[ass.CaptionSpec](path, c.input)
	if err != nil {
		return ass.Script{}, ass.Preset{}, ctxerrors.Wrap(err, "read timed captions")
	}

	return preset.Captions(spec), preset, nil
}

func (c cli) inspect(source string, validate bool) error {
	document, err := c.read(source)
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	if !validate {
		if err := writeJSON(c.output, document.Contents()); err != nil {
			return ctxerrors.Wrap(err, "print ASS contents")
		}

		return nil
	}
	if err := document.Validate(); err != nil {
		return ctxerrors.Wrap(err, "validate ASS file")
	}
	if err := writeString(c.output, validMessage); err != nil {
		return ctxerrors.Wrap(err, "print validation result")
	}

	return nil
}

func (c cli) edit(spec, target, source string) error {
	if spec == "" || (spec == stream && source == stream) {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "provide --spec and use stdin for at most one input")
	}
	edits, err := decodeFile[ass.Edit](spec, c.input)
	if err != nil {
		return ctxerrors.Wrap(err, "load edits")
	}
	document, err := c.read(source)
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	updated, err := document.Apply(edits)
	if err != nil {
		return ctxerrors.Wrap(err, "apply edits")
	}
	if err := c.save(target, updated.String()); err != nil {
		return ctxerrors.Wrap(err, "save edited file")
	}

	return nil
}

// themeOptions are the inputs of the theme command.
type themeOptions struct {
	preset, custom, style, target, source, position string
	fonts                                           fontOptions
}

func (c cli) theme(ctx context.Context, opts themeOptions) error {
	preset, custom, style, target, source := opts.preset, opts.custom, opts.style, opts.target, opts.source
	if (preset != "" && custom != "") || (custom == stream && source == stream) {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "choose one of --preset and --theme-file and use stdin once")
	}
	look, err := c.loadPreset(preset, custom)
	if err != nil {
		return ctxerrors.Wrap(err, "load theme")
	}
	document, err := c.read(source)
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	updated, err := document.Theme(look, style)
	if err != nil {
		return ctxerrors.Wrap(err, "apply theme")
	}
	if opts.position != "" {
		position, err := ass.LookupTextPosition(opts.position)
		if err != nil {
			return ctxerrors.Wrap(err, "choose text position")
		}
		updated, err = updated.WithTextPosition(position, style)
		if err != nil {
			return ctxerrors.Wrap(err, "position themed styles")
		}
	}
	if opts.fonts.enabled() {
		embedded, err := opts.fonts.load(ctx, look)
		if err != nil {
			return ctxerrors.Wrap(err, "load fonts to embed")
		}
		updated, err = updated.EmbedFonts(embedded)
		if err != nil {
			return ctxerrors.Wrap(err, "embed fonts")
		}
	}
	if err := c.save(target, updated.String()); err != nil {
		return ctxerrors.Wrap(err, "save themed file")
	}

	return nil
}

func (c cli) position(name, style, target, source string) error {
	position, err := ass.LookupTextPosition(name)
	if err != nil {
		return ctxerrors.Wrap(err, "choose text position")
	}
	document, err := c.read(source)
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	updated, err := document.WithTextPosition(position, style)
	if err != nil {
		return ctxerrors.Wrap(err, "position styles")
	}
	if err := c.save(target, updated.String()); err != nil {
		return ctxerrors.Wrap(err, "save positioned file")
	}

	return nil
}

func (c cli) positions(showJSON bool) error {
	positions := ass.TextPositions()
	if showJSON {
		if err := writeJSON(c.output, positions); err != nil {
			return ctxerrors.Wrap(err, "print text positions")
		}

		return nil
	}
	for _, position := range positions {
		if err := writeString(c.output, position.Name+"\n"); err != nil {
			return ctxerrors.Wrap(err, "list text positions")
		}
	}

	return nil
}

func (c cli) tags(source string) error {
	override, err := decodeFile[ass.Override](source, c.input)
	if err != nil {
		return ctxerrors.Wrap(err, "read override JSON")
	}
	tags, err := override.Tags()
	if err != nil {
		return ctxerrors.Wrap(err, "format override tags")
	}
	if err := writeString(c.output, tags+"\n"); err != nil {
		return ctxerrors.Wrap(err, "print override tags")
	}

	return nil
}

func (c cli) loadPreset(name, path string) (ass.Preset, error) {
	if path != "" {
		preset, err := decodeFile[ass.Preset](path, c.input)
		if err != nil {
			return ass.Preset{}, ctxerrors.Wrap(err, "read theme file")
		}

		return preset, nil
	}
	if name == "" {
		name = ass.DefaultPreset
	}
	preset, err := ass.LookupPreset(name)
	if err != nil {
		return ass.Preset{}, ctxerrors.Wrap(err, "look up preset")
	}

	return preset, nil
}

func (c cli) presets(showJSON bool) error {
	presets := ass.Presets()
	if showJSON {
		if err := writeJSON(c.output, presets); err != nil {
			return ctxerrors.Wrap(err, "print presets")
		}

		return nil
	}
	for _, preset := range presets {
		if err := writeString(c.output, preset.Name+"\n"); err != nil {
			return ctxerrors.Wrap(err, "list presets")
		}
	}

	return nil
}

// safeArea prints every style or event that reaches into the platform's unsafe
// edges and fails when there is one.
func (c cli) safeArea(platform, source string) error {
	area, err := ass.LookupSafeArea(platform)
	if err != nil {
		return ctxerrors.Wrap(err, "choose safe area")
	}
	document, err := c.read(source)
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	issues, err := document.CheckSafeArea(area)
	if err != nil {
		return ctxerrors.Wrap(err, "check safe area")
	}
	if issues == nil {
		issues = []ass.SafeAreaIssue{}
	}
	if err := writeJSON(c.output, issues); err != nil {
		return ctxerrors.Wrap(err, "print safe area issues")
	}
	if len(issues) > 0 {
		return ctxerrors.Wrapf(commerr.ErrValidationFailed, "%d placement(s) reach outside the %s safe area", len(issues), area.Name)
	}

	return nil
}

func (c cli) safeAreas(showJSON bool) error {
	areas := ass.SafeAreas()
	if showJSON {
		if err := writeJSON(c.output, areas); err != nil {
			return ctxerrors.Wrap(err, "print safe areas")
		}

		return nil
	}
	for _, area := range areas {
		if err := writeString(c.output, area.Name+"\n"); err != nil {
			return ctxerrors.Wrap(err, "list safe areas")
		}
	}

	return nil
}

// burn draws subtitles into a copy of the input video. FFmpeg's progress goes
// to the diagnostics stream.
func (c cli) burn(ctx context.Context, input, subtitles, output string, crf *int) error {
	if subtitles == "" || output == "" || subtitles == stream || output == stream || input == stream {
		return ctxerrors.Wrap(commerr.ErrInvalidArgument, "burn needs an input video, --subtitles and --output files")
	}
	if err := burn.Burn(ctx, burn.Options{
		Input: input, Subtitles: subtitles, Output: output, CRF: crf, Diagnostics: c.diagnostics,
	}); err != nil {
		return ctxerrors.Wrap(err, "burn subtitles into video")
	}

	return nil
}
