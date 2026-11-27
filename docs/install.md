---
title: Installation
description: Install assctl, set up preset fonts, and update the command.
---

# Installation

Install Go 1.26 or newer, then run:

```bash
go install github.com/psyb0t/assctl/cmd/assctl@latest
assctl version
assctl fonts-install
```

Go puts the executable in `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. Add that directory to your shell's `PATH` if needed. The font command installs the built-in presets' required DejaVu fonts for your user.

With an existing ASS file, run:

```bash
assctl inspect captions.ass
assctl theme --preset ivory-navy --output styled.ass captions.ass
assctl validate styled.ass
```

To create a file from timed text, use the [caption example](cli.md#build-captions-from-timed-text).

## Update and uninstall

Repeat `go install` to update. Replace `@latest` with a published version or commit to select a particular revision. Remove the `assctl` executable from your Go binary directory to uninstall. Fonts remain in your user font directory until you remove them.

## Completion

Run `assctl completion bash --help` for shell setup instructions. Bash, Zsh, Fish, and PowerShell are supported.
