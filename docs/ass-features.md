---
title: ASS features
description: Typed tags, drawing, karaoke, script headers, wrapping modes, and preservation boundaries.
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
| Style and wrapping | `ResetStyle`, `WrapStyle` |

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

Typed generation rejects conflicting position/movement, rectangular/vector clipping, and simple/full fades. It checks finite coordinates, timing order, vector-clip drawing syntax, and whether transform fields are animatable. Literal tags in event text remain authored input and can override earlier generated tags.

## Wrapping modes

| Numeric value | Go constant | Behavior |
| --- | --- | --- |
| 0 | `WrapStyleSmartTop` | Smart wrapping, wider upper line |
| 1 | `WrapStyleEndOfLine` | End-of-line wrapping |
| 2 | `WrapStyleForced`, `WrapStyleNoWrap` | No automatic wrapping; explicit line breaks remain available |
| 3 | `WrapStyleSmartBottom` | Smart wrapping, wider lower line |

The initial release swapped the names for modes 2 and 3. The constants now match ASS semantics. Callers using those names should check their intended behavior when updating. Literal numeric modes and existing ASS text retain their meaning.

## Headers, comments, and lossless edits

`ass.Script.Info` accepts additional Script Info headers, including renderer settings. Typed `Title` and `Resolution` fields and the generated `ScriptType` remain canonical; duplicate, conflicting, or line-injecting header names are rejected. Set `Event.Comment` to emit a comment event rather than dialogue.

Use `Parse`, `Apply`, `Theme`, and `WithTextPosition` to preserve an existing file's layout and unrelated bytes. Field edits follow each section's `Format:` declaration. `create` builds a new document and does not perform a lossless round trip. Embedded attachments are preserved during edits, not decoded or installed as fonts.

`validate` checks supported document structure, style references, and event timing. It does not parse every literal override, verify glyph availability, or replace a render test.

## Format references

See the [libass ASS file format guide](https://github.com/libass/libass/wiki/ASS-File-Format-Guide) and [Aegisub override-tag reference](https://aegisub.org/docs/latest/ass_tags/). Historical ASS sketches and stalled ASSv5 drafts are not a complete specification for current ASS v4+ rendering.
