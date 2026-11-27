package main

import (
	"github.com/spf13/cobra"

	"github.com/psyb0t/ctxerrors"

	"github.com/psyb0t/assctl/internal/pkg/fonts"
	"github.com/psyb0t/assctl/pkg/ass"
)

func exactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(count)(cmd, args); err != nil {
			return ctxerrors.Wrap(errUsage, err.Error())
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
			return cmd.Help()
		},
	}
	root.Flags().BoolVar(&showVersion, "version", false, "Print the installed version")
	root.AddCommand(
		c.createCommand(false), c.createCommand(true),
		c.inspectCommand(false), c.inspectCommand(true),
		c.editCommand(), c.themeCommand(), c.presetsCommand(),
		c.fontsCommand(),
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
			command.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
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
			return writeString(c.output, directory+"\n")
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
	var spec, output, preset, theme string
	command := &cobra.Command{
		Use: name, Short: description, Args: exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			return c.create(spec, output, captions, preset, theme)
		},
	}
	command.Flags().StringVar(&spec, "spec", stream, "JSON file or - for stdin")
	command.Flags().StringVar(&output, "output", stream, "ASS file or - for stdout")
	if captions {
		command.Flags().StringVar(&preset, "preset", "", "Built-in preset (default "+presetDefault+")")
		command.Flags().StringVar(&theme, "theme-file", "", "JSON file with custom preset values")
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
	var preset, custom, style, output string
	command := &cobra.Command{
		Use: "theme INPUT", Short: "Apply a preset or custom theme to existing styles", Args: exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return c.theme(preset, custom, style, output, args[0])
		},
	}
	command.Flags().StringVar(&preset, "preset", "", "Built-in preset (default "+presetDefault+")")
	command.Flags().StringVar(&custom, "theme-file", "", "JSON file with custom preset values")
	command.Flags().StringVar(&style, "style", "", "Theme one named style, or omit to theme every style")
	command.Flags().StringVar(&output, "output", stream, "ASS file or - for stdout")
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
