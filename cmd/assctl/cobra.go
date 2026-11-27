package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"

	"github.com/psyb0t/assctl/internal/pkg/fonts"
	"github.com/psyb0t/assctl/pkg/ass"
)

func exactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(count)(cmd, args); err != nil {
			return ctxerrors.Wrap(errors.Join(commerr.ErrInvalidArgument, err), "check arguments")
		}

		return nil
	}
}

func (c cli) rootCommand() *cobra.Command {
	var showVersion bool
	root := &cobra.Command{
		Use: "assctl", Short: "Create, edit, validate, and theme ASS subtitles",
		SilenceErrors: true, SilenceUsage: true,
		Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				return c.version()
			}

			return showHelp(cmd)
		},
	}
	root.Flags().BoolVar(&showVersion, "version", false, "Print the installed version")
	root.AddCommand(
		c.createCommand(false), c.createCommand(true),
		c.inspectCommand(false), c.inspectCommand(true),
		c.editCommand(), c.themeCommand(), c.presetsCommand(),
		c.fontsCommand(),
		c.positionsCommand(), c.positionCommand(), c.tagsCommand(),
		c.safeAreaCommand(), c.safeAreasCommand(), c.burnCommand(),
		&cobra.Command{
			Use: "version", Short: "Print the installed version", Args: exactArgs(0),
			RunE: func(_ *cobra.Command, _ []string) error { return c.version() },
		},
	)
	root.SetOut(c.output)
	root.InitDefaultCompletionCmd()
	for _, command := range root.Commands() {
		if command.Name() == "completion" {
			command.Args = exactArgs(0)
			command.RunE = func(cmd *cobra.Command, _ []string) error { return showHelp(cmd) }
		}
	}
	return root
}

func (c cli) fontsCommand() *cobra.Command {
	var name string
	command := &cobra.Command{
		Use: "fonts-install", Short: "Download and install the built-in preset fonts for your user", Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			presets := ass.Presets()
			if name != "" {
				preset, err := ass.LookupPreset(name)
				if err != nil {
					return ctxerrors.Wrap(err, "choose font preset")
				}
				presets = []ass.Preset{preset}
			}
			directory, err := fonts.Install(cmd.Context(), presets)
			if err != nil {
				return ctxerrors.Wrap(err, "install fonts")
			}
			if err := writeString(c.output, directory+"\n"); err != nil {
				return ctxerrors.Wrap(err, "print font directory")
			}

			return nil
		},
	}
	command.Flags().StringVar(&name, "preset", "", "Install one preset's fonts, or omit for every built-in preset")
	return command
}

func (c cli) createCommand(captions bool) *cobra.Command {
	name, description := "create", "Create an ASS script from typed JSON"
	if captions {
		name, description = "captions", "Style timed captions using a preset or custom theme"
	}
	opts := createOptions{captions: captions}
	command := &cobra.Command{
		Use: name, Short: description, Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("text-position") && opts.position == "" {
				return ctxerrors.Wrap(commerr.ErrInvalidArgument, "--text-position cannot be empty")
			}

			return c.create(cmd.Context(), opts)
		},
	}
	command.Flags().StringVar(&opts.spec, "spec", stream, "JSON file or - for stdin")
	command.Flags().StringVar(&opts.target, "output", stream, "ASS file or - for stdout")
	command.Flags().StringVar(&opts.position, "text-position", "", "Named style placement; existing event positions are preserved")
	command.Flags().StringArrayVar(&opts.fonts.files, "embed-font", nil, "Font file to embed in the script; repeat for more")
	if captions {
		command.Flags().StringVar(&opts.preset, "preset", "", "Built-in preset (default "+ass.DefaultPreset+")")
		command.Flags().StringVar(&opts.theme, "theme-file", "", "JSON file with custom preset values")
		command.Flags().BoolVar(&opts.fonts.presetFonts, "embed-preset-fonts", false, "Install the preset's verified fonts and embed them")
	}
	return command
}

func (c cli) inspectCommand(validate bool) *cobra.Command {
	name, description := "inspect", "Print ASS script settings, styles, and events as JSON"
	if validate {
		name, description = "validate", "Check ASS structure, style references, and event timing"
	}
	return &cobra.Command{
		Use: name + " INPUT", Short: description, Args: exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error { return c.inspect(args[0], validate) },
	}
}

func (c cli) editCommand() *cobra.Command {
	var spec, output string
	command := &cobra.Command{
		Use: "edit INPUT", Short: "Apply JSON edits to an existing ASS file", Args: exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error { return c.edit(spec, output, args[0]) },
	}
	command.Flags().StringVar(&spec, "spec", "", "JSON file with edits")
	command.Flags().StringVar(&output, "output", stream, "ASS file or - for stdout")
	return command
}

