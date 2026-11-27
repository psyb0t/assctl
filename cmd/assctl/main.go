package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/psyb0t/ctxerrors"
)

const (
	devVersion     = "dev"
	goDevelVersion = "(devel)"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		// The process exits with status 1 either way, and stderr has nowhere else to report a failed write.
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, diagnostics io.Writer) error {
	checkedOutput := &commandOutput{writer: output}
	command := cli{input: input, output: checkedOutput}
	root := command.rootCommand()
	root.SetIn(input)
	root.SetOut(checkedOutput)
	root.SetErr(diagnostics)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return ctxerrors.Wrap(err, "execute assctl")
	}
	if checkedOutput.err != nil {
		return ctxerrors.Wrap(checkedOutput.err, "write command output")
	}

	return nil
}

func (c cli) version() error {
	version := devVersion
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != goDevelVersion {
		version = info.Main.Version
	}
	if err := writeString(c.output, version+"\n"); err != nil {
		return ctxerrors.Wrap(err, "print version")
	}

	return nil
}
