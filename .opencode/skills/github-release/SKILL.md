---
name: github-release
description: Cut a tagged GitHub release for this fork through the release workflow. Use only when the user explicitly asks to publish a release.
license: MIT
metadata:
  opencode/autoinvoke: "false"
---

## What I do

Publishing is irreversible, so confirm the version with the user first.

1. Confirm `main` is green: `gh run list --branch main --limit 3`.
2. Pick the next version after `git tag --sort=-creatordate | head -1`.
3. Tag and push: `git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z`. The tag triggers the `release` workflow.
4. Watch it: `gh run watch <id> --exit-status`.
5. Verify assets with `gh release view vX.Y.Z --json assets` and check a downloaded archive against `checksums.txt`.

## Notes

- The workflow publishes Linux and macOS archives, `.deb`/`.rpm` packages and `checksums.txt`.
- Validate `.goreleaser.yml` changes first with `goreleaser check` and a local `goreleaser build --snapshot --clean --single-target`.
- Tags created from this environment are unsigned.
