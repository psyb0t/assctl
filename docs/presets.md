---
title: Presets and fonts
description: Choose a complete look and install the fonts required by built-in presets.
---

# Presets and fonts

Each preset includes its font, text and secondary karaoke colors, background opacity, outline, shadow, and placement. Font size and vertical margin scale with the script height. Social presets also scale their horizontal margins with the script width.

| Preset | Font | Text and background | Secondary karaoke color |
| --- | --- | --- | --- |
| `black-gold` | DejaVu Sans Bold | White on opaque black | Gold |
| `ivory-navy` | DejaVu Serif | Ivory on navy | Sky blue |
| `mint-plum` | DejaVu Sans Mono Bold | Mint on plum | Peach |
| `clean` | DejaVu Sans | White text, black outline, clear background | Sky blue |
| `tiktok-bold` | TikTok Sans Bold | White text, black outline | Yellow |
| `reels-bold` | Barlow Condensed ExtraBold | White text, black outline | Yellow |
| `shorts-impact` | Anton | Yellow text, black outline | White |
| `social-center` | Anton | White text, black outline | Yellow |
| `social-box` | TikTok Sans Bold | White on translucent black | Yellow |
| `social-hook` | Barlow Condensed ExtraBold | Yellow text, black outline | White |
| `the-bold` | THE BOLD FONT free version | White text, black outline | Yellow |

Karaoke tags control the primary and secondary colors. Existing inline tags can override style fields.

## Install fonts

```bash
assctl fonts-install
assctl fonts-install --preset ivory-navy
assctl fonts-install --preset tiktok-bold
assctl fonts-install --preset the-bold
```

Omit `--preset` to install every built-in preset's required font. Downloads come directly from upstream. The installer verifies pinned SHA-256 checksums, installs ordinary TTF files, and keeps each source's license or accompanying notice beside them. Installed, verified files work offline. Corrupted files are replaced on the next successful installation. Verification errors stop that font download before it is written.

| Font | Source and download | License |
| --- | --- | --- |
| TikTok Sans Bold | [TikTok's release](https://github.com/tiktok/TikTokSans/releases/tag/v4.000) | [SIL OFL 1.1](https://github.com/tiktok/TikTokSans/blob/main/OFL.txt), permits commercial use |
| Anton | [Google Fonts](https://github.com/google/fonts/tree/main/ofl/anton) | [SIL OFL 1.1](https://github.com/google/fonts/blob/main/ofl/anton/OFL.txt), permits commercial use |
| Barlow Condensed ExtraBold | [Google Fonts](https://github.com/google/fonts/tree/main/ofl/barlowcondensed) | [SIL OFL 1.1](https://github.com/google/fonts/blob/main/ofl/barlowcondensed/OFL.txt), permits commercial use |
| THE BOLD FONT free version | [Author's DaFont download page](https://www.dafont.com/the-bold-font.font), also linked by [the foundry](https://the-bold-font.com/) | Listed as 100% Free on DaFont; retains the author's accompanying RTF notice, not an OFL license |
| DejaVu | [DejaVu 2.37](https://dejavu-fonts.github.io/Download.html) | DejaVu font license, installed as `LICENSE` |

THE BOLD FONT's free version has fewer glyphs and no lowercase designs. Use uppercase text for that look. Its full version is a separate product sold by the author. assctl includes source URLs and checksums, not font binaries.

| Platform | User font directory |
| --- | --- |
| Linux and FreeBSD | `$XDG_DATA_HOME/fonts/assctl`, or `~/.local/share/fonts/assctl` |
| macOS | `~/Library/Fonts/assctl` |
| Windows | `%LOCALAPPDATA%/Microsoft/Windows/Fonts` |

Restart a renderer that was running during installation. On Linux, run `fc-cache -f` if Fontconfig still selects a fallback. On Windows, `fonts-install` also registers each font for the current user, so applications find it without opening the file. The installation and rendering checks run on Linux.

ASS files store family names. Install fonts on the machine that renders the video. For custom fonts, supply the font files yourself.

## Customize a preset

Run `assctl presets --json` and save one object as a custom theme. Edit its fields, then use `--theme-file`. The [CLI guide](cli.md#custom-themes) shows the commands and color format. Box presets create separate background and foreground events for new captions. Applying a theme to an existing file changes style definitions.

## Caption placement

Presets include a default layout. Choose `--text-position` to replace it independently of appearance. The [text position guide](positions.md) owns the layout ratios, canvas rules, and event override behavior. These are practical starting layouts, not platform-certified safe areas. Preview them with the destination app's controls visible.

The MIT license covers assctl's own code. Downloaded fonts retain their own terms.
