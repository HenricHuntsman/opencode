# Archived upstream recovery review

Reviewed 18 October 2026 (Melbourne). Fork baseline: `9c101864c3d4b6eb5580c5b154948187a87f3b9c`. Upstream main: `73ee493265acf15fcd8caab2bc8cd3bd375b63cb`.

## Scope and method

Inventoried all 25 upstream branches, including main, and all 42 open pull requests returned in one page. Compared the 24 non-main branches using `git cherry HEAD archive/<branch>`, which detects patch-equivalent non-merge commits. An unmatched patch does not prove missing behavior: squash merges and rewritten implementations can evade patch equivalence. This is a history inventory plus selected code review, not a build certification of every branch or PR.

## Branch inventory

| Branch | Equivalent patches | Unmatched patches |
| --- | ---: | ---: |
| adam/keybinds | 0 | 1 |
| adam/themes | 0 | 9 |
| add-xai | 1 | 0 |
| bedrock | 10 | 0 |
| bubbletea_v2 | 3 | 1 |
| change-db-driver | 3 | 0 |
| cleanup-logs-status | 12 | 0 |
| cleanup-tools | 5 | 0 |
| feature/context-warning-compact-cmd | 0 | 8 |
| feature/improve-compact | 0 | 3 |
| feature/openrouter-deepseek | 1 | 0 |
| fix-acc-error | 1 | 0 |
| fix-default-config | 1 | 0 |
| fix-gemini | 1 | 0 |
| fix-help | 1 | 0 |
| fix-kitty | 1 | 0 |
| fix-mcp-tools | 1 | 0 |
| fix-warn-message | 1 | 0 |
| openai-compat-providers | 0 | 3 |
| opencode | 1 | 11 |
| pre-release-fixes | 1 | 0 |
| update-gemini-readme | 1 | 0 |
| update-name | 1 | 0 |
| ux-improvements | 5 | 0 |

Sixteen branches have no unmatched non-merge patches. Do not wholesale-merge these historic branches.

## Selected code review and recovery decisions

- **PR #281 — parameterless MCP calls:** confirmed missing on fork main. Normalize empty/whitespace input to an empty JSON object. Implemented in this change, with fake-client tests proving tool invocation, argument preservation, malformed-input rejection, and client closure. Credit: gedzeppelin, https://github.com/opencode-ai/opencode/pull/281.
- **feature/improve-compact:** its central behavior already exists: SummaryMessageID persistence, history filtering, and same-session summary storage. A wholesale merge would also revert later logging/context changes. No import.
- **adam/themes:** the listed themes (including Flexoki, One Dark, Monokai, Dracula, Tokyo Night, Tron) already exist in the fork. No whole-branch import.
- **adam/keybinds:** partly present (Ctrl+S session switching). Proposed Ctrl+M model selection needs terminal/Enter conflict testing. No shortcut changes here.
- **bubbletea_v2:** final commit explicitly says WIP; changes about 30 files. Treat as a separate migration project, not a recovered fix.
- **openai-compat-providers / PR #98:** draft provider configuration redesign. Review separately against the current provider abstraction.
- **PR #346 — Ollama integration:** mixes local endpoint discovery, spec-server configuration, extra cloud providers, and configuration commands. Evaluate native Ollama ideas separately; do not overwrite current bounded discovery or add default external tools.
- **PR #348 — Todo System:** promising task-list persistence/UI. Requires migration, tool validation, and sidebar tests. The patch adds repetitive mandatory-task-list prompt text; that text is not needed to implement the feature. Deferred to a dedicated feature change.
- **PR #167 — permissions:** useful motivation, but the reviewed patch has incorrect import paths (including foreign Sourcegraph internal paths). Do not import directly. Design permissions in the fork's actual package structure.
- **PR #336 — one-shot compaction:** adds synchronous compaction in tool loops, but rebuilds history by appending the current user message after reloading persisted messages. Check duplicate-user messages, tool-call/result pairing, token estimation, and cancellation before adapting. No import.
- **PR #308 — LOCAL_API_KEY:** only adds discovery-request authorization in the reviewed patch, not an end-to-end authentication design; its README example also changes config structure. Review model discovery and inference authorization together before adopting.

## Open pull-request inventory

Only the selected entries above received code-level review in this pass. Titles below are an inventory, not endorsements.

