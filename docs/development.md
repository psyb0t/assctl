---
title: Development
description: Build and test assctl and its Docusaurus documentation in Docker.
---

# Development and releases

- `make dep` resolves checksums and refreshes the committed vendor directory.
- `make format`, `make lint`, and `make test` run in the pinned development image.
- `make test-coverage` writes the coverage number for the CI badge.
- `make sec` checks for secrets and known Go vulnerabilities. It writes `sec.sarif` for GitHub Code Scanning.
- `make docs` builds the Docusaurus site and checks internal links. The output is `website/build` under the `/assctl/` URL prefix.
- `make docs-lock` refreshes the pinned documentation lockfile. Build tooling runs in a pinned Node container with a seven-day dependency age gate and install scripts disabled.

Tag an already checked default-branch commit for a release. The tag workflow waits for that exact run, creates the GitHub release from the annotated tag, and publishes only the `assctl` skill to ClawHub. Git tags supply the Go module version. Keep the Codex manifest's version in sync when cutting the release.

The pipeline needs the repository's `CLAWHUB_TOKEN` secret with permission to publish the package. GitHub supplies its own token for releases, scanning, and badges.

The documentation caller uses the shared Pages workflow. Pull requests build the site; default-branch pushes deploy it. GitHub Pages must use GitHub Actions as its publishing source. Local search, the sitemap, and `llms.txt` are part of the static output. Visitor tracking loads the shared `https://psyb0t.github.io/analytics.js` script.
