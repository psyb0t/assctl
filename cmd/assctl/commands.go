package main

import (
	"io"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	stream       = "-"
	validMessage = "valid ASS\n"
)

type cli struct {
	input  io.Reader
	output io.Writer
}

func (c cli) create(specPath, target string, captions bool, presetName, themePath, positionName string) error {
	script, err := c.readScript(specPath, captions, presetName, themePath)
	if err != nil {
		return ctxerrors.Wrap(err, "prepare script")
	}
	if positionName != "" {
		position, err := ass.LookupTextPosition(positionName)
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
	if err := c.save(target, string(data)); err != nil {
		return ctxerrors.Wrap(err, "save script")
	}

	return nil
}

func (c cli) readScript(path string, captions bool, presetName, themePath string) (ass.Script, error) {
	if !captions {
		script, err := decodeFile[ass.Script](path, c.input)
		if err != nil {
			return ass.Script{}, ctxerrors.Wrap(err, "read script spec")
		}

		return script, nil
	}
	if (presetName != "" && themePath != "") || (themePath == stream && path == stream) {
		return ass.Script{}, ctxerrors.Wrap(commerr.ErrInvalidArgument, "choose one of --preset and --theme-file and use stdin once")
	}
	preset, err := c.loadPreset(presetName, themePath)
	if err != nil {
		return ass.Script{}, ctxerrors.Wrap(err, "choose caption preset")
	}
	spec, err := decodeFile[ass.CaptionSpec](path, c.input)
	if err != nil {
		return ass.Script{}, ctxerrors.Wrap(err, "read timed captions")
	}

	return preset.Captions(spec), nil
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

func (c cli) theme(preset, custom, style, target, source, positionName string) error {
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
	if positionName != "" {
		position, err := ass.LookupTextPosition(positionName)
		if err != nil {
			return ctxerrors.Wrap(err, "choose text position")
		}
		updated, err = updated.WithTextPosition(position, style)
		if err != nil {
			return ctxerrors.Wrap(err, "position themed styles")
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
