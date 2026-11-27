---
title: Presets and fonts
description: Choose a complete look, install or embed the fonts built-in presets need, and keep social captions inside platform safe areas.
---

# Presets and fonts

Each preset includes its font, text and secondary karaoke colors, background opacity, outline, shadow, and placement. Font size and vertical margin scale with the script height. Social presets also scale their horizontal margins with the script width and stay inside the [combined safe area](positions.md#platform-safe-areas) of TikTok, YouTube Shorts and Instagram Reels on a vertical video of any size. Lower and centered social looks keep a wider right margin, clear of TikTok's buttons. `assctl safe-area` checks a finished file.

A custom theme file can set `SideMarginRatio` for both sides and `RightMarginRatio` to give the right side its own margin.

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

## Embed fonts

So that the machine that renders the video needs no installed fonts, store them in the subtitle file's `[Fonts]` section:

```bash
assctl captions --preset tiktok-bold --embed-preset-fonts --spec timing.json --output captions.ass
assctl theme --preset the-bold --embed-preset-fonts --output themed.ass captions.ass
assctl create --embed-font MyFont.ttf --spec script.json --output script.ass
```

`--embed-preset-fonts` on `captions` or `theme` first installs the preset's verified fonts on your machine, like `fonts-install`, downloading only the missing ones, then embeds them. It works only with the fonts of the built-in presets; a `--theme-file` with any other font fails, so embed that font with `--embed-font`. `--embed-font FILE` on `create`, `captions` or `theme` embeds any font file and can repeat. A file embedded twice under the same name is rejected.

FFmpeg's `subtitles` and `ass` filters load embedded fonts. Embedding copies the font into every file you share, so check that its license allows that. The SIL OFL fonts and DejaVu allow it. Read THE BOLD FONT's accompanying notice before sharing files that embed it.

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

ASS files store family names. Install the fonts on the machine that renders the video, or [embed them](#embed-fonts). For custom fonts, supply the font files yourself.

## Customize a preset

Run `assctl presets --json` and save one object as a custom theme. Edit its fields, then use `--theme-file`. The [CLI guide](cli.md#custom-themes) shows the commands and color format. Box presets create separate background and foreground events for new captions. Applying a theme to an existing file changes style definitions.

## Caption placement

Presets include a default layout. Choose `--text-position` to replace it independently of appearance. The [text position guide](positions.md) owns the layout ratios, canvas rules, and event override behavior. The social layouts follow the platforms' published safe areas. Preview captions with the destination app's controls visible, because a long caption or an extra overlay can still cover them.

The MIT license covers assctl's own code. Downloaded fonts retain their own terms.
