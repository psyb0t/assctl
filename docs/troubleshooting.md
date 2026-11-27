---
title: Troubleshooting
description: Fix rejected input, invalid timings, missing fonts, and failed writes.
---

# Troubleshooting

## The font looks wrong

Run `assctl fonts-install` on the machine that renders your subtitles. Check the family with `assctl presets --json`. On Linux, refresh Fontconfig with `fc-cache -f` and restart the renderer. Inline font overrides may take precedence over a theme.

## Embedding fonts fails

`--embed-preset-fonts` embeds only the fonts of the built-in presets. With a `--theme-file` that names another font, pass that font's file with `--embed-font` instead. A font file is limited to 64 MiB. A font whose file name is already in the script, ignoring case, is rejected; rename the file or leave it out.

## The safe-area check fails

`safe-area` lists each style or event whose margin is smaller than the platform needs, with the margin it has and the one it needs. Apply a social preset or a named position, or raise that margin with `edit`. See [platform safe areas](positions.md#platform-safe-areas).

## Burning fails

`burn` needs `ffmpeg` and `ffprobe` on your `PATH`; install FFmpeg or use the [Docker image](install.md#docker). It encodes H.264, HEVC, VP9 and AV1 video and rejects other codecs. When FFmpeg refuses to write a copied stream, write the output in the input's container, such as `.mkv` for a Matroska input. `--crf` must fall in the encoder's range listed in the [CLI guide](cli.md#burn-captions-into-a-video).

## The font download fails

A first install requires access to the selected font's pinned upstream download. Sources include GitHub and DaFont. Restore connectivity and retry. Verified installed files are reused offline. Checksum failures stop installation before files are changed.

## Validation rejects the file

Input must be UTF-8 ASS v4+ with a `[Script Info]` section. Each record must match its section's `Format:` columns. Event text goes last. Styles need distinct, nonempty names. Events must reference an existing style, and end after they start. Validation errors identify the rejected line.

Input is limited to 64 MiB. Use `\N` in ASS text for a line break, escaped as `\\N` inside a JSON string.

## Event timing is rejected

Typed JSON durations use nanoseconds. ASS timestamps have centisecond precision, so a new event needs at least one representable centisecond between its start and end. Negative starts and equal or reversed intervals are rejected.

## An edit fails

Event indices start at one and include comment rows. Inspect the file to see available styles and field names. The same row or case-insensitive field cannot appear twice in one edit. Failed edits preserve the original document.

## Writing the output fails

Create the parent directory and check write permissions. A selected output file can replace an existing file. Successful output writes use a temporary file and rename. Use a new output path while experimenting.