| PR | Title | Draft |
| --- | --- | --- |
| [#348](https://github.com/opencode-ai/opencode/pull/348) | Todo System | No |
| [#346](https://github.com/opencode-ai/opencode/pull/346) | Ollama integration | No |
| [#344](https://github.com/opencode-ai/opencode/pull/344) | Update README.md with Amazon Bedrock | No |
| [#336](https://github.com/opencode-ai/opencode/pull/336) | feat(autocompcat): for non interactive mode | No |
| [#329](https://github.com/opencode-ai/opencode/pull/329) | Add DeepSeek integration support | No |
| [#327](https://github.com/opencode-ai/opencode/pull/327) | fix: typo in keyboard shortcuts window | No |
| [#323](https://github.com/opencode-ai/opencode/pull/323) | add groq Kimi K2 model support | No |
| [#316](https://github.com/opencode-ai/opencode/pull/316) | feat: add OpenRouter x-ai/grok-4 model support | No |
| [#315](https://github.com/opencode-ai/opencode/pull/315) | feat(models): add OpenRouter Kimi K2 model support | No |
| [#314](https://github.com/opencode-ai/opencode/pull/314) | Opencodedigzo.json | No |
| [#312](https://github.com/opencode-ai/opencode/pull/312) | Adding the grok4 model information to the xai models - verifying info… | No |
| [#310](https://github.com/opencode-ai/opencode/pull/310) | feat: add DeepSeek API support | No |
| [#308](https://github.com/opencode-ai/opencode/pull/308) | add LOCAL_API_KEY environment variable to support api like ChatAnywhere | No |
| [#307](https://github.com/opencode-ai/opencode/pull/307) | feat: add xAI Grok integration with its current features | No |
| [#301](https://github.com/opencode-ai/opencode/pull/301) | fix: gemini 429 response | No |
| [#294](https://github.com/opencode-ai/opencode/pull/294) | Fix GitHub Copilot authentication flow | No |
| [#289](https://github.com/opencode-ai/opencode/pull/289) | feat: Transparent theme | No |
| [#287](https://github.com/opencode-ai/opencode/pull/287) | feat: add claude 4 sonnet/opus to openrouter models | No |
| [#281](https://github.com/opencode-ai/opencode/pull/281) | fix(mcp): handle empty input by initializing with empty JSON object | No |
| [#272](https://github.com/opencode-ai/opencode/pull/272) | chore: update mcp-go dependency to v0.32.0 | No |
| [#268](https://github.com/opencode-ai/opencode/pull/268) | Feature/shell improvements /w Windows support | No |
| [#250](https://github.com/opencode-ai/opencode/pull/250) | Add GitHub Action for golangci-lint | No |
| [#249](https://github.com/opencode-ai/opencode/pull/249) | Add dependabot config | No |
| [#245](https://github.com/opencode-ai/opencode/pull/245) | feat: Add support for custom provider endpoints and headers | No |
| [#243](https://github.com/opencode-ai/opencode/pull/243) | feat(session and litellm): add session deletion flow with Ctrl+P and overlay dialog; support LiteLLM (or any other) proxy; extend vertex models | No |
| [#239](https://github.com/opencode-ai/opencode/pull/239) | fix model typo and limit project size in system prompt to 4K max | No |
| [#238](https://github.com/opencode-ai/opencode/pull/238) | feat(models): add latest openrouter and gemini models | No |
| [#235](https://github.com/opencode-ai/opencode/pull/235) | Add Flake Dev Environment and Optional Package Output | No |
| [#227](https://github.com/opencode-ai/opencode/pull/227) | feat: hide cursor when process loses focus | No |
| [#216](https://github.com/opencode-ai/opencode/pull/216) | OpenAI baseUrl support | No |
| [#192](https://github.com/opencode-ai/opencode/pull/192) | feat(vertexai): update Gemini model versions and add new preview models | No |
| [#170](https://github.com/opencode-ai/opencode/pull/170) | Create SECURITY.md | No |
| [#167](https://github.com/opencode-ai/opencode/pull/167) | Enhance privacy and security by adding permission checks and improvin… | No |
| [#163](https://github.com/opencode-ai/opencode/pull/163) | docs: fix typos | No |
| [#149](https://github.com/opencode-ai/opencode/pull/149) | Support global context paths by merging built-in defaults with user-specified paths | No |
| [#142](https://github.com/opencode-ai/opencode/pull/142) | fix: improve script safety and clarity in path and temp handling | No |
| [#141](https://github.com/opencode-ai/opencode/pull/141) | docs: add Traditional Chinese localization and improve README clarity | No |
| [#110](https://github.com/opencode-ai/opencode/pull/110) | chore: adds nix and nixflakes environment | No |
| [#104](https://github.com/opencode-ai/opencode/pull/104) | feat: add golangci && resolve current lint errors | No |
| [#98](https://github.com/opencode-ai/opencode/pull/98) | WIP Provider Config | Yes |
| [#81](https://github.com/opencode-ai/opencode/pull/81) | Improve language in conversation example about prime | No |
| [#80](https://github.com/opencode-ai/opencode/pull/80) | fix tiny typo, "tune"->"tone" | No |

## Next focused work

1. Persistent task lists with explicit migration and data-validation tests.
2. One-shot compaction with realistic multi-turn tool histories.
3. Local endpoint authentication covering discovery and inference.
4. Permission review for network-capable tools, preserving the local-only workflow.

Preserve contributor attribution and the existing MIT notices for any adapted code. No upstream branch was merged wholesale in this pass.
