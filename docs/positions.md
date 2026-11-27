---
title: Text positions
description: Place captions with independent layouts, keep them inside TikTok, YouTube Shorts and Instagram Reels safe areas, and understand ASS margins.
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

| Name | ASS alignment | Vertical margin | Left margin | Right margin |
| --- | --- | --- | --- | --- |
| `portrait-upper` | 8, top center | 18% of height | 18% of width | 18% of width |
| `portrait-center` | 5, middle center | 0 | 120/1080 of width | 300/1080 of width |
| `portrait-lower-middle` | 2, bottom center | 40% of height | 120/1080 of width | 300/1080 of width |

Every named position stays inside the [combined safe area](#platform-safe-areas) of TikTok, YouTube Shorts and Instagram Reels on a vertical video of any size. Centered and lower text keeps a wider right margin because TikTok's like, comment and share buttons cover the lower right of the screen, so that text sits slightly left of center. Explicit selection also works on landscape and square canvases.

Margins use the script's `PlayResX` and `PlayResY` coordinate space. Ratios are multiplied by the corresponding dimension and rounded up to a whole pixel, so a margin never ends short of its ratio. At 1080 x 1920, `portrait-lower-middle` sets the left margin to 120, the right margin to 300 and the bottom margin to 768, placing the bottom anchor at y=1152. Bottom-anchored text grows upward; top-anchored text grows downward. Center alignment ignores the vertical margin in libass.

Foreground and background styles receive the same geometry when a caption preset creates a box. Margins constrain wrapping, not clipping. Long words, outlines, padding, and too many lines can extend beyond them.

For files without dimensions, layout follows libass: 384 x 288 when both are missing, normally a 4:3 ratio when one is missing, with the renderer's special 1280 x 1024 case. Like libass, assctl treats a dimension of zero or less as missing. assctl does not inspect a video to discover its dimensions.

## Existing overrides

Nonzero event margins and inline `\an`, `\pos`, `\move`, or `\r` tags retain their ASS precedence. A style change preserves them, so an explicitly positioned event might not move. Inspect the event and use `edit` when changing that authored behavior is intentional.

ASS owns layout. FFmpeg's ASS/subtitles filter supplies libass with video frames and fonts. Set script dimensions to the intended video dimensions for predictable scaling.

## Platform safe areas

Each platform publishes the part of a vertical video its buttons, captions and account name leave uncovered. On a 1080 x 1920 video, keep text out of these edges:

| Area | Top | Bottom | Left | Right | Source |
| --- | --- | --- | --- | --- | --- |
| `tiktok` | 240 | 660 | 120 | 120, and 300 below y=840 | The In-Feed Standard safe zone template on [TikTok's In-Feed ads page](https://ads.tiktok.com/help/article/tiktok-auction-in-feed-ads?lang=en), measured from the image |
| `shorts` | 288 | 672 | 48 | 192 | Google's [Universal Video Ad Safe Zones](https://services.google.com/fh/files/misc/universalsafezones-youtube.pdf), vertical video |
| `reels` | 14% | 35% | 6% | 6% | [Meta's Reels ad guide](https://www.facebook.com/business/ads-guide/update/image/instagram-reels) |
| `all` | 288 | 672 | 120 | 192, and 300 below y=840 | The largest edge of the three |

TikTok's safe area shrinks further when an ad has a long caption or an anchor link. Google and Meta publish these areas for ads, which run in the same feed with the same buttons as other videos.

Check a file against an area:

```bash
assctl safe-area captions.ass
assctl safe-area --platform tiktok captions.ass
assctl safe-areas --json
```

`safe-area` prints a JSON list of every style, and every event with its own margins, whose margins let text reach into an unsafe edge. Each entry has `Style`, `Event` (the one-based row number in `[Events]`, counting comment rows, or 0 for a style), `Edge` (`top`, `bottom`, `left` or `right`), `Margin` (the margin it has) and `Required` (the margin it needs), in script pixels. The command exits with an error when the list is not empty, and prints `[]` when it is.

The check works like this:

- Top-aligned styles (7 to 9) need the top margin and bottom-aligned styles (1 to 3) need the bottom margin. Middle-aligned styles (4 to 6) get no vertical check, because libass ignores their vertical margin.
- Every alignment needs the left and right margins. For `tiktok` and `all`, the wider right margin applies when the text's anchor line sits below 840/1920 of the height. The anchor line is the top margin for top-aligned text, the middle of the screen for middle-aligned text, and the height minus the bottom margin for bottom-aligned text.
- An event with a nonzero margin is checked with its own margins, taking the style's margin for each side it leaves at zero. Events with all margins at zero follow their style and are not listed again. Comment rows count like dialogue.
- It reads styles and margins only. Inline `\an`, `\pos`, `\move` and `\r` tags are not checked, and it cannot see how many lines a caption wraps to.

[BBC subtitle guidance](https://www.bbc.co.uk/accessibility/forproducts/guides/subtitles/) adds advice that no margin can check: keep captions off faces and important action, and keep them to one or two lines. Preview short phrases with the destination app's controls visible.

## Custom positions in Go

`ass.TextPosition` exposes `Alignment`, `MarginRatio`, `LeftMarginRatio`, and `RightMarginRatio`. Apply it with `Script.WithTextPosition` or `Document.WithTextPosition`. Ratios must be finite, at least zero, and less than one; horizontal ratios must sum to less than one. Middle alignment requires a zero vertical ratio because the renderer ignores that margin.

`ass.SafeAreas` and `ass.LookupSafeArea` return the areas above as `ass.SafeArea` values: `Name`, `TopRatio`, `BottomRatio`, `LeftRatio`, `RightRatio`, `LowerRightRatio` and `LowerFromRatio`, as fractions of the script size. `LowerRightRatio` replaces `RightRatio` when the text's anchor line is below `LowerFromRatio` of the height. `ass.SafeAreaAll` names the combined area. `LookupSafeArea` wraps `commerr.ErrNotFound` for an unknown name.

`Document.CheckSafeArea` returns the same issues as the `safe-area` command, as `[]ass.SafeAreaIssue` with the fields above. `ass.EdgeTop`, `ass.EdgeBottom`, `ass.EdgeLeft` and `ass.EdgeRight` name the edges.

See [Aegisub's style documentation](https://aegisub.org/docs/latest/styles/) for alignment and margin behavior.
