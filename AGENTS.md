# assctl

The public ASS library is in `pkg/ass`. The `cmd/assctl` command reads files and flags, calls the library, and writes results. Keep the two surfaces on the same code paths.

Read [the ASS rules](.agents/rules/ass.md) before changing formatting or presets. Run `make check` before opening a pull request.

Keep tests beside the matching source file, in the same Go package. Normal tests run offline in the development container with FFmpeg/libass available; missing renderer binaries must fail. `make test-real` is the explicit upstream-font download check, built only with the `real` tag. Tests use testify `require` and `assert`. Errors wrap the shared `commerr` sentinels from `github.com/psyb0t/ctxerrors/commerr`; a package declares its own sentinel in `errors.go` only when no shared one fits. Keep the complete offline suite in `make test-coverage`, including renderer integration tests, and preserve its 90% floor.

Authored documentation lives in `docs/`. Docusaurus reads it directly through `website/`. Add guides to `website/sidebars.mjs` and run `make docs` to check links and produce `website/build`. The shared Pages workflow publishes that output using GitHub Actions.

`internal/pkg/fonts` downloads and verifies preset fonts from pinned upstream sources. Keep source URLs, archive and per-file checksums, accompanying notices, and download limits in sync. Tests use a local HTTP server and temporary font directories.
