# DragonForge Code — independent project preparation

This branch prepares an **independent GitHub repository** at `HenricHuntsman/dragonforge-code` while preserving this original OpenCode Revival fork and its GitHub issues/pull requests. **Do not unarchive or delete `opencode-ai/opencode`.**

## Intended architecture
- One Go coding agent (the revived OpenCode Go engine).
- Local Ollama/Qwen mode, no required cloud account.
- A standalone engine mode.
- An isolated Nintendo 64 specialist workspace with read-only ROM utilities.
- A separate TraceForge **reference** workspace (not the existing TraceForge checkout).
- Build/test commands and GitHub Actions.

## Migration requirements

Create the new GitHub repository as **standalone, not as a GitHub fork**. Preserve the original `LICENSE` notice, contributors and main Git ancestry. Copy this repository into a new directory, update Go module imports **in the new copy only**, run `go mod verify`, `go vet ./...`, `go test ./...`, and `go build ./...`. Only after verification create and push the new standalone repository. Leave the current fork unchanged, including its PR #1–#10 history.

The working and locally checked migration/launcher scripts are available as the **DragonForge independence kit ZIP attached in the ChatGPT project session**. That package includes the N64 skill, model-selection launcher, a `gh repo create` bootstrap and focused local tests.

## What is not complete

A new GitHub repo has **not** been created through this PR. This ChatGPT GitHub connector has source, branch and PR write access, but no `create repository` action. The complete kit still needs to be installed/run from Ubuntu, where `gh auth login` can authorize `gh repo create`. This branch is preparatory, not a substitute for the new independent repo.

The launcher is not a host filesystem sandbox: review AI tool permissions, and never point it directly at the user's existing TraceForge source without explicit approval.
