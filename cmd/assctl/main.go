package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/psyb0t/ctxerrors"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, diagnostics io.Writer) error {
	command := cli{input: input, output: output}
	root := command.rootCommand()
	root.SetIn(input)
	root.SetOut(output)
	root.SetErr(diagnostics)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return ctxerrors.Wrap(err, "execute assctl")
	}
	return nil
}

func (c cli) version() error {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return writeString(c.output, info.Main.Version+"\n")
	}
	return writeString(c.output, "dev\n")
}
