---
title: Development
description: Build and test assctl and its Docusaurus documentation in Docker.
---

# Development and releases

- `make dep` resolves checksums and refreshes the committed vendor directory.
- `make format`, `make lint`, and `make lint-fix` run in the pinned development image.
- `make test` runs the complete offline suite, including real FFmpeg/libass renders and CLI subprocess checks, with Go's race detector.
- `make test-unit` excludes render tests; `make test-integration` runs the renderer tests alone, including `burn` on H.264, HEVC, VP9, AV1 and rotated videos. A missing renderer is a test failure.
- `make test-coverage` runs the complete offline suite, writes the badge percentage, and fails below 90% coverage.
- `make test-fuzz` fuzzes parser preservation and generated-script round trips with two workers and bounded runtimes. Their regression seeds also run in normal tests.
- `make test-real` downloads every preset's pinned fonts into temporary user directories, verifies installation, checks that libass selects the intended font when rendering, and renders captions made with `--embed-preset-fonts` with no installed fonts. It requires upstream network access and checks verified offline reuse afterward. These tests, in `internal/pkg/fonts` and `cmd/assctl`, sit behind the `real` build tag, so no other target compiles them.
- `make test-image` builds the production `Dockerfile` under a temporary tag, then checks it with networking off: the stamped version, and for every preset, `captions --embed-preset-fonts` and an FFmpeg render in which libass selects the installed preset font, and `burn` on a test video. It removes the image afterward. Building the image downloads the preset fonts.
- `make help` lists the available operations.
- `make sec` checks for secrets and known Go vulnerabilities. It writes `sec.sarif` for GitHub Code Scanning.
- `make docs` builds the Docusaurus site and checks internal links. The output is `website/build` under the `/assctl/` URL prefix.
- `make docs-lock` refreshes the pinned documentation lockfile. Build tooling runs in a pinned Node container with a seven-day dependency age gate and install scripts disabled.

A push to the default branch builds the Docker image for `linux/amd64` and `linux/arm64` and publishes `psyb0t/assctl:latest` and `psyb0t/assctl:sha-<commit>`. Tag an already checked default-branch commit for a release. The tag workflow waits for that exact run, promotes its image to `psyb0t/assctl:vX.Y.Z` by digest without rebuilding, creates the GitHub release from the annotated tag, and publishes only the `assctl` skill to ClawHub. Git tags supply the Go module version. Keep the Codex manifest's version in sync when cutting the release.

The pipeline needs the repository's `CLAWHUB_TOKEN` secret with permission to publish the package, and `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` to push the image. GitHub supplies its own token for releases, scanning, and badges.

The code workflow runs coverage, bounded fuzzing, and real-font verification before publication. Its real-font stage requires upstream network access; local `make test` remains offline.

The normal renderer suite checks every appearance/position pair on portrait, landscape, and square canvases. It uses the development image's DejaVu font for deterministic geometry. It also checks drawing and clipping bounds, movement, transforms, rotation, transparency, comments, event intervals, wrapping, fades, and karaoke progression. The separate real-font suite verifies each preset's actual upstream family. No font binaries are stored in the repository.

Unit and boundary tests cover JSON ambiguity, record and tag injection, invalid UTF-8, input limits, numeric limits, interrupted downloads, archive size and checksum failures, immutable concurrent edits, and output failures. Tests follow their source filenames and use the same Go packages. CLI process tests assert exit status, stdout, and stderr.

The documentation caller uses the shared Pages workflow. Pull requests build the site; default-branch pushes deploy it. GitHub Pages must use GitHub Actions as its publishing source. Local search, the sitemap, and `llms.txt` are part of the static output. Visitor tracking loads the shared `https://psyb0t.github.io/analytics.js` script.
