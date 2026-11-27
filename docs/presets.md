---
title: Presets and fonts
description: Choose a complete look and install the fonts required by built-in presets.
---

# Presets and fonts

Each preset includes its font, text and secondary karaoke colors, background opacity, outline, shadow, and placement. Font size and vertical margin scale with the script height.

| Preset | Font | Text and background | Secondary karaoke color |
| --- | --- | --- | --- |
| `black-gold` | DejaVu Sans Bold | White on opaque black | Gold |
| `ivory-navy` | DejaVu Serif | Ivory on navy | Sky blue |
| `mint-plum` | DejaVu Sans Mono Bold | Mint on plum | Peach |
| `clean` | DejaVu Sans | White text, black outline, clear background | Sky blue |

Karaoke tags control the primary and secondary colors. Existing inline tags can override style fields.

## Install fonts

```bash
assctl fonts-install
assctl fonts-install --preset ivory-navy
```

Omit `--preset` to install every built-in preset's required font. The installer verifies the DejaVu 2.37 archive and each font with SHA-256 before writing, and preserves the font license. Installed, verified files work offline. Corrupted files are replaced on the next successful installation. Failed verification leaves installed fonts intact.

| Platform | User font directory |
| --- | --- |
| Linux and FreeBSD | `$XDG_DATA_HOME/fonts/assctl`, or `~/.local/share/fonts/assctl` |
| macOS | `~/Library/Fonts/assctl` |
| Windows | `%LOCALAPPDATA%/Microsoft/Windows/Fonts` |

Restart a renderer that was running during installation. On Linux, run `fc-cache -f` if Fontconfig still selects a fallback. Windows may require opening the installed files and choosing Install to register them. The installation and rendering checks run on Linux.

ASS files store family names. Install fonts on the machine that renders the video. For custom fonts, supply the font files yourself.

## Customize a preset

Run `assctl presets --json` and save one object as a custom theme. Edit its fields, then use `--theme-file`. The [CLI guide](cli.md#custom-themes) shows the commands and color format. Box presets create separate background and foreground events for new captions. Applying a theme to an existing file changes style definitions.

The fonts come from the [DejaVu project](https://dejavu-fonts.github.io/Download.html) under its font license. The MIT license covers assctl's own code.