func (c cli) themeCommand() *cobra.Command {
	var opts themeOptions
	command := &cobra.Command{
		Use: "theme INPUT", Short: "Apply a preset or custom theme to existing styles", Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("text-position") && opts.position == "" {
				return ctxerrors.Wrap(commerr.ErrInvalidArgument, "--text-position cannot be empty")
			}
			opts.source = args[0]

			return c.theme(cmd.Context(), opts)
		},
	}
	command.Flags().StringVar(&opts.preset, "preset", "", "Built-in preset (default "+ass.DefaultPreset+")")
	command.Flags().StringVar(&opts.custom, "theme-file", "", "JSON file with custom preset values")
	command.Flags().StringVar(&opts.style, "style", "", "Theme one named style, or omit to theme every style")
	command.Flags().StringVar(&opts.target, "output", stream, "ASS file or - for stdout")
	command.Flags().StringVar(&opts.position, "text-position", "", "Named style placement; existing event positions are preserved")
	command.Flags().StringArrayVar(&opts.fonts.files, "embed-font", nil, "Font file to embed in the output; repeat for more")
	command.Flags().BoolVar(&opts.fonts.presetFonts, "embed-preset-fonts", false, "Install the preset's verified fonts and embed them")
	return command
}

func (c cli) positionsCommand() *cobra.Command {
	var showJSON bool
	command := &cobra.Command{
		Use: "positions", Short: "List named relative text placements", Args: exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error { return c.positions(showJSON) },
	}
	command.Flags().BoolVar(&showJSON, "json", false, "Include alignments and normalized margins")
	return command
}

func (c cli) positionCommand() *cobra.Command {
	var position, style, output string
	command := &cobra.Command{
		Use: "position INPUT", Short: "Change style placement while preserving appearance and event overrides", Args: exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return c.position(position, style, output, args[0])
		},
	}
	command.Flags().StringVar(&position, "text-position", "", "Named text position (required)")
	command.Flags().StringVar(&style, "style", "", "Position one style, or omit for every style")
	command.Flags().StringVar(&output, "output", stream, "ASS file or - for stdout")
	return command
}

func (c cli) tagsCommand() *cobra.Command {
	var spec string
	command := &cobra.Command{
		Use: "tags", Short: "Encode typed override JSON as an ASS tag block", Args: exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error { return c.tags(spec) },
	}
	command.Flags().StringVar(&spec, "spec", stream, "Override JSON file or - for stdin")
	return command
}

func (c cli) presetsCommand() *cobra.Command {
	var showJSON bool
	command := &cobra.Command{
		Use: "presets", Short: "List built-in subtitle looks", Args: exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error { return c.presets(showJSON) },
	}
	command.Flags().BoolVar(&showJSON, "json", false, "Include fonts, colors, and other settings")
	return command
}

func showHelp(cmd *cobra.Command) error {
	if err := cmd.Help(); err != nil {
		return ctxerrors.Wrap(err, "print help")
	}

	return nil
}

func (c cli) safeAreaCommand() *cobra.Command {
	var platform string
	command := &cobra.Command{
		Use:   "safe-area INPUT",
		Short: "Report styles and events whose margins reach under a platform's buttons and captions",
		Args:  exactArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return c.safeArea(platform, args[0]) },
	}
	command.Flags().StringVar(&platform, "platform", ass.SafeAreaAll, "Safe area: all, tiktok, shorts or reels")

	return command
}

func (c cli) safeAreasCommand() *cobra.Command {
	var showJSON bool
	command := &cobra.Command{
		Use: "safe-areas", Short: "List platform safe areas", Args: exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error { return c.safeAreas(showJSON) },
	}
	command.Flags().BoolVar(&showJSON, "json", false, "Include the edge ratios")

	return command
}

func (c cli) burnCommand() *cobra.Command {
	var subtitles, output string
	var crf int
	command := &cobra.Command{
		Use:   "burn INPUT --subtitles FILE --output FILE",
		Short: "Draw an ASS file into a video with FFmpeg, keeping its size, codec and other streams",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var quality *int
			if cmd.Flags().Changed("crf") {
				quality = &crf
			}

			return c.burn(cmd.Context(), args[0], subtitles, output, quality)
		},
	}
	command.Flags().StringVar(&subtitles, "subtitles", "", "ASS file to draw into the video (required)")
	command.Flags().StringVar(&output, "output", "", "Video file to write; its extension selects the container (required)")
	command.Flags().IntVar(&crf, "crf", 0, "Encoder quality instead of the visually lossless default; lower is better")

	return command
}
