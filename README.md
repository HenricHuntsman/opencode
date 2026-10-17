# OpenCode

**Your code. Your terminal. Your choice of AI.**

A terminal coding assistant maintained by HenricHuntsman, with local model support, reusable skills, and tools to inspect, edit, and test your projects. Use the interactive terminal UI to work with a project, or run a single prompt from a script.

## Capabilities

- Chat with supported models from Anthropic, OpenAI, Google Gemini, GitHub Copilot, Groq, OpenRouter, xAI, Azure OpenAI, AWS Bedrock, Google Cloud Vertex AI, and compatible local endpoints.
- Search and inspect source files, edit files, apply patches, run shell commands, and request language-server diagnostics.
- Connect external tools through Model Context Protocol (MCP) over stdio or server-sent events (SSE).
- Load reusable Agent Skills from project and user `SKILL.md` files.
- Keep conversations and sessions in local SQLite storage.
- Run one-shot prompts with text or JSON output.
- Review permission requests before file writes, shell commands, and external tool execution.

## Requirements

- Go 1.24 or later to build from source.
- A terminal and access to a supported model provider or local OpenAI-compatible server. Local servers do not require a cloud subscription.
- Optional: language servers for diagnostics and MCP servers for additional tools.

## Build

```sh
git clone https://github.com/HenricHuntsman/opencode.git
cd opencode
go build -o opencode .
```

Set a provider credential and start the interactive app:

```sh
export ANTHROPIC_API_KEY="your-api-key"
./opencode
```

You can install the binary in a directory on your `PATH`:

```sh
install -Dm755 ./opencode "$HOME/.local/bin/opencode"
```

## Command line

```sh
# Start the interactive terminal UI
opencode

# Open a project directory
opencode --cwd /path/to/project

# Run one prompt and print text
opencode --prompt "Explain the main packages in this repository"

# Print one-shot output as JSON
opencode --prompt "Summarize this project" --output-format json

# Hide the spinner for scripted use
opencode --prompt "Explain this function" --quiet

# Show the build version or command help
opencode --version
opencode --help
```

| Flag | Short form | Description |
| --- | --- | --- |
| `--help` | `-h` | Show command help |
| `--version` | `-v` | Show the build version |
| `--debug` | `-d` | Enable debug logging |
| `--cwd` | `-c` | Set the working directory |
| `--prompt` | `-p` | Run one prompt and exit |
| `--output-format` | `-f` | Select `text` or `json` output for one-shot mode |
| `--quiet` | `-q` | Hide the spinner in one-shot mode |

## Providers and configuration

OpenCode reads JSON configuration from `~/.opencode.json`, `$XDG_CONFIG_HOME/opencode/.opencode.json`, or `~/.config/opencode/.opencode.json`. A project-level `.opencode.json` is merged with user configuration.

Provider credentials can be set in the environment or in the `providers` object in the config file.

