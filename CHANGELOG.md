# Changelog

## v0.3.0 (2026-11-27)

assctl now ships as the `psyb0t/assctl` Docker image, with FFmpeg and libass and every built-in preset's font already verified and installed, so it can create, check and burn captions with no Go, no font download and no network.

- Added the `psyb0t/assctl` image for `linux/amd64` and `linux/arm64`. Default-branch builds publish `latest` and `sha-<commit>`, and each release promotes its commit's image to `vX.Y.Z` without rebuilding.
- `assctl version` prints the version stamped at build time when the binary has no module version, such as `sha-<commit>` in the image.

## v0.2.0 (2026-11-27)

Social captions now stay out from under TikTok, YouTube Shorts and Instagram Reels buttons, using each platform's published safe area, and scripts can carry their own fonts. Go callers and existing specs have three breaking changes to migrate:

- The `Effect` constants are removed. libass ignores `Banner`, `Scroll up` and `Scroll down` without their numbers, and does not know `Scroll`, `Scroll left`, `Scroll right` or `Karaoke` as Effect field values. Build effects with `ass.BannerEffect` and `ass.ScrollEffect`. Karaoke is unaffected: it comes from the `\k`, `\kf`, `\ko` and `\kt` tags, which `Override.Karaoke` and `Override.KaraokeStart` still write.
- `Script.Info` no longer accepts `Kerning` or `Language`. Set `Script.Kerning` and `Script.Language` instead.
- Styles with a `BorderStyle` other than 0, 1, 3 or 4, or an `Encoding` below -1, are now rejected when a script is written. Renderers do not support other values.

Changes:

- Added `safe-area` and `safe-areas`, and `ass.SafeArea`, `ass.SafeAreas`, `ass.LookupSafeArea`, `ass.SafeAreaAll`, `Document.CheckSafeArea`, `ass.SafeAreaIssue` and the `ass.Edge*` constants. The check reports every style or event whose margins reach under a platform's buttons and captions. The areas come from TikTok's In-Feed safe zone template, Google's vertical video safe zones and Meta's Reels ad guide.
- Changed `portrait-center`, `portrait-lower-middle` and the lower and centered social presets to a left margin of 120/1080 and a right margin of 300/1080 of the width, clear of TikTok's button column. `social-hook` and `portrait-upper` keep 18%. Added `Preset.RightMarginRatio` for custom themes.
- Added embedded fonts: `Script.Fonts`, `ass.EmbeddedFont`, `Document.EmbedFonts`, `--embed-font FILE` on `create`, `captions` and `theme`, and `--embed-preset-fonts` on `captions` and `theme`, which installs the preset's fonts and embeds them. FFmpeg renders with them without installed fonts.
- Added `ass.BorderStyleEventBox` (one box behind the whole event), `ass.EncodingAuto` (automatic text direction), and `Script.Kerning` and `Script.Language` headers. These are libass extensions.
- Added `Override.ResetScale` for `\fsc`, and a `Transform` with `End` 0 now runs until the event ends.
- Added `ass.EscapeText` and `ass.ScrollDirection`. `captions` now escapes `{` in plain dialogue, which used to hide the text after it, and still wraps and uppercases text that contains braces.
- Changed margins from ratios to round up to a whole pixel instead of down, so positions and social presets stay inside the safe areas at every video size, and numbers in generated scripts to at most three decimals, so a font size prints as `86.4` rather than `86.39999999999999`.
- Fixed `PlayResX` or `PlayResY` of zero or less failing themes and positions. They are now treated as missing and inferred, as libass does.

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
