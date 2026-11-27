# assctl

[![Go Reference](https://pkg.go.dev/badge/github.com/psyb0t/assctl/pkg/ass.svg)](https://pkg.go.dev/github.com/psyb0t/assctl/pkg/ass)
[![CI](https://github.com/psyb0t/assctl/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/assctl/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/assctl/badges/coverage.svg)](https://github.com/psyb0t/assctl/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/assctl/badges/version.svg)](https://github.com/psyb0t/assctl/releases)
[![license](https://raw.githubusercontent.com/psyb0t/assctl/badges/license.svg)](LICENSE)
[![imported by](https://raw.githubusercontent.com/psyb0t/assctl/badges/importers.svg)](https://github.com/psyb0t/assctl/blob/badges/importers.md)

Your subtitles already have timing. Now make the damn things look good. assctl creates, inspects, edits, validates, and themes Advanced SubStation Alpha (ASS) files, from a terminal or your own Go code.

## Contents

- [Install](#install)
- [Quick start](#quick-start)
- [Presets](#presets)
- [Go package](#go-package)
- [Agent integrations](#agent-integrations)
- [Documentation](#documentation)
- [Development](#development)
- [Sponsor](#sponsor)
- [License](#license)

## Install

Go 1.26 or newer.

```bash
go install github.com/psyb0t/assctl/cmd/assctl@latest
assctl fonts-install
```

## Quick start

```bash
assctl inspect captions.ass
assctl theme --preset ivory-navy --output styled.ass captions.ass
assctl position --text-position portrait-lower-middle --output moved.ass styled.ass
assctl validate styled.ass
```

`theme` updates the style definitions and preserves the original dialogue, event timing, inline tags, and unrelated sections. Use `--style NAME` to theme one style. Use `-` to read or write through a pipe.

`create` builds a script from typed JSON. `captions` turns already timed text into styled captions. `edit` changes selected script settings, style columns, and individual events. `tags` generates override blocks for karaoke, drawing, fades, clipping, and animation. See the [CLI guide](https://psyb0t.github.io/assctl/cli/) for examples and the JSON shape.

## Presets

Pick a whole look, including the font, colors, background, outline, shadow, and placement. Tweak any of it when the defaults aren't your thing.

| Preset | Text and background | Secondary karaoke color |
| --- | --- | --- |
| `black-gold` | Bold white text, opaque black box | Gold |
| `ivory-navy` | Ivory serif text, navy box | Sky blue |
| `mint-plum` | Bold mint text, plum box | Peach |
| `clean` | White text, black outline, clear background | Sky blue |
| `tiktok-bold` | TikTok Sans Bold, white with a black outline, lower-middle | Yellow |
| `reels-bold` | Barlow Condensed ExtraBold, white with a black outline, lower-middle | Yellow |
| `shorts-impact` | Anton, yellow with a black outline, lower-middle | White |
| `social-center` | Anton, white with a black outline, centered | Yellow |
| `social-box` | TikTok Sans Bold, white on translucent black, lower-middle | Yellow |
| `social-hook` | Barlow Condensed ExtraBold, yellow with a black outline, upper-center | White |
| `the-bold` | THE BOLD FONT free version, white with a black outline, lower-middle | Yellow |

Run `assctl presets --json` to see the full values. Use one of those objects as a starting point for `--theme-file` with `captions` or `theme`. Inline overrides in an existing subtitle still take effect.

Social presets keep text inside the safe area TikTok, YouTube Shorts and Instagram Reels publish for their buttons and captions, on a vertical video of any size. `assctl safe-area FILE` checks any subtitle file against one platform or all three. Fonts download from their upstream sources through `fonts-install`, or straight into the subtitle file with `--embed-preset-fonts`, so the machine that renders the video needs no installed fonts. See [font licenses, source downloads, and positioning](https://psyb0t.github.io/assctl/presets/).

Use `--text-position` with `captions`, `create`, or `theme` to choose placement independently of appearance. `assctl positions` lists the layouts. Existing event positioning still takes precedence. See [text positions](https://psyb0t.github.io/assctl/positions/).

## Go package

```bash
go get github.com/psyb0t/assctl/pkg/ass
```

Use `ass.Parse` and `Document.Apply` for edits, or `ass.NewScript` and `Script.Encode` to create a script. See the [Go guide](https://psyb0t.github.io/assctl/go/).

## Agent integrations

The skill documents how to use the installed command on local subtitle files.

```bash
claude plugin marketplace add psyb0t/agents
claude plugin install assctl@psyb0t

codex plugin marketplace add psyb0t/agents
codex plugin add assctl@psyb0t

openclaw skills install @psyb0t/assctl
```

## Documentation

The [assctl documentation](https://psyb0t.github.io/assctl/) covers the CLI and Go library.

| Guide | What you need |
| --- | --- |
| [Installation](https://psyb0t.github.io/assctl/install/) | Install, update, and prepare fonts |
| [CLI](https://psyb0t.github.io/assctl/cli/) | Commands, flags, and JSON examples |
| [Presets and fonts](https://psyb0t.github.io/assctl/presets/) | Palettes, backgrounds, font installation, and embedded fonts |
| [Text positions](https://psyb0t.github.io/assctl/positions/) | Independent placement, platform safe areas, margins, and override precedence |
| [ASS features](https://psyb0t.github.io/assctl/ass-features/) | Tags, drawing, karaoke, effects, embedded fonts, libass extensions, headers, and wrapping modes |
| [Go library](https://psyb0t.github.io/assctl/go/) | Create and edit scripts in Go |
| [Agents](https://psyb0t.github.io/assctl/agents/) | Skills and plugin setup |
| [Troubleshooting](https://psyb0t.github.io/assctl/troubleshooting/) | Input, font, and output errors |
| [Development](https://psyb0t.github.io/assctl/development/) | Build, test, docs, and releases |

## Development

```bash
make check
make test-coverage
make sec
make docs
```

The Makefile runs Go checks in the same development image locally and in CI. Branch pushes check the source. Version tags wait for that commit's default-branch checks before the release and ClawHub publish. See [development and releases](https://psyb0t.github.io/assctl/development/).

## Sponsor

[Sponsor on GitHub](https://github.com/sponsors/psyb0t)

<a href="https://www.buymeacoffee.com/psyb0t"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-blue.png" alt="Buy Me a Coffee" height="60" width="217"></a>

## License

[MIT](LICENSE).
