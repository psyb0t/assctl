# ASS edits

- Keep untouched lines byte-for-byte, including line endings, embedded data, comments, and unknown sections.
- Respect each section's `Format:` columns. The final event text field may contain commas and inline ASS tags.
- Named themes belong in `pkg/ass`. The Go API and CLI use the same presets.
- Never evaluate ASS text as shell input. Test the writer and parser, then exercise commands against complete subtitle files.
