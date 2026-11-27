---
title: ASS features
description: Typed tags, drawing, karaoke, literal braces, scroll and banner effects, embedded fonts, libass extensions, script headers, wrapping modes, and preservation boundaries.
---

# ASS features

assctl reads and writes UTF-8 ASS v4+ documents. Editing preserves unrelated lines, comments, attachments, unknown sections, and literal override tags. Event `Text` can contain literal ASS tags and drawing commands, including syntax without a typed helper.

## Typed overrides

| Feature | Fields on `ass.Override` |
| --- | --- |
| Typography | `FontName`, `FontSize`, `Bold`, `BoldWeight`, `Italic`, `Underline`, `StrikeOut`, `Spacing`, `Encoding` |
| Color and transparency | Primary, secondary, outline, and back colors; global and per-channel alpha |
| Borders and shadows | `Border`, `BorderX`, `BorderY`, `Shadow`, `ShadowX`, `ShadowY`, `Blur`, `BlurEdges` |
| Geometry | Scale, rotation, shear, `Alignment`, `Position`, `Movement`, `Origin` |
| Clipping | `Clip` for rectangles or `VectorClip` for ASS drawings, optionally inverse |
| Animation | `Fade`, `SimpleFade`, `Transform` |
| Karaoke | `Karaoke` with `k`, `kf`, or `ko`; `KaraokeStart` for `\kt` |
| Drawing | `DrawingMode` for `\p`, `DrawingBaseline` for `\pbo` |
| Style and wrapping | `ResetStyle`, `ResetScale` for `\fsc`, `WrapStyle` |

`assctl tags --spec override.json` prints an override block without building a full script:

```json
{
  "Spacing": 2,
  "BorderX": 3,
  "BorderY": 4,
  "SimpleFade": {
    "In": 150000000,
    "Out": 200000000
  },
  "Karaoke": {
    "Mode": "kf",
    "Duration": 350000000
  }
}
```

The same object works as an event's `Override` in `create` or `edit`. Typed durations use nanoseconds in JSON. Output tags use milliseconds for movement, fades, and transforms, and centiseconds for karaoke. One typed karaoke object describes one syllable segment. Put successive inline blocks in event text for a complete karaoke line.

`DrawingMode: 1` starts drawing mode. Put the drawing in `Text`, for example `m 0 0 l 100 0 100 100 0 100`, and use a literal `\p0` block before returning to text. A vector clip accepts the drawing in `VectorClip.Drawing`, with optional `Scale` and `Inverse` fields.

Typed generation rejects conflicting position/movement, rectangular/vector clipping, and simple/full fades. It checks finite coordinates, timing order, vector-clip drawing syntax, and whether transform fields are animatable. A `Transform` with `End` 0 runs from `Start` until the event ends. Literal tags in event text remain authored input and can override earlier generated tags.

## Literal braces

An opening brace in event text starts an override block, so `see {note}` hides `{note}`. `ass.EscapeText` writes the brace as `\{{}`, which libass shows as `{`. VSFilter shows a backslash there instead, because it has no brace escape. `captions` escapes braces in any dialogue text that has no backslash; text with a backslash is treated as authored ASS and kept as written.

## Scroll and banner effects

An event's `Effect` field moves the whole event. Build it with `ass.BannerEffect` or `ass.ScrollEffect`, which write the parameters renderers require. `ass.ScrollDirection` is `ass.ScrollUp` or `ass.ScrollDown`:

| Effect | Go | Result |
| --- | --- | --- |
| `Banner;delay;direction` | `BannerEffect(delay, leftToRight, fadeWidth)` | One line moves across the screen, one script pixel every `delay` milliseconds |
| `Scroll up;top;bottom;delay` | `ScrollEffect(ass.ScrollUp, top, bottom, delay, fadeHeight)` | Text moves up between script rows `top` and `bottom` |
| `Scroll down;top;bottom;delay` | `ScrollEffect(ass.ScrollDown, top, bottom, delay, fadeHeight)` | Text moves down between the same rows |

libass ignores the fade width and height; other renderers may use them. A bare `Banner` or `Scroll up` without numbers does nothing in libass.

## Embedded fonts

`Script.Fonts` stores font files in the script's `[Fonts]` section, and `Document.EmbedFonts` adds them to an existing file without touching its other lines. Each `ass.EmbeddedFont` has a `Name`, the file name such as `DejaVuSans.ttf`, and the file's bytes in `Data`, which JSON specs give as base64. Names must be one line without surrounding spaces, and a name already in the file, ignoring case, is rejected, as is an empty font. A renderer that reads embedded fonts, such as FFmpeg's `subtitles` and `ass` filters, then needs no installed copy. The CLI does the same with `--embed-font FILE` on `create`, `captions` and `theme`, and `--embed-preset-fonts` on `captions` and `theme`. Each embedded font adds about four thirds of its own size to the file.

## libass extensions

These settings change rendering in libass, which FFmpeg uses. VSFilter ignores them or draws something else, so use them only for files rendered with libass.

| Setting | Effect |
| --- | --- |
| `BorderStyleEventBox` (4) | One box behind the whole event instead of one per line, colored by `ShadowColor` and padded by `Shadow`. VSFilter draws an outline and shadow. |
| `EncodingAuto` (-1) | Detects the text direction for right-to-left and mixed text. VSFilter rejects it. |
| `Script.Kerning` | Writes `Kerning: yes`, which turns on font kerning. |
| `Script.Language` | Writes `Language: xx` with a lowercase two-letter ISO 639-1 code such as `en`, which selects language-specific glyph forms. |

## Wrapping modes

| Numeric value | Go constant | Behavior |
| --- | --- | --- |
| 0 | `WrapStyleSmartTop` | Smart wrapping, wider upper line |
| 1 | `WrapStyleEndOfLine` | End-of-line wrapping |
| 2 | `WrapStyleForced`, `WrapStyleNoWrap` | No automatic wrapping; explicit line breaks remain available |
| 3 | `WrapStyleSmartBottom` | Smart wrapping, wider lower line |

The initial release swapped the names for modes 2 and 3. The constants now match ASS semantics. Callers using those names should check their intended behavior when updating. Literal numeric modes and existing ASS text retain their meaning.

## Headers, comments, and lossless edits

`ass.Script.Info` accepts additional Script Info headers, including renderer settings. Typed `Title`, `Resolution`, `Kerning` and `Language` fields and the generated `ScriptType` remain canonical, and `Info` cannot set them; duplicate, conflicting, or line-injecting header names are rejected. Set `Event.Comment` to emit a comment event rather than dialogue.

Use `Parse`, `Apply`, `Theme`, and `WithTextPosition` to preserve an existing file's layout and unrelated bytes. Field edits follow each section's `Format:` declaration. `create` builds a new document and does not perform a lossless round trip. Embedded attachments are preserved during edits, not decoded or installed as fonts.

`validate` checks supported document structure, style references, and event timing. It does not parse every literal override, verify glyph availability, or replace a render test.

## Format references

See the [libass ASS file format guide](https://github.com/libass/libass/wiki/ASS-File-Format-Guide) and [Aegisub override-tag reference](https://aegisub.org/docs/latest/ass_tags/). Historical ASS sketches and stalled ASSv5 drafts are not a complete specification for current ASS v4+ rendering.
