# OpenCode

A terminal-based AI coding assistant maintained in the HenricHuntsman fork. Use the interactive terminal UI to work with a project, or run a single prompt from a script.

## Capabilities

- Chat with supported models from Anthropic, OpenAI, Google Gemini, GitHub Copilot, Groq, OpenRouter, xAI, Azure OpenAI, AWS Bedrock, Google Cloud Vertex AI, and compatible local endpoints.
- Search and inspect source files, edit files, apply patches, run shell commands, and request language-server diagnostics.
- Connect external tools through Model Context Protocol (MCP) over stdio or server-sent events (SSE).
- Keep conversations and sessions in local SQLite storage.
- Run one-shot prompts with text or JSON output.
- Review permission requests before file writes, shell commands, and external tool execution.

## Requirements

- Go 1.24 or later to build from source.
- A terminal and credentials for at least one supported model provider.
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

For Ollama, each model's real context length is read from the server; other servers fall back to 4096 tokens. Discovery gives up after about 1.5 seconds per probe, so an unreachable endpoint delays startup by a few seconds and does not block it. Pick a model with `agents.coder.model` in your config, for example `local.qwen2.5-coder:7b`. Models need tool-calling support to work as a coding agent.

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

Run the test suite and build the application with:

```sh
go test ./...
go build ./...
```

The CLI entry point is in `cmd/`; providers, models, and tools are under `internal/llm/`; the terminal UI is under `internal/tui/`.

## Acknowledgments

This fork continues the original [opencode-ai/opencode](https://github.com/opencode-ai/opencode) project, which was archived in favor of [Crush](https://github.com/charmbracelet/crush). Thanks to the original author and contributors, to [@isaacphi](https://github.com/isaacphi) for the [mcp-language-server](https://github.com/isaacphi/mcp-language-server) project that underpins the LSP client, and to [@adamdottv](https://github.com/adamdottv) for the UI/UX direction.

## License

OpenCode is licensed under the MIT License. See [LICENSE](LICENSE).
