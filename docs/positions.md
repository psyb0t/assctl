---
title: Text positions
description: Place captions with independent layouts and understand ASS margins and override precedence.
---

# Text positions

Choose appearance and placement separately. Any built-in appearance preset can use any named position.

```bash
assctl positions
assctl positions --json
assctl captions --preset the-bold --text-position portrait-upper \
  --spec timing.json --output captions.ass
assctl theme --preset tiktok-bold --text-position portrait-center \
  --output themed.ass captions.ass
assctl position --text-position portrait-lower-middle \
  --output moved.ass themed.ass
```

`create` also accepts `--text-position`. `position` changes alignment and margins without changing fonts, colors, timing, or text. Add `--style NAME` to `position` or `theme` to restrict the change to that style. Omit the position flag to keep the preset's own layout. Selecting a position replaces all four geometry fields, including zero values.

## Named layouts

| Name | ASS alignment | Vertical margin | Left and right margins |
| --- | --- | --- | --- |
| `portrait-upper` | 8, top center | 18% of height | 18% of width each |
| `portrait-center` | 5, middle center | 0 | 18% of width each |
| `portrait-lower-middle` | 2, bottom center | 40% of height | 18% of width each |

The portrait prefix describes intended use. Explicit selection also works on landscape and square canvases. These ratios are assctl design choices, not platform specifications or a popularity ranking.

Margins use the script's `PlayResX` and `PlayResY` coordinate space. Ratios are multiplied by the corresponding dimension and rounded down. At 1080 x 1920, `portrait-lower-middle` sets horizontal margins to 194 and the bottom margin to 768, placing the bottom-center anchor at y=1152. Bottom-anchored text grows upward; top-anchored text grows downward. Center alignment ignores the vertical margin in libass.

Foreground and background styles receive the same geometry when a caption preset creates a box. Margins constrain wrapping, not clipping. Long words, outlines, padding, and too many lines can extend beyond them.

For files without dimensions, layout follows libass: 384 x 288 when both are missing, normally a 4:3 ratio when one is missing, with the renderer's special 1280 x 1024 case. Explicit dimensions must be positive 32-bit integers. assctl does not inspect a video to discover its dimensions.

## Existing overrides

Nonzero event margins and inline `\an`, `\pos`, `\move`, or `\r` tags retain their ASS precedence. A style change preserves them, so an explicitly positioned event might not move. Inspect the event and use `edit` when changing that authored behavior is intentional.

ASS owns layout. FFmpeg's ASS/subtitles filter supplies libass with video frames and fonts. Set script dimensions to the intended video dimensions for predictable scaling.

## Choosing a layout

[BBC subtitle guidance](https://www.bbc.co.uk/accessibility/forproducts/guides/subtitles/) discusses portrait captions, avoiding faces and important action, and limiting line counts. It addresses accessibility captions rather than prescribing a universal social template. [TikTok's in-feed guidance](https://ads.tiktok.com/help/article/tiktok-auction-in-feed-ads?lang=en) says safe areas vary with dimensions, captions, and additional formats. [YouTube's Shorts guides](https://support.google.com/youtube/answer/16215842?hl=en) show areas where interface controls can cover overlays.

Preview short phrases with the destination app's controls visible. A named position cannot detect faces or guarantee an unobstructed result.

## Custom positions in Go

`ass.TextPosition` exposes `Alignment`, `MarginRatio`, `LeftMarginRatio`, and `RightMarginRatio`. Apply it with `Script.WithTextPosition` or `Document.WithTextPosition`. Ratios must be finite, at least zero, and less than one; horizontal ratios must sum to less than one. Middle alignment requires a zero vertical ratio because the renderer ignores that margin.

See [Aegisub's style documentation](https://aegisub.org/docs/latest/styles/) for alignment and margin behavior.
