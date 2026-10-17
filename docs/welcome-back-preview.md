# Welcome Back Edition — release preparation

This is an independently maintained community fork by HenricHuntsman. It continues the original MIT-licensed OpenCode codebase and is not an official release from the original maintainers or the Crush team.

## Included changes

- Reusable Agent Skills with project and user discovery.
- More conservative shell permission classification.
- Configured ordering and deduplication for overlapping context paths.
- Local model discovery with a shared three-second deadline.
- Ollama model context metadata, with a fallback when unavailable.
- Go CI, updated release configuration, and expanded setup documentation.
- Original MIT notices retained, with attribution for fork contributions.
- Fork package metadata and an installer with checksum verification, isolated temporary files, and offline failure tests.
- Explicit release version preserved instead of overwritten by module metadata.

## Release checks

Before publishing a numbered release:

1. Confirm the intended commit's CI passes.
2. Run `goreleaser check` and `goreleaser build --snapshot --clean`.
3. Run the Linux build's `--version` and `--help` commands.
4. Exercise an interactive session against a local model with tool calling. Verify permission prompts and one-shot output.
5. Test other supported platforms on suitable machines; cross-compilation alone is not a runtime test.
6. Confirm packaged archives include `LICENSE` and `README.md`, and verify their checksums.
7. Choose a new fork release tag and review release notes before publication.

## Additional validation

The installer has offline tests for explicit and latest versions, ARM64 asset selection, paths containing spaces, failed downloads, missing version metadata, invalid versions, checksum mismatches, and temporary-file cleanup. These tests use fixtures and do not download a real release.

The Linux preview also passed model discovery, streamed text and JSON output, and a file-view tool round trip against a deterministic local OpenAI-compatible fixture server. This is a protocol test, not a real Ollama/model test.

One-shot mode automatically approves session tool requests; the README documents this behavior.

## Current limitations

- A reported model context capacity may exceed the context allocated by the server.
- Shell command classification is a heuristic, not an operating-system sandbox.
- Provider integrations and individual models require compatibility testing.
- Preview builds are development artifacts, not a stability guarantee.

The upstream release history remains available at https://github.com/opencode-ai/opencode/releases. New fork builds belong at https://github.com/HenricHuntsman/opencode/releases.
