# Implementation

## Overview

- **Platform**: MacOS only. Uses MacOS-specific APIs (`CGEventTap`, `CFRunLoop`, `CoreGraphics`, `CoreFoundation`, `Vision framework`) via cgo and AppleScript via `osascript`.
- **In-memory processing**: No screenshots, intermediate images, or temporary files are written to disk at any point in the pipeline.
- **Daemon operation**: Runs as a background CLI daemon. Does not appear in the Dock or Cmd-Tab (pure CLI process with no GUI elements).
- **Event-driven**: Idle until the hotkey is pressed. No polling, no timers, no periodic screen checks.
- **Single-trigger pipeline**: Each hotkey press executes a full sequential pipeline (capture -> OCR -> AI -> Telegram). The pipeline does not overlap; if a capture is already in progress, additional triggers are dropped.
- **Language**: Go 1.24+ with cgo (for `CoreGraphics`/`CoreFoundation` bindings and Vision framework).
- **Settings**: All settings via environment variables. No config files, no CLI flags.
- **Logging**: Structured log output to stderr using Go's slog TextHandler (time, level, message, and key-value fields). Log verbosity is controlled by `LOG_LEVEL`.
- **External dependencies**: No external OCR dependencies required (uses built-in Apple Vision framework).
- **Security permissions**: Requires MacOS Accessibility and Screen Recording permissions granted to the terminal or binary.

## Environment Variable Management

### Loading Mechanism

The application uses `godotenv` (safe variant) to load environment variables from a `.env` file. The loading order and precedence is:

1. **Shell environment** (highest priority) — variables already exported before process start
2. **`.env` file** — loaded via `godotenv.Load()`, only fills in missing variables
3. **Code defaults** — hardcoded fallbacks in the config struct

This design ensures that:

- Variables injected externally (shell, CI, LaunchAgent plist) always take precedence
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

**Service Installation (`make service-install`):**

```
.env file → service-install.sh (reads and validates) → embedded into MacOS LaunchAgent plist
         → LaunchAgent startup → plist EnvironmentVariables (set in process environment)
         → godotenv.Load() (only supplements missing vars) → config.Load()
```

**CI/Container:**

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
- `CLAUDE_MODEL` — Claude model to use (default: "claude-sonnet-4-6")
- `SYSTEM_PROMPT` — Custom system prompt for Claude (default: generic questionnaire assistant)
- `HOTKEY_TRIGGER_KEYNAME` — Trigger hotkey name (default: "RightShift")
- `HOTKEY_BOUNDS_KEYNAME` — Bounds hotkey name (default: "RightOption")
- Various timeout settings for pipeline stages and Telegram communication

See the `Config` struct in `src/utils/config/config.go` for a complete list with defaults.
