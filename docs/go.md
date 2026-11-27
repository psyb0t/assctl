---
title: Go library
description: Read, write, validate, and edit ASS scripts through the public Go package.
---

# Go guide

Import `github.com/psyb0t/assctl/pkg/ass`. All styles and events use the same structures as the CLI.

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/psyb0t/assctl/pkg/ass"
)

func main() {
    if err := writeCaptions(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func writeCaptions() error {
    script := ass.NewScript("Intro", 1080, 1920)
    script.AddStyle(ass.Style{
        Name: "Default", FontName: ass.FontDejaVuSans, FontSize: 64,
        ScaleX: 100, ScaleY: 100, Bold: true,
        BorderStyle: ass.BorderStyleOutlineDropShadow,
        Alignment: ass.AlignmentBottomCenter, Encoding: ass.EncodingDefault,
    })
    script.AddEvent(ass.Event{
        Start: time.Second, End: 3 * time.Second,
        Style: "Default", Text: "Hello, world",
    })
    position, err := ass.LookupTextPosition("portrait-lower-middle")
    if err != nil {
        return err
    }
    positioned, err := script.WithTextPosition(position)
    if err != nil {
        return err
    }
    return positioned.Encode(os.Stdout)
}
```

Use `Parse` to keep a complete UTF-8 ASS v4+ document, including comments, line endings, and unknown sections. `Apply` and `Theme` return a new document. Call `Validate` to check that referenced styles exist and event timings are ordered. `Contents` returns copies of the parsed fields.

`LookupPreset` and `Presets` expose the built-in palettes. Use `Preset.Captions` to build styles and events from timed text or `Document.Theme` to rework existing style definitions.

`Override.Tags` writes a typed override block. A caller can also put ASS tags directly in the event text, including karaoke timing tags. The package preserves those strings when it edits unrelated fields.

`SafeAreas` and `LookupSafeArea` return the TikTok, YouTube Shorts, Instagram Reels and combined safe areas, and `Document.CheckSafeArea` checks a document's margins against one; see [platform safe areas](positions.md#platform-safe-areas). `Script.Fonts` and `Document.EmbedFonts` embed `EmbeddedFont{Name, Data}` font files in the script; see [embedded fonts](ass-features.md#embedded-fonts). `BannerEffect`, `ScrollEffect` and `EscapeText` build event effects and literal text.

`TextPositions` and `LookupTextPosition` expose layouts independently of appearance. Apply them with `Script.WithTextPosition` or `Document.WithTextPosition`. See [text positions](positions.md) and [ASS features](ass-features.md) for geometry, headers, comments, and tag generation.

## Errors

Match errors with `errors.Is`:

| Error | Meaning |
| --- | --- |
| `commerr.ErrNotFound` | A named preset, text position, safe area, style, field or event does not exist. |
| `commerr.ErrInvalidValue` | A typed value cannot be written as valid ASS, such as a negative size, a bad color or a header that would break the file. |
| `ass.ErrInvalidDocument` | Input text is not a usable ASS v4+ document, or `Validate` found a missing style or bad timing. |
| `ass.ErrInputTooLarge` | `Parse` read more than 64 MiB. |

`commerr` is `github.com/psyb0t/ctxerrors/commerr`.
