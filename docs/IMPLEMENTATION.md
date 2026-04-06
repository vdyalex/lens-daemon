# Implementation

## Overview

- **Platform**: MacOS only. Uses MacOS-specific APIs (`CGEventTap`, `CFRunLoop`, `CoreGraphics`, `CoreFoundation`, `Vision framework`) via cgo and AppleScript via `osascript`.
- **In-memory processing**: No screenshots, intermediate images, or temporary files are written to disk at any point in the pipeline.
- **Browser content-area capture**: The full window is always captured first. When the focused application is a recognised browser (Safari, Chrome, Firefox), the image is cropped in Go to the content-area rectangle (derived by `browser.CanvasBounds` subtracting the browser toolbar height). This avoids `CGDisplayCreateImageForRect` coordinate-offset issues and gives Vision OCR a focused image. Canvas bounds are also used independently for teleprompter grid positioning; custom capture bounds do not affect the grid — the teleprompter always distributes over the canvas or raw window. Falls back to the full window for unrecognised apps.
- **Capture isolation**: The teleprompter overlay is hidden synchronously (`orderOut`, `waitUntilDone:YES`) before each screenshot and restored after, ensuring it does not appear in the captured image despite `CGDisplayCreateImageForRect` capturing the display framebuffer.
- **CLI and daemon operation**: Single binary with Cobra subcommands (daemon, start, stop, status, logs, restart, set). The `start` command daemonizes via process re-exec with `syscall.SysProcAttr{Setsid: true}`. Does not appear in Dock or Cmd-Tab (pure CLI process).
- **Event-driven**: Idle until hotkey pressed or IPC command received. The only background polling is the window monitor (`CGWindowListCopyWindowInfo` at 200ms intervals, keyed by `CGWindowID`) which is a pure metadata query — no screen capture, no privacy indicator. Tracking by window ID rather than app PID scopes monitoring to the exact captured window; switching apps mid-session does not cause spurious overlay fades.
- **Two-phase pipeline**: Each hotkey press triggers a capture (Phase 1) that enqueues results for concurrent analysis (Phase 2: OCR -> AI -> output routing based on `OUTPUT_METHOD`). Multiple captures and analyses run concurrently; triggers are only dropped when the analyse queue is full.
- **Stealth overlay**: A macOS overlay window (teleprompter) displays the short answer when the AI response is `deterministic=true`. Uncertain or hedged responses are suppressed from the overlay to prevent obviously AI-generated content from appearing on screen. Excluded from screen sharing via `NSWindowSharingNone`. The AppKit run loop runs on the main OS thread; all daemon logic runs in background goroutines.
- **Language**: Go 1.25+ with cgo (for `CoreGraphics`/`CoreFoundation` bindings and Vision framework).
- **Configuration**: All settings via environment variables. CLI flags on start/daemon/restart commands are forwarded as env vars to child processes. No config files.
- **IPC communication**: Unix domain socket with length-prefixed JSON for inter-process communication. Enables remote status checks, log streaming, runtime setting changes (`set` command), and graceful shutdown.
- **Logging**: Structured log output using Go's slog (time, level, message, and key-value fields). Log verbosity controlled by `LOG_LEVEL`. Daemon output goes to stderr (for `make daemon`) and is replicated to IPC log broker for `<binary> logs` streaming. At `debug` level, end-to-end latency from hotkey trigger to committed display (teleprompter) or broadcast (Telegram) is logged as `trigger to display latency` and `trigger to broadcast latency` respectively.
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

- **`.env`** — Runtime configuration (development and local service installation). Contains the required key (`ANTHROPIC_API_KEY`) and optional overrides (including `TELEGRAM_BOT_TOKEN` when using telegram output). Git-ignored to prevent committing secrets.
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

- `OUTPUT_METHOD` — Output method: "telegram" or "teleprompter" (default: "teleprompter"). Switchable at runtime via `lensd set output-method`
- `TELEGRAM_BOT_TOKEN` — Telegram bot token. Required when `OUTPUT_METHOD=telegram`
- `LOG_LEVEL` — Minimum log level: "debug", "info", "warn", "error" (default: "info")
- `ANTHROPIC_MODEL` — Anthropic model to use (default: "claude-sonnet-4-6")
- `ANTHROPIC_SYSTEM_PROMPT` — Custom system prompt for Claude (default: generic questionnaire assistant)
- `ANTHROPIC_CACHE_TTL` — Prompt caching TTL for system prompt and tool definitions: "5m" or "1h" (default: "1h")
- `HOTKEY_TRIGGER_KEYNAME` — Trigger hotkey name (default: "RightShift")
- `HOTKEY_BOUNDS_KEYNAME` — Bounds hotkey name (default: "RightOption")
- `HOTKEY_TOGGLE_KEYNAME` — Teleprompter toggle hotkey name (default: "RightCommand")
- `TELEPROMPTER_FONT_FAMILY` — Font family name (default: system font)
- `TELEPROMPTER_FONT_WEIGHT` — Font weight: ultralight, thin, light, regular, medium, semibold, bold, heavy, black (default: "ultralight")
- `TELEPROMPTER_FONT_SIZE` — Font size in points (default: 14.0)
- `TELEPROMPTER_OPACITY` — Text opacity 0.0-1.0 (default: 0.05). Adjustable at runtime via bounds hotkey + minus/plus keys (±0.01 per step). Press bounds hotkey + 0 to reset to default. Font size is also adjustable at runtime via bounds hotkey + comma/period keys (±0.5pt per step, clamped to 5–48pt)
- `TELEPROMPTER_VISIBLE` — Initial visibility on startup (default: false)
- `TELEPROMPTER_ALIGNMENT` — Text alignment: left, center, right, dynamic (default: "dynamic"). Dynamic adapts based on grid column position
- `TELEPROMPTER_ADAPTIVE_COLOR` — Per-pixel adaptive text color via background inversion (default: true)
- `TELEPROMPTER_FADE_DURATION` — Fade animation duration in seconds for show/hide/text updates (default: 0.8)
- `TELEPROMPTER_GRID_STEP` — Percentage increment per arrow-key press, 0.0–1.0 (default: 0.005)
- `TELEPROMPTER_GRID_INITIAL_COL` — Initial horizontal position, 0.0–1.0 (default: 0.5)
- `TELEPROMPTER_GRID_INITIAL_ROW` — Initial vertical position, 0.0–1.0 (default: 0.5)
- `TELEPROMPTER_GRID_MOVE_DEBOUNCE_DURATION` — Idle delay before snap commit (default: 300ms)
- `TELEPROMPTER_WINDOW_MONITOR_INTERVAL` — Window-bounds poll interval (default: 200ms)
- `TELEPROMPTER_WINDOW_STABILIZE_DELAY` — Stable-window delay before restoring overlay (default: 500ms)
- Various timeout settings for pipeline stages and Telegram communication

See the `Config` struct in `src/utils/config/config.go` for a complete list with defaults.
