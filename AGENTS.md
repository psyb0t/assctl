# assctl

The public ASS library is in `pkg/ass`. The `cmd/assctl` command reads files and flags, calls the library, and writes results. Keep the two surfaces on the same code paths.

Read [the ASS rules](.agents/rules/ass.md) before changing formatting or presets. Run `make check` before opening a pull request.

Authored documentation lives in `docs/`. Docusaurus reads it directly through `website/`. Add guides to `website/sidebars.mjs` and run `make docs` to check links and produce `website/build`. The shared Pages workflow publishes that output using GitHub Actions.

`internal/pkg/fonts` downloads and verifies the pinned DejaVu fonts. Keep the archive checksum, per-file checksums, and download limits in sync. Tests use a local HTTP server and temporary font directories.
