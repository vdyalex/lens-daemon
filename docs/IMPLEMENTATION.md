# Implementation

## Overview

- **Platform**: MacOS only. Uses MacOS-specific APIs (`CGEventTap`, `CFRunLoop`, `CoreGraphics`, `CoreFoundation`, `Vision framework`) via cgo and AppleScript via `osascript`.
- **In-memory processing**: No screenshots, intermediate images, or temporary files are written to disk at any point in the pipeline.
- **CLI and daemon operation**: Single binary with Cobra subcommands (daemon, start, stop, status, logs, restart). The `start` command daemonizes via process re-exec with `syscall.SysProcAttr{Setsid: true}`. Does not appear in Dock or Cmd-Tab (pure CLI process).
- **Event-driven**: Idle until hotkey pressed or IPC command received. No polling, no timers, no periodic screen checks.
- **Two-phase pipeline**: Each hotkey press triggers a capture (Phase 1) that enqueues results for concurrent analysis (Phase 2: OCR -> AI -> Teleprompter + Telegram). Multiple captures and analyses run concurrently; triggers are only dropped when the analyse queue is full.
- **Stealth overlay**: A macOS overlay window (teleprompter) displays the short answer. Excluded from screen sharing via `NSWindowSharingNone`. The AppKit run loop runs on the main OS thread; all daemon logic runs in background goroutines.
- **Language**: Go 1.25+ with cgo (for `CoreGraphics`/`CoreFoundation` bindings and Vision framework).
- **Configuration**: All settings via environment variables. CLI flags on start/daemon/restart commands are forwarded as env vars to child processes. No config files.
- **IPC communication**: Unix domain socket with length-prefixed JSON for inter-process communication. Enables remote status checks, log streaming, and graceful shutdown.
- **Logging**: Structured log output using Go's slog (time, level, message, and key-value fields). Log verbosity controlled by `LOG_LEVEL`. Daemon output goes to stderr (for `make daemon`) and is replicated to IPC log broker for `<binary> logs` streaming.
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

**Development (`make daemon`):**

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

### Optional Environment Variables

See `.env.example` for the complete list with descriptions and defaults. Common variables:

- `TELEGRAM_BOT_TOKEN` — Telegram bot token. When absent, Telegram is disabled (teleprompter-only mode)
- `LOG_LEVEL` — Minimum log level: "debug", "info", "warn", "error" (default: "info")
- `ANTHROPIC_MODEL` — Anthropic model to use (default: "claude-sonnet-4-6")
- `ANTHROPIC_SYSTEM_PROMPT` — Custom system prompt for Claude (default: generic questionnaire assistant)
- `HOTKEY_TRIGGER_KEYNAME` — Trigger hotkey name (default: "RightShift")
- `HOTKEY_BOUNDS_KEYNAME` — Bounds hotkey name (default: "RightOption")
- `HOTKEY_TOGGLE_KEYNAME` — Teleprompter toggle hotkey name (default: "RightCommand")
- `TELEPROMPTER_FONT_FAMILY` — Font family name (default: system font)
- `TELEPROMPTER_FONT_WEIGHT` — Font weight: ultralight, thin, light, regular, medium, semibold, bold, heavy, black (default: "ultralight")
- `TELEPROMPTER_FONT_SIZE` — Font size in points (default: 12.0)
- `TELEPROMPTER_OPACITY` — Text opacity 0.0-1.0 (default: 0.3)
- `TELEPROMPTER_VISIBLE` — Initial visibility on startup (default: false)
- `TELEPROMPTER_POSITION` — Window alignment: left, center, right (default: "right")
- Various timeout settings for pipeline stages and Telegram communication

See the `Config` struct in `src/utils/config/config.go` for a complete list with defaults.