| Provider | Environment variables or credentials |
| --- | --- |
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Google Gemini | `GEMINI_API_KEY` |
| Groq | `GROQ_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| xAI | `XAI_API_KEY` |
| Azure OpenAI | `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_API_KEY` |
| Google Cloud Vertex AI | `VERTEXAI_PROJECT`, `VERTEXAI_LOCATION`, or Google Cloud credentials |
| AWS Bedrock | AWS SDK credentials, profiles, and region settings |
| Local OpenAI-compatible server | `LOCAL_ENDPOINT` |
| GitHub Copilot | A Copilot-enabled GitHub token or supported local Copilot credentials |

Example `~/.opencode.json`:

```json
{
  "providers": {
    "anthropic": {
      "apiKey": "your-api-key"
    }
  },
  "agents": {
    "coder": {
      "model": "claude-4-sonnet"
    }
  },
  "autoCompact": true
}
```

Keep credentials out of source control. Environment variables are recommended for local development and automation.

The default data directory is `.opencode`. Use the `data.directory` setting to choose another location. Other settings include `agents`, `mcpServers`, `lsp`, `shell`, `contextPaths`, `debug`, `debugLSP`, `autoCompact`, and `tui.theme`.

## Local models (Ollama, LM Studio)

Point `LOCAL_ENDPOINT` at any OpenAI-compatible server. Models are discovered at startup and appear as `local.<name>`.

```sh
# Ollama
LOCAL_ENDPOINT=http://localhost:11434/v1 opencode
```

For Ollama, OpenCode queries `/api/show` for the model's reported context capacity when the model listing does not provide a loaded context length. This capacity is not a measurement of the context allocated to a running Ollama session. LM Studio's reported loaded context length takes precedence when available; otherwise the fallback is 4096 tokens.

Discovery has a shared three-second deadline across listing and metadata requests, with a 1.5-second timeout per request. Models already discovered remain available if metadata requests time out, using the fallback where needed. An unreachable server therefore does not stall startup indefinitely.

Select a model with `agents.coder.model`, for example `local.qwen2.5-coder:7b`. The model and server must support tool calling for coding-agent operations. Start Ollama and download the model before launching OpenCode:

```sh
ollama pull qwen2.5-coder:7b
LOCAL_ENDPOINT=http://localhost:11434/v1 opencode
```

## Agent Skills

Skills provide reusable instructions loaded on demand by the coding agent through the `skill` tool. Create a directory whose name matches the skill name, containing a `SKILL.md` file:

```text
.opencode/skills/project-checks/SKILL.md
```

```markdown
---
name: project-checks
description: Verify this project's Go code before proposing a change.
---
Run `go vet ./...`, `go build ./...`, and `go test ./... -count=1`.
Report the checks actually run and any failures.
```

Project discovery searches `.opencode/skills`, `.claude/skills`, and `.agents/skills`, starting at the working directory and walking up to the Git root (or filesystem root when there is no Git root). User locations are `~/.config/opencode/skills`, `~/.claude/skills`, and `~/.agents/skills`. The first valid skill of a given name wins, so nearer project definitions take precedence. Invalid skills are skipped with a warning.

Names must match their directory and use lowercase letters, digits, and single hyphens, up to 64 characters. A description is required. Optional `license` and `metadata` fields are supported. Setting `metadata: {opencode/autoinvoke: "false"}` hides a skill from the advertised skill list; it does not prevent loading it by name.

This repository includes `go-verify` for development checks and `github-release` for explicitly requested releases. Skills supply instructions; they do not install additional tools or grant permissions.

## Project context

Use `contextPaths` in `.opencode.json` to include project instructions. Paths are resolved relative to the working directory. End a directory path with `/` to include its files recursively:

```json
{
  "contextPaths": ["AGENTS.md", "docs/project-context/"]
}
```

Context files follow configured path order, with directory contents traversed in lexical order. Repeated or overlapping file paths are included once. Symlink aliases are not resolved for deduplication.

## Permissions and tools

The coding agent can read and search files, edit and write files, apply patches, run shell commands, fetch web content, and use configured MCP tools. In interactive sessions, operations that can modify files or perform external actions ask for approval. Review each request before allowing it; approval can also be remembered for the current session.

The delegated search agent is read-only and can list, search, and view files without modifying them.

## MCP servers

Configure MCP servers in the `mcpServers` object. Stdio is the default transport:

```json
{
  "mcpServers": {
    "example": {
      "type": "stdio",
      "command": "path-to-server",
      "args": [],
      "env": []
    }
  }
}
```

For SSE, set `"type": "sse"`, provide a `url`, and optionally add request `headers`. Discovered tools follow the application's permission flow.

## Language servers

Configure language servers in the `lsp` object. Install each server separately and ensure its command is available to OpenCode.

```json
{
  "lsp": {
    "go": {
      "command": "gopls",
      "args": []
    }
  }
}
```

Connected language servers can provide diagnostics to the coding agent.

## Development

Run the repository verification checks with:

```sh
gofmt -l .
go vet ./...
go build ./...
go test ./... -count=1
```

`gofmt -l .` should produce no output. CI also verifies module checksums with `go mod verify` and runs build, vet, and tests on Ubuntu 24.04 using the Go version from `go.mod`.

The CLI entry point is in `cmd/`; providers, models, and tools are under `internal/llm/`; the terminal UI is under `internal/tui/`.

## Acknowledgments

This fork continues the original [opencode-ai/opencode](https://github.com/opencode-ai/opencode) project, which was archived in favor of [Crush](https://github.com/charmbracelet/crush). Thanks to the original author and contributors, to [@isaacphi](https://github.com/isaacphi) for the [mcp-language-server](https://github.com/isaacphi/mcp-language-server) project that underpins the LSP client, and to [@adamdottv](https://github.com/adamdottv) for the UI/UX direction.

## License

OpenCode is licensed under the MIT License. Original copyright: 2025 Kujtim Hoxha. Fork contributions: 2026 HenricHuntsman. The original copyright and permission notices are preserved in [LICENSE](LICENSE). Third-party dependencies and assets remain subject to their respective licences.
