package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/psyb0t/ctxerrors"
)

const usage = "Usage: assctl <create|captions|inspect|validate|edit|theme|presets|version> [options]\n"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, diagnostics io.Writer) error {
	if len(args) == 0 {
		return writeString(output, usage)
	}
	name, args := args[0], args[1:]
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	command := cli{input: input, output: output}

	var err error
	switch name {
	case "help", "--help", "-h":
		err = writeString(output, usage)
	case "create":
		err = command.create(flags, args, false)
	case "captions":
		err = command.create(flags, args, true)
	case "inspect", "validate":
		err = command.inspect(flags, args, name == "validate")
	case "edit":
		err = command.edit(flags, args)
	case "theme":
		err = command.theme(flags, args)
	case "presets":
		err = command.presets(flags, args)
	case "version", "--version":
		err = command.version(flags, args)
	default:
		err = ctxerrors.Wrapf(errUsage, "unknown command %q", name)
	}
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return ctxerrors.Wrapf(err, "assctl %s", name)
	}

	return nil
}

func (c cli) version(flags *flag.FlagSet, args []string) error {
	if err := parseArgs(flags, args, 0); err != nil {
		return ctxerrors.Wrap(err, "version")
	}
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return writeString(c.output, info.Main.Version+"\n")
	}
	return writeString(c.output, "dev\n")
}
