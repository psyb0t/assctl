---
title: Installation
description: Install assctl with Go or run the Docker image, set up preset fonts, and update the command.
---

# Installation

Install Go 1.26 or newer, then run:

```bash
go install github.com/psyb0t/assctl/cmd/assctl@latest
assctl version
assctl fonts-install
```

`assctl burn` also needs FFmpeg and ffprobe on your `PATH`. Every other command works without them.

Go puts the executable in `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. Add that directory to your shell's `PATH` if needed. The font command installs the fonts every built-in preset needs for your user.

With an existing ASS file, run:

```bash
assctl inspect captions.ass
assctl theme --preset ivory-navy --output styled.ass captions.ass
assctl validate styled.ass
```

To create a file from timed text, use the [caption example](cli.md#build-captions-from-timed-text).

## Docker

The [`psyb0t/assctl`](https://hub.docker.com/r/psyb0t/assctl) image holds assctl, FFmpeg with libass for `burn`, and every built-in preset's font, already verified and installed. It needs no Go, no font download and no network at run time. It is published for `linux/amd64` and `linux/arm64`.

Mount the working directory at `/work` and run as your own user, so the files it writes belong to you:

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" psyb0t/assctl \
  captions --preset tiktok-bold --spec timing.json --output captions.ass
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" psyb0t/assctl \
  safe-area captions.ass
```

Burn the captions into a video with [`burn`](cli.md#burn-captions-into-a-video), which uses the image's FFmpeg:

```bash
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" psyb0t/assctl \
  burn input.mp4 --subtitles captions.ass --output output.mp4
```

`--embed-preset-fonts` works offline in the image, because the fonts are already installed. Paths are relative to `/work`, so files outside the mounted directory are not visible.

| Tag | Image |
| --- | --- |
| `latest` | The newest build of the default branch |
| `vX.Y.Z` | The release, the same image as the `sha-` tag of its commit |
| `sha-<commit>` | The build of one commit |

`assctl version` in the image prints the `sha-<commit>` it was built from, including in a release tag, because a release reuses that build instead of rebuilding it.

## Update and uninstall

Run `docker pull psyb0t/assctl` to update the image. Repeat `go install` to update the Go install. Replace `@latest` with a published version or commit to select a particular revision. Remove the `assctl` executable from your Go binary directory to uninstall. Fonts remain in your user font directory until you remove them.

## Completion

Run `assctl completion bash --help` for shell setup instructions. Bash, Zsh, Fish, and PowerShell are supported.
