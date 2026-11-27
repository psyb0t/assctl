# Changelog

## v0.1.0 (2026-11-27)

Adds social caption looks, independent text positions, typed override tags, and a single verified font installer. Go callers have three breaking changes to migrate:

- `ass.ErrNotFound` and `ass.ErrInvalidValue` are removed. Match `commerr.ErrNotFound` and `commerr.ErrInvalidValue` from `github.com/psyb0t/ctxerrors/commerr` with `errors.Is` instead. `ass.ErrInvalidDocument` and `ass.ErrInputTooLarge` are unchanged.
- `WrapStyleSmartBottom` is now 3 and `WrapStyleForced` is 2, the values libass (the subtitle renderer FFmpeg uses) gives them. v0.0.1 had them swapped. Code that uses the constants now gets the wrapping it asked for; code that wrote the numbers needs to swap them.
- `Color.Ptr` returns `*Color` instead of `*string`, so it fills `Override` color fields.

Changes:

- Added seven social caption looks with TikTok Sans Bold, Barlow Condensed ExtraBold, Anton, and THE BOLD FONT's free version, with resolution-scaled horizontal margins and lower-middle, center, and upper-hook placement.
- Added independent text positions, the `position` and `positions` commands, and `--text-position` for creation and theming.
- Added `tags` for typed override generation: karaoke, drawing, vector clips, simple fades, style resets, spacing, and axis-specific borders and shadows.
- Added custom Script Info headers and comment event creation. With custom headers, `PlayResX` and `PlayResY` follow them. Scripts without custom headers are unchanged. Header names containing `;` are rejected.
- Added `ass.DefaultPreset`, font family constants for the built-in presets, and `Alpha.Ptr`.
- Changed `fonts-install` to use one verified installer for every preset font. It downloads each archive once, fetches only missing or changed files, and checks every file before writing any. On Windows it also registers each font for the current user.
- Changed the canvas size used for themes and positions when a script omits `PlayResX` or `PlayResY` to the size libass assumes, so margins land where the renderer draws them.
- Rejected duplicate JSON fields, ambiguous ASS Format columns, duplicate header edits, invalid generated UTF-8, negative style geometry, conflicting typed overrides, malformed vector clips, invalid fade timings, and non-finite geometry.
- Fixed partial writes and help-output failures being reported as success.
- Fixed reader and writer disagreeing at the maximum timestamp duration, and rejected timestamp arithmetic overflow.
- Fixed a JSON spec over 64 MiB being reported as an oversized ASS file.
- Added tests that render every preset and position with libass, fuzz tests for the reader and writer, and a font download check that runs with `make test-real`. CI fails below 90% test coverage.

## v0.0.1 (2026-11-27)

Initial release of the ASS subtitle CLI and Go library.

- Added commands to create, inspect, edit, validate, and theme ASS subtitle scripts.
- Added four presets with fonts, colors, backgrounds, borders, shadows, and placement.
- Preserved untouched text, karaoke tags, and metadata when editing existing ASS files.
- Added Cobra command help, shell completion, and flags before or after the input path.
- Added `fonts-install` to verify and install preset fonts in the user's font directory.
- Published CLI and Go guides with search and the shared GitHub Pages workflow.
- Added release checks, badges, skill publishing, repository mirrors, and funding links.
