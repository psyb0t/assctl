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
| `safe-area INPUT --platform NAME` | Print a JSON list of styles and events whose margins reach under TikTok, Shorts or Reels buttons and captions, and fail when there are any. `NAME` is `all` (default), `tiktok`, `shorts` or `reels`. See [platform safe areas](positions.md#platform-safe-areas) for the fields. |
| `safe-areas --json` | List the `tiktok`, `shorts` and `reels` safe areas and the combined `all` area. `--json` adds the edge ratios. |
| `burn INPUT --subtitles FILE --output FILE` | Draw an ASS file into a video with FFmpeg, keeping its size, codec and other streams. See [burn captions into a video](#burn-captions-into-a-video). |
| `version` or `--version` | Print the installed version. |
| `completion SHELL` | Generate completion for Bash, Zsh, Fish, or PowerShell. |
| `fonts-install --preset NAME` | Install one built-in preset's required fonts. Omit the flag for all presets. |

Flags can appear before or after the positional input. Run `assctl COMMAND --help` for command-specific options. `--output` defaults to stdout and accepts `-`. `create`, `captions` and `tags` default to stdin for their JSON specs. `captions` and `theme` use `black-gold` when neither `--preset` nor `--theme-file` is given, and the two flags cannot be combined. An explicit output file may replace an existing file. Omit the output path or write to a new file to keep the original.

The command tree and help use Cobra. Run `assctl completion SHELL --help` for installation instructions for your shell.

JSON specs must contain exactly one object. Unknown fields, duplicate field names ignoring case, trailing values, and nesting beyond 64 levels are rejected. Typed strings must produce valid UTF-8 ASS output.

`create`, `captions`, and `theme` accept `--text-position NAME` and a repeatable `--embed-font FILE`, which stores the font in the output file. `captions` and `theme` also accept `--embed-preset-fonts`, which installs the preset's verified fonts and embeds them. See [embedded fonts](presets.md#embed-fonts). See [text positions](positions.md) for geometry and override precedence, and [ASS features](ass-features.md) for typed tags, script headers, comments, and wrapping modes.

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

Lines containing a backslash (ASS tags, `\N` or other escapes) are written as given: no uppercase, wrapping or escaping. In other lines, `captions` escapes `{` so the text after it shows, as described under [literal braces](ass-features.md#literal-braces).

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

## Burn captions into a video

`burn` draws a subtitle file into a video with the FFmpeg and ffprobe on your `PATH`, which the [Docker image](install.md#docker) includes:

```bash
assctl burn input.mp4 --subtitles captions.ass --output output.mp4
```

Drawing on the picture means the video stream has to be encoded again; it cannot be copied. `burn` encodes it with the same codec, resolution, frame rate, pixel format and color tags as the source, at a quality meant to look the same. Everything else is copied without change: audio, other video and subtitle tracks, cover art, chapters and metadata.

| Source codec | Encoder | Default `--crf` | `--crf` range |
| --- | --- | --- | --- |
| H.264 | libx264 | 18 | 0 to 51 |
| HEVC | libx265 | 20 | 0 to 51 |
| VP9 | libvpx-vp9 | 20 | 0 to 63 |
| AV1 | libsvtav1 | 23 | 1 to 63 |

A lower `--crf` gives higher quality and a larger file. Other video codecs are rejected. An HEVC stream tagged `hvc1`, the tag Apple players need, keeps that tag.

- The output's extension selects its container. Use the input's container, because another container may not accept every copied stream.
- A phone video stored sideways with a rotation flag is turned upright before drawing, so the captions appear the right way up. The output stores the upright picture without the flag and shows the same image.
- libass scales the script's `PlayResX` and `PlayResY` to the video. A script made for a different shape of video stretches to fit; `burn` does not check it.
- HDR mastering metadata is not carried into the new encode.
- `burn` writes a temporary file beside the output and renames it when FFmpeg succeeds, so a failed burn leaves no partial file. The output must differ from the input. FFmpeg's progress goes to stderr.

## Custom themes

Use `assctl presets --json` and save one complete object as `theme.json`. Edit the colors, font, geometry, and opacity values. Both `captions` and `theme` accept the file.

```bash
assctl captions --theme-file theme.json --spec timing.json --output captions.ass
assctl theme --theme-file theme.json --output themed.ass captions.ass
```

Colors in style and preset JSON use ASS hexadecimal order, with blue before red. Opacity values run from `&H00&` for opaque to `&HFF&` for transparent. The renderer needs the chosen font installed or attached to the media. See the [libass format guide](https://github.com/libass/libass/wiki/ASS-File-Format-Guide).
