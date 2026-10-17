---
name: go-verify
description: Run this repository's full Go verification (gofmt, vet, build, tests) and report only real failures. Use before committing or opening a pull request.
license: MIT
---

## What I do

Run the same checks CI runs, in order, and stop at the first failing stage.

1. `gofmt -l .` must print nothing. Fix with `gofmt -w` on only the listed files.
2. `go vet ./...`
3. `go build ./...`
4. `go test ./... -count=1`

## Notes

- Warnings about `rg` or `fzf` missing from `$PATH` are harmless.
- Tests that use global config must call `config.Load` with a temp directory and set `WorkingDir` explicitly.
- Do not use `-count` above 1 on `internal/llm/prompt`; its context cache is process-wide.
- Report exact failing test names and the first error line, not the whole log.
