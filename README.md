# ClaudeGo

A CLI-based AI coding agent written in Go, powered by LLMs via a streaming REPL interface.

## Features

- **Interactive REPL** — Line-editing with history (`liner`), ANSI-styled output, Ctrl+C interruption with conversation rollback
- **LLM Streaming** — OpenAI-compatible API client with streaming chat completions and tool calling
- **Plan Mode** — Complex tasks are decomposed into multi-step plans: create → execute → pause/resume → complete
- **Built-in Tools** — `bash` (with dangerous command detection) and `file_handler`
- **Conversation State** — Checkpoint/rollback preserves history across interrupts
- **Persistent Plans** — Plans saved as JSON to `~/.claudego/plans/`
- **Rotating Logs** — Logs stored in `~/.claudego/logs/` with 7-day retention

## Architecture

```
cmd/claudego/main.go     — REPL entry point, signal handling, command routing
internal/loop/agent.go    — Agent loop: streams LLM responses, executes tools
internal/plan/            — Plan creation, step-by-step execution, persistence
internal/tools/          — Tool registry, bash tool, file handler tool
internal/config/         — JSON config loader
pkg/llm/                 — LLM client wrapping OpenAI SDK
pkg/conversation/        — Conversation state with checkpoint/rollback
pkg/ui/                  — CLI styling, markdown rendering, spinner
pkg/logger/              — Singleton logger with file rotation
```

## Installation

```bash
git clone https://github.com/yizhigopher/learn-claude-with-go.git
cd learn-claude-with-go
go build -o claudego ./cmd/claudego
```

## Configuration

Create `~/.claudego/config.json`:

```json
{
  "api_key": "your-api-key",
  "base_url": "https://api.deepseek.com/v1",
  "model": "deepseek-chat"
}
```

- `api_key` — Your LLM provider API key
- `base_url` — OpenAI-compatible API endpoint
- `model` — Model name (e.g. `deepseek-chat`, `gpt-4o`)

## Usage

```bash
./claudego
```

### REPL Commands

| Command | Description |
|---------|-------------|
| `q` / `exit` | Quit the session |
| `/plan <goal>` | Force plan mode for a specific goal |

### Auto-detection

ClaudeGo automatically detects complex tasks (refactor, migrate, implement, build, etc.) and switches to plan mode.

### Plan Mode

When plan mode activates:
1. The agent analyzes your goal and creates a step-by-step plan
2. Steps are displayed and saved to `~/.claudego/plans/`
3. Each step executes in sequence with LLM + tool access
4. Press `Ctrl+C` to interrupt — conversation rolls back to the last checkpoint
5. Resume a paused plan with the saved plan ID

### Built-in Tools

**bash** — Execute shell commands
- Dangerous commands are blocked: `rm -rf /`, `sudo`, `shutdown`, fork bombs, piping remote scripts, etc.

**file_handler** — Read, write, and edit files

### Conversation Rollback

Press `Ctrl+C` during any LLM call to interrupt. The conversation state is rolled back to the checkpoint before the current query, preserving the integrity of your conversation history.

## Project Structure

```
claudego/
├── cmd/claudego/main.go       # Application entry point
├── internal/
│   ├── loop/agent.go          # Agent loop
│   ├── plan/                  # Plan mode (create, execute, persist)
│   ├── tools/                 # Tool registry + built-in tools
│   └── config/                # Configuration loader
├── pkg/
│   ├── llm/                   # LLM client
│   ├── conversation/          # Conversation state
│   ├── ui/                    # CLI output styling
│   └── logger/                # Logging
└── utils/                     # Utilities
```

## Dependencies

- [openai-go](https://github.com/openai/openai-go) — LLM API client
- [liner](https://github.com/peterh/liner) — Line editing for REPL
- [logrus](https://github.com/sirupsen/logrus) — Logging
- [go-playground/validator](https://github.com/go-playground/validator) — Input validation

## License

MIT
