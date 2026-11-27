---
title: Go library
description: Read, write, validate, and edit ASS scripts through the public Go package.
---

# Go guide

Import `github.com/psyb0t/assctl/pkg/ass`. All styles and events use the same structures as the CLI.

```go
script := ass.NewScript("Intro", 1080, 1920)
script.AddStyle(ass.Style{
    Name: "Default", FontName: "DejaVu Sans", FontSize: 64,
    ScaleX: 100, ScaleY: 100, Bold: true,
    BorderStyle: ass.BorderStyleOutlineDropShadow,
    Alignment: ass.AlignmentBottomCenter, Encoding: ass.EncodingDefault,
})
script.AddEvent(ass.Event{
    Start: time.Second, End: 3 * time.Second,
    Style: "Default", Text: "Hello, world",
})
if err := script.Encode(output); err != nil {
    return err
}
```

Use `Parse` to keep a complete UTF-8 ASS v4+ document, including comments, line endings, and unknown sections. `Apply` and `Theme` return a new document. Call `Validate` to check that referenced styles exist and event timings are ordered. `Contents` returns copies of the parsed fields.

`LookupPreset` and `Presets` expose the built-in palettes. Use `Preset.Captions` to build styles and events from timed text or `Document.Theme` to rework existing style definitions.

`Override.Tags` writes a typed override block. A caller can also put ASS tags directly in the event text, including karaoke timing tags. The package preserves those strings when it edits unrelated fields.
