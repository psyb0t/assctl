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

// buildVersion is stamped by builds that have no module version, such as the
// Docker image: go build -ldflags "-X main.buildVersion=...".
var buildVersion string

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		// The process exits with status 1 either way, and stderr has nowhere else to report a failed write.
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output, diagnostics io.Writer) error {
	checkedOutput := &commandOutput{writer: output}
	command := cli{input: input, output: checkedOutput, diagnostics: diagnostics}
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
	info, _ := debug.ReadBuildInfo()
	if err := writeString(c.output, resolveVersion(buildVersion, info)+"\n"); err != nil {
		return ctxerrors.Wrap(err, "print version")
	}

	return nil
}

// resolveVersion prefers a stamped version, then the module version Go records
// for go install, then devVersion.
func resolveVersion(stamped string, info *debug.BuildInfo) string {
	if stamped != "" {
		return stamped
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != goDevelVersion {
		return info.Main.Version
	}

	return devVersion
}
