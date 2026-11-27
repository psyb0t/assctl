---
title: CLI guide
description: Commands, flags, JSON formats, file editing, and shell completion.
---

# CLI guide

| Command | What it reads and writes |
| --- | --- |
| `create --spec FILE --output FILE` | Create a script from a JSON `ass.Script`. |
| `captions --preset NAME --spec FILE --output FILE` | Create caption events from a JSON `ass.CaptionSpec`. |
| `inspect INPUT` | Print the script settings and the style and event columns as JSON. |
| `validate INPUT` | Check the file structure, style references, and event timing. |
| `edit --spec FILE --output FILE INPUT` | Apply a JSON `ass.Edit` to an existing script. |
| `theme --preset NAME --output FILE INPUT` | Apply a named look. Add `--style NAME` to restrict it to one style. |
| `presets --json` | Print the full values for all presets. |
| `positions --json` | List independent caption layouts and their ratios. |
| `position INPUT --text-position NAME --output FILE` | Replace style alignment and margins. Add `--style NAME` to select one style. |
| `tags --spec FILE` | Print a typed ASS override block from JSON. |
| `version` or `--version` | Print the installed version. |
| `completion SHELL` | Generate completion for Bash, Zsh, Fish, or PowerShell. |
| `fonts-install --preset NAME` | Install one built-in preset's required fonts. Omit the flag for all presets. |

Flags can appear before or after the positional input. Run `assctl COMMAND --help` for command-specific options. `--output` defaults to stdout and accepts `-`. `create` and `captions` default to stdin for their JSON specs. An explicit output file may replace an existing file. Omit the output path or write to a new file to keep the original.

The command tree and help use Cobra. Run `assctl completion SHELL --help` for installation instructions for your shell.

JSON specs must contain exactly one object. Unknown fields, duplicate field names ignoring case, trailing values, and nesting beyond 64 levels are rejected. Typed strings must produce valid UTF-8 ASS output.

`create`, `captions`, and `theme` accept `--text-position NAME`. See [text positions](positions.md) for geometry and override precedence, and [ASS features](ass-features.md) for typed tags, script headers, comments, and wrapping modes.

## Build captions from timed text

```json
{
  "Title": "Intro",
  "Width": 1080,
  "Height": 1920,
  "WordsPerLine": 4,
  "Dialogues": [
    {
      "Start": 0,
      "End": 2200000000,
      "Text": "Write the first line here."
    }
  ]
}
```

Durations in the typed JSON files use nanoseconds. Here the line lasts 2.2 seconds. The command reads timing you supply. Each box preset produces separate background and foreground events for new captions.

Lines containing ASS tags or escapes keep their original text when uppercase or word wrapping is enabled.

```bash
assctl captions --preset black-gold --spec timing.json --output captions.ass
```

## Change existing dialogue and styles

```json
{
  "Info": {
    "Title": "New intro"
  },
  "Styles": [
    {
      "Name": "Text",
      "Fields": {
        "Fontsize": "54",
        "PrimaryColour": "&H00FFFFFF"
      }
    }
  ],
  "Events": [
    {
      "Index": 2,
      "Fields": {
        "Start": "0:00:00.50"
      },
      "Override": {
        "Position": {
          "X": 540,
          "Y": 1550
        }
      }
    }
  ]
}
```

```bash
assctl edit --spec edits.json --output updated.ass captions.ass
```

Event indices count dialogue and comment rows in file order, starting at one. Column names follow the file's `Format:` line. Event field updates use string values, including ASS timestamps. A typed `Override` prepends tags and keeps existing text and tags. If a later inline tag overrides the same property, it still takes effect.

## Build the full script

Give `create` a JSON `ass.Script` with `Title`, `Resolution`, `Styles`, and `Events`. An event can include a typed `Override` with movement, fades, transforms, clipping, or other fields from the Go package. Use `Text` for literal ASS text and inline karaoke tags. For a hard line break in the subtitle, use `\\N` in JSON.

```json
{
  "Title": "Timed dialogue",
  "Resolution": {
    "Width": 1920,
    "Height": 1080
  },
  "Styles": [
    {
      "Name": "Default",
      "FontName": "DejaVu Sans",
      "FontSize": 46,
      "Bold": true,
      "ScaleX": 100,
      "ScaleY": 100,
      "BorderStyle": 1,
      "Outline": 3,
      "Alignment": 2,
      "MarginV": 50,
      "Encoding": 1
    }
  ],
  "Events": [
    {
      "Start": 0,
      "End": 2000000000,
      "Style": "Default",
      "Text": "{\\k60}Hello, {\\k100}world",
      "Override": {
        "Position": {
          "X": 960,
          "Y": 920
        }
      }
    }
  ]
}
```

## Custom themes

Use `assctl presets --json` and save one complete object as `theme.json`. Edit the colors, font, geometry, and opacity values. Both `captions` and `theme` accept the file.

```bash
assctl captions --theme-file theme.json --spec timing.json --output captions.ass
assctl theme --theme-file theme.json --output themed.ass captions.ass
```

Colors in style and preset JSON use ASS hexadecimal order, with blue before red. Opacity values run from `&H00&` for opaque to `&HFF&` for transparent. The renderer needs the chosen font installed or attached to the media. See the [libass format guide](https://github.com/libass/libass/wiki/ASS-File-Format-Guide).
