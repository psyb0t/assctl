package main

import (
	"errors"
	"flag"
	"io"

	"github.com/psyb0t/ctxerrors"

	"github.com/psyb0t/assctl/pkg/ass"
)

const (
	presetDefault = "black-gold"
	stream        = "-"
)

var errUsage = errors.New("invalid command arguments")

type cli struct {
	input  io.Reader
	output io.Writer
}

func parseArgs(flags *flag.FlagSet, args []string, want int) error {
	if err := flags.Parse(args); err != nil {
		return ctxerrors.Wrap(err, "parse flags")
	}
	if flags.NArg() != want {
		return ctxerrors.Wrapf(errUsage, "expected %d positional arguments", want)
	}
	return nil
}

func (c cli) create(flags *flag.FlagSet, args []string, captions bool) error {
	specPath := flags.String("spec", stream, "JSON script file or - for stdin")
	target := flags.String("output", stream, "ASS output file or - for stdout")
	var presetName, themePath string
	if captions {
		flags.StringVar(&presetName, "preset", "", "built-in preset name (default "+presetDefault+")")
		flags.StringVar(&themePath, "theme-file", "", "JSON file with custom preset values")
	}
	if err := parseArgs(flags, args, 0); err != nil {
		return ctxerrors.Wrap(err, "create subtitles")
	}
	script, err := c.readScript(*specPath, captions, presetName, themePath)
	if err != nil {
		return ctxerrors.Wrap(err, "prepare script")
	}
	data, err := script.MarshalText()
	if err != nil {
		return ctxerrors.Wrap(err, "format script")
	}

	return c.save(*target, string(data))
}

func (c cli) readScript(path string, captions bool, presetName, themePath string) (ass.Script, error) {
	if !captions {
		return decodeFile[ass.Script](path, c.input)
	}
	if (presetName != "" && themePath != "") || (themePath == stream && path == stream) {
		return ass.Script{}, ctxerrors.Wrap(errUsage, "choose one of --preset and --theme-file and use stdin once")
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

func (c cli) inspect(flags *flag.FlagSet, args []string, validate bool) error {
	if err := parseArgs(flags, args, 1); err != nil {
		return ctxerrors.Wrap(err, "read ASS input")
	}
	document, err := c.read(flags.Arg(0))
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	if !validate {
		return writeJSON(c.output, document.Contents())
	}
	if err := document.Validate(); err != nil {
		return ctxerrors.Wrap(err, "validate ASS file")
	}

	return writeString(c.output, "valid ASS\n")
}

func (c cli) edit(flags *flag.FlagSet, args []string) error {
	spec := flags.String("spec", "", "JSON file with edits")
	target := flags.String("output", stream, "ASS output file or - for stdout")
	if err := parseArgs(flags, args, 1); err != nil {
		return ctxerrors.Wrap(err, "edit subtitles")
	}
	source := flags.Arg(0)
	if *spec == "" || (*spec == stream && source == stream) {
		return ctxerrors.Wrap(errUsage, "provide --spec and use stdin for at most one input")
	}
	edits, err := decodeFile[ass.Edit](*spec, c.input)
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

	return c.save(*target, updated.String())
}

func (c cli) theme(flags *flag.FlagSet, args []string) error {
	preset := flags.String("preset", "", "built-in preset name")
	custom := flags.String("theme-file", "", "JSON file with custom preset values")
	style := flags.String("style", "", "theme one named style, or omit to theme every style")
	target := flags.String("output", stream, "ASS output file or - for stdout")
	if err := parseArgs(flags, args, 1); err != nil {
		return ctxerrors.Wrap(err, "theme subtitles")
	}
	source := flags.Arg(0)
	if (*preset != "" && *custom != "") || (*custom == stream && source == stream) {
		return ctxerrors.Wrap(errUsage, "choose one of --preset and --theme-file and use stdin once")
	}
	look, err := c.loadPreset(*preset, *custom)
	if err != nil {
		return ctxerrors.Wrap(err, "load theme")
	}
	document, err := c.read(source)
	if err != nil {
		return ctxerrors.Wrap(err, "load ASS file")
	}
	updated, err := document.Theme(look, *style)
	if err != nil {
		return ctxerrors.Wrap(err, "apply theme")
	}

	return c.save(*target, updated.String())
}

func (c cli) loadPreset(name, path string) (ass.Preset, error) {
	if path != "" {
		return decodeFile[ass.Preset](path, c.input)
	}
	if name == "" {
		name = presetDefault
	}

	return ass.LookupPreset(name)
}

func (c cli) presets(flags *flag.FlagSet, args []string) error {
	showJSON := flags.Bool("json", false, "include fonts, colors, and other settings")
	if err := parseArgs(flags, args, 0); err != nil {
		return ctxerrors.Wrap(err, "list presets")
	}
	presets := ass.Presets()
	if *showJSON {
		return writeJSON(c.output, presets)
	}
	for _, preset := range presets {
		if err := writeString(c.output, preset.Name+"\n"); err != nil {
			return ctxerrors.Wrap(err, "list presets")
		}
	}

	return nil
}
