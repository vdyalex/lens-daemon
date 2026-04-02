# Implementation

## Overview

- **Platform**: MacOS only. Uses MacOS-specific APIs (`CGEventTap`, `CFRunLoop`, `CoreGraphics`, `CoreFoundation`, `Vision framework`) via cgo and AppleScript via `osascript`.
- **In-memory processing**: No screenshots, intermediate images, or temporary files are written to disk at any point in the pipeline.
- **CLI and daemon operation**: Single binary with Cobra subcommands (daemon, start, stop, status, logs, restart). The `start` command daemonizes via process re-exec with `syscall.SysProcAttr{Setsid: true}`. Does not appear in Dock or Cmd-Tab (pure CLI process).
- **Event-driven**: Idle until hotkey pressed or IPC command received. No polling, no timers, no periodic screen checks.
- **Single-trigger pipeline**: Each hotkey press executes a full sequential pipeline (capture -> OCR -> AI -> Telegram). The pipeline does not overlap; if a capture is already in progress, additional triggers are dropped.
- **Language**: Go 1.24+ with cgo (for `CoreGraphics`/`CoreFoundation` bindings and Vision framework).
- **Configuration**: All settings via environment variables. CLI flags on start/daemon/restart commands are forwarded as env vars to child processes. No config files.
- **IPC communication**: Unix domain socket with length-prefixed JSON for inter-process communication. Enables remote status checks, log streaming, and graceful shutdown.
- **Logging**: Structured log output using Go's slog (time, level, message, and key-value fields). Log verbosity controlled by `LOG_LEVEL`. Daemon output goes to stderr (for `make run daemon`) and is replicated to IPC log broker for `<binary> logs` streaming.
- **External dependencies**: No external OCR dependencies required (uses built-in Apple Vision framework).
- **Security permissions**: Requires MacOS Accessibility and Screen Recording permissions granted to the terminal or binary.

## Environment Variable Management

### Loading Mechanism

The application uses `godotenv` (safe variant) to load environment variables from a `.env` file. The loading order and precedence is:

1. **Shell environment** (highest priority) — variables already exported before process start
2. **`.env` file** — loaded via `godotenv.Load()`, only fills in missing variables
3. **Code defaults** — hardcoded fallbacks in the config struct

This design ensures that:

- Variables injected externally (shell, CI) always take precedence
- The `.env` file supplements but never overrides externally-set variables
- The application works in all environments: development, service mode, and CI/CD

### Configuration Files

- **`.env`** — Runtime configuration (development and local service installation). Contains required keys (`ANTHROPIC_API_KEY`, `TELEGRAM_BOT_TOKEN`) and optional overrides. Git-ignored to prevent committing secrets.
- **`.env.example`** — Template with all available variables and explanations. Committed to version control for reference.

### Deployment Scenarios

**Development (`make run`):**

```
.env file → Makefile (sources and exports) → godotenv.Load() → config.Load()
```

**CI:**

```
Environment variables injected before process start → godotenv.Load() (no-op if .env absent)
         → config.Load()
```

### Required Environment Variables

- `ANTHROPIC_API_KEY` — Anthropic API key (required, no default)
- `TELEGRAM_BOT_TOKEN` — Telegram bot token (required, no default)

### Optional Environment Variables

See `.env.example` for the complete list with descriptions and defaults. Common variables:

- `LOG_LEVEL` — Minimum log level: "debug", "info", "warn", "error" (default: "info")
- `ANTHROPIC_MODEL` — Anthropic model to use (default: "claude-sonnet-4-6")
- `ANTHROPIC_SYSTEM_PROMPT` — Custom system prompt for Claude (default: generic questionnaire assistant)
- `HOTKEY_TRIGGER_KEYNAME` — Trigger hotkey name (default: "RightShift")
- `HOTKEY_BOUNDS_KEYNAME` — Bounds hotkey name (default: "RightOption")
- Various timeout settings for pipeline stages and Telegram communication

See the `Config` struct in `src/utils/config/config.go` for a complete list with defaults.
