# Lens

[![Build Status](https://img.shields.io/github/actions/workflow/status/vdyalex/lens-daemon/pipeline.yml?branch=main&style=flat-square)](https://github.com/vdyalex/lens-daemon/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vdyalex/lens-daemon?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/github/license/vdyalex/lens-daemon?style=flat-square)](LICENSE)

A MacOS daemon that captures your screen on demand via a global hotkey, extracts text using OCR, processes it through Claude AI (Anthropic), and routes the response to either a stealth teleprompter overlay or Telegram subscribers (configurable via `OUTPUT_METHOD`). All operations happen in-memory with zero disk writes for screenshots.

- [Lens](#lens)
  - [⚡ How it works](#-how-it-works)
  - [📋 Pre-requisites](#-pre-requisites)
  - [🔧 Installation](#-installation)
  - [⚙️ Configuration](#️-configuration)
    - [Required](#required)
    - [Optional](#optional)
    - [🎛️ Hotkey Configuration](#️-hotkey-configuration)
  - [▶️ Running](#️-running)
    - [CLI Commands](#cli-commands)
    - [Manually](#manually)
    - [Build and Test Commands](#build-and-test-commands)
  - [🔐 Permissions](#-permissions)
  - [📦 Dependencies](#-dependencies)
    - [🛠️ Tools](#️-tools)
  - [📄 License](#-license)

## ⚡ How it works

The daemon uses a **two-phase pipeline** for low-latency capture and parallel processing:

```
Phase 1 (Fast)      Phase 2 (Async)
Hotkey → Capture    ↓ (queue)    OCR → AI → Route (Teleprompter | Telegram)
                    ↓ (concurrent)
```

1. **Hotkey** — global keyboard listener via MacOS `CGEventTap`. Default: `RightShift` key. Customizable via `HOTKEY_TRIGGER_KEYNAME` environment variable
2. **Capture** — grabs the active window via AppleScript and CoreGraphics (direct Objective-C bridge via `src/bridges/core_graphics`, no external libraries). When the focused app is a recognised browser (Safari, Chrome, Firefox), the capture is automatically clipped to the page content area, excluding the browser toolbar. Hold `RightOption` to define explicit custom bounds (overrides automatic canvas detection). Customizable via `HOTKEY_BOUNDS_KEYNAME`
3. **Queue** — captured images are enqueued for Phase 2 analysis (queue capacity: 5 by default, configurable via `ANALYSE_QUEUE_CAPACITY`)
4. **OCR** — extracts text from the image using Apple Vision framework, entirely in-memory
5. **AI** — sends extracted text to Claude with a configurable system prompt (max 1024 response tokens). The system prompt and tool definition use ephemeral prompt caching (configurable TTL, default 1h) to reduce cost and latency on repeated calls. The response is a JSON object with `short` and `detailed` fields
6. **Route** — routes the AI response based on `OUTPUT_METHOD` (default: `teleprompter`, switchable at runtime via `lensd set output-method`):
   - **Teleprompter** — displays the short answer on a stealth overlay excluded from screen sharing (`NSWindowSharingNone`), invisible to Zoom, Mission Control, Dock, and Cmd+Tab. Toggle visibility with `RightCommand` (configurable via `HOTKEY_TOGGLE_KEYNAME`)
   - **Telegram** — broadcasts the detailed response to all subscribers, auto-chunking messages exceeding 4096 runes. Requires `TELEGRAM_BOT_TOKEN`

**Key benefit**: Rapid hotkey presses are captured immediately in Phase 1 while Phase 2 processes previous results concurrently. If analysis is slower than capture (typical case), results queue up and are processed in parallel without losing captures.

Captures happen **only** when you press the hotkey. No continuous polling or background recording.

For detailed documentation, see:

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — architecture, design decisions, and pipeline flow
- [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) — application requirements
- [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) — implementation details

## 📋 Pre-requisites

- **MacOS** (uses CoreGraphics CGEventTap, Vision framework, and AppleScript)
- **Go 1.25+** (with cgo support)
- **Anthropic API key** — get one at [console.anthropic.com](https://console.anthropic.com)
- **Telegram bot** (optional) — create one via [@BotFather](https://t.me/BotFather). Required when `OUTPUT_METHOD=telegram`

## 🔧 Installation

```bash
git clone https://github.com/vdyalex/lens-daemon.git
cd lens-daemon
make build
```

This produces a binary named `lensd` (default) in the `bin/` directory. To build with a custom binary name:

```bash
BINARY_NAME=myapp make build
```

The binary name is injected at compile time and controls the CLI command name and all daemon runtime file paths (`$TMPDIR/<binary>-<uid>.{pid,sock,log}`).

## ⚙️ Configuration

All configuration is done through environment variables. Copy `.env.example` to `.env` and fill in your values, or export them directly.

### Required

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Anthropic API key (e.g., `sk-ant-...`) |

### Optional

| Variable | Default | Description |
|---|---|---|
| `VISION_ACCURACY` | `accurate` | Vision accuracy level: `accurate` (slower, higher quality) or `fast` (faster, lower quality) |
| `ANTHROPIC_MAX_RESPONSE_TOKENS` | `1024` | Maximum tokens per Anthropic API response |
| `ANTHROPIC_CACHE_TTL` | `1h` | Prompt caching TTL for system prompt and tool definitions (`5m` or `1h`) |
| `TELEGRAM_MESSAGE_CHUNK_SIZE` | `4096` | Maximum runes per Telegram message (auto-splits longer responses) |
| `TELEGRAM_MAX_RETRIES` | `1` | Retry attempts on Telegram rate limit (HTTP 429) |
| `TELEGRAM_LONG_POLL_TIMEOUT` | `30s` | Server-side long-poll timeout for Telegram updates |
| `TELEGRAM_POLLER_TIMEOUT` | `35s` | Client-side timeout for Telegram poller (5s longer than server timeout for network jitter) |
| `TELEGRAM_HTTP_CLIENT_TIMEOUT` | `0` (disabled) | Per-request HTTP timeout (disabled to avoid racing with long-poll; `TELEGRAM_POLLER_TIMEOUT` is the correct bound) |
| `TIMEOUT_PIPELINE_OVERALL` | `5m` | Total deadline for a complete capture-to-broadcast cycle |
| `TIMEOUT_FOREGROUND_WINDOW` | `5s` | Deadline for detecting the active window via AppleScript |
| `TIMEOUT_CAPTURE` | `30s` | Deadline for taking a screenshot |
| `TIMEOUT_OCR_EXTRACT` | `30s` | Deadline for OCR text extraction via Vision framework |
| `TIMEOUT_AI_PROCESS` | `60s` | Deadline for Claude AI API call and response |
| `TELEGRAM_BROADCAST_TIMEOUT` | `30s` | Deadline for broadcasting to all subscribers |
| `TIMEOUT_CAPTURE_PHASE` | `40s` | Total deadline for Phase 1 (window detection + screenshot) |
| `TIMEOUT_ANALYSE_PHASE` | `5m` | Total deadline for Phase 2 (OCR + AI + broadcast) |
| `EVENT_TAP_POLL_INTERVAL` | `500ms` | CFRunLoop polling interval for keyboard event detection |
| `ANALYSE_QUEUE_CAPACITY` | `5` | Buffer size for Phase 2 analyse queue (captures are queued when analyse is slower than capture) |
| `LOG_LEVEL` | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `VISION_LANG` | *(auto-detect)* | Vision language hint (BCP 47 code, e.g., `en-US`, `fr-FR`). Empty enables auto-detection |
| `ANTHROPIC_MODEL` | `claude-sonnet-4-6` | Anthropic model ID to use for AI processing |
| `ANTHROPIC_SYSTEM_PROMPT` | *(built-in)* | System prompt sent to Anthropic with each request |
| `OUTPUT_METHOD` | `teleprompter` | Output method: `telegram` or `teleprompter`. Switchable at runtime via `lensd set output-method <value>` |
| `TELEGRAM_BOT_TOKEN` | *(none)* | Telegram bot token from @BotFather. Required when `OUTPUT_METHOD=telegram` |
| `TELEGRAM_SUBSCRIBER_STORE_PATH` | `tmp/subscribers` | File path for the subscriber list (persists users who sent `/start`) |
| `HOTKEY_TRIGGER_KEYNAME` | `RightShift` | Hotkey to trigger capture (see [Hotkey Configuration](#-hotkey-configuration) below) |
| `HOTKEY_BOUNDS_KEYNAME` | `RightOption` | Hotkey to define custom capture bounds (see [Hotkey Configuration](#-hotkey-configuration) below) |
| `HOTKEY_TOGGLE_KEYNAME` | `RightCommand` | Hotkey to toggle teleprompter overlay visibility (see [Hotkey Configuration](#️-hotkey-configuration) below) |
| `TELEPROMPTER_FONT_FAMILY` | *(system font)* | Font family name (e.g., `Menlo`, `Helvetica Neue`). Empty uses the system font |
| `TELEPROMPTER_FONT_WEIGHT` | `ultralight` | Font weight: `ultralight`, `thin`, `light`, `regular`, `medium`, `semibold`, `bold`, `heavy`, `black` |
| `TELEPROMPTER_FONT_SIZE` | `14.0` | Font size in points |
| `TELEPROMPTER_OPACITY` | `0.05` | Text opacity from `0.0` (invisible) to `1.0` (fully opaque). Adjustable at runtime with bounds hotkey + `−`/`+` keys (±0.01 per step). Press `0` to reset to default |
| `TELEPROMPTER_VISIBLE` | `false` | Initial visibility on startup (`true` to show immediately) |
| `TELEPROMPTER_ALIGNMENT` | `dynamic` | Text alignment: `left`, `center`, `right`, or `dynamic` (adapts to grid column position) |
| `TELEPROMPTER_GRID_STEP` | `0.005` | Percentage increment per arrow-key press (0.0–1.0) |
| `TELEPROMPTER_GRID_INITIAL_COL` | `0.5` | Initial horizontal grid position (0.0 = left, 1.0 = right) |
| `TELEPROMPTER_GRID_INITIAL_ROW` | `0.5` | Initial vertical grid position (0.0 = top, 1.0 = bottom) |
| `TELEPROMPTER_GRID_MOVE_DEBOUNCE_DURATION` | `300ms` | Idle delay before teleprompter repositions after arrow presses |
| `TELEPROMPTER_WINDOW_MONITOR_INTERVAL` | `200ms` | How often to check if the captured window moved/resized |
| `TELEPROMPTER_WINDOW_STABILIZE_DELAY` | `500ms` | How long window must stay still before teleprompter restores |
| `TELEPROMPTER_ADAPTIVE_COLOR` | `true` | Per-pixel adaptive text color: captures the background behind the overlay, inverts it, and uses the result as the text color so each pixel contrasts with whatever is beneath it. Sampling is event-gated (runs once on each text update, co-timed with the OCR hotkey) rather than periodic |
| `TELEPROMPTER_FADE_DURATION` | `0.8` | Fade animation duration in seconds for show, hide, and text updates. Set to `0` to disable |

Claude responds with a structured JSON tool call containing `short` (concise answer for the teleprompter) and `detailed` (answer + reason for Telegram).

### 🎛️ Hotkey Configuration

Configure which keys trigger captures, bounds selection, and teleprompter visibility via `HOTKEY_TRIGGER_KEYNAME`, `HOTKEY_BOUNDS_KEYNAME`, and `HOTKEY_TOGGLE_KEYNAME`.

**Supported key names:** `LeftShift`, `RightShift`, `LeftControl`, `RightControl`, `LeftCommand`, `RightCommand`, `LeftOption`, `RightOption`, `Fn`

**Examples:**

```bash
# Use Left Command for capture, Right Control for bounds
export HOTKEY_TRIGGER_KEYNAME="LeftCommand"
export HOTKEY_BOUNDS_KEYNAME="RightControl"

# Use Fn key for capture
export HOTKEY_TRIGGER_KEYNAME="Fn"
```

Invalid key names will be rejected at startup with a clear error listing all supported options.

## ▶️ Running

### CLI Commands

The binary provides a set of subcommands for starting, stopping, and managing the daemon. The examples below use the default binary name `lensd`:

| Command | Purpose |
|---------|---------|
| `lensd daemon` | Run the pipeline with IPC server (called by `start` command); accepts config flags |
| `lensd start` | Start daemon in background (re-execs `lensd daemon` detached); accepts config flags |
| `lensd stop` | Stop the running daemon |
| `lensd status` | Check daemon status (PID, uptime, subscribers, output method, last window) |
| `lensd logs` | Stream daemon logs to stdout (with level-based colorization) |
| `lensd restart` | Stop and start the daemon; accepts config flags |
| `lensd set output-method <telegram\|teleprompter>` | Switch output method at runtime |

All start/daemon/restart commands accept optional flags to override configuration:

```bash
--api-key              Anthropic API key
--bot-token            Telegram bot token
--model                Anthropic model name
--system-prompt        AI system prompt
--max-tokens           Max response tokens
--log-level            Log level (debug/info/warn/error)
--store-path           Telegram subscriber store file path
--output-method        Output method: telegram or teleprompter
```

### Manually

```bash
# Export required variables
export ANTHROPIC_API_KEY="sk-ant-..."
export TELEGRAM_BOT_TOKEN="123456:ABC..."

# Start daemon in background
./bin/lensd start

# Check status
./bin/lensd status

# Stream logs
./bin/lensd logs

# Stop daemon
./bin/lensd stop

# Or build and run in foreground
make daemon
```

Once running:

1. **Toggle teleprompter**: Press the configured teleprompter hotkey (default: `RightCommand`) to show/hide the overlay (fades in/out)
2. **Subscribe** (optional): Send `/start` to your Telegram bot to begin receiving detailed responses
3. **Capture**: Press the configured trigger hotkey (default: `RightShift`) at any time to trigger a capture
   - Multiple rapid captures are queued and processed concurrently
   - If the queue is full (5 items by default), the newest capture is dropped with a warning log
   - The teleprompter text is cleared while the new result is being processed
4. **Custom bounds** (optional): Hold the configured bounds hotkey (default: `RightOption`), move your mouse to define a region, then release
5. **Reposition teleprompter** (optional): Hold the bounds hotkey (default: `RightOption`) and press arrow keys (Up/Down/Left/Right) to move the teleprompter by `TELEPROMPTER_GRID_STEP` (0.5%) per press. Position wraps circularly. Rapid presses debounce — the teleprompter fades out, waits for input to stop, then repositions and fades in
6. **Adjust text opacity** (optional): Hold the bounds hotkey (default: `RightOption`) and press `−` to decrease or `+` to increase text opacity by 0.01 per step (clamped to 0.0–1.0). Press hotkey + `0` to reset to the configured default
7. **Adjust font size** (optional): Hold the bounds hotkey (default: `RightOption`) and press `,` to decrease or `.` to increase font size by 0.5pt per step (clamped to 5–48pt)
8. The daemon captures the screen, enqueues for analysis, processes OCR, sends the text to Claude, then routes the response based on `OUTPUT_METHOD`: displays the short answer on the teleprompter, or broadcasts the detailed response to Telegram subscribers

Send `/stop` to the Telegram bot to unsubscribe. Run `./bin/lensd stop` to stop the daemon.

### Build and Test Commands

| Command | Purpose |
|---|---|
| `make build` | Compile the binary to `bin/` |
| `make test` | Run all unit tests |
| `make check` | Run all static checks and tests (format + validate + lint + vulnerabilities + test) |
| `make clean` | Remove build artifacts from `bin/` |
| `make format` | Format source files with gofmt |
| `make validate` | Run go vet static analysis |
| `make lint` | Run golangci-lint static analysis |
| `make vulnerabilities` | Run govulncheck vulnerability scanner |
| `make coverage` | Generate test coverage report (produces `coverage/coverage.html`) |
| `make generate` | Run go generate on all packages |
| `make tools` | Install analysis tools (golangci-lint, govulncheck, mockgen) |
| `make daemon` | Build and run in daemon mode (foreground) |
| `make develop` | Build, restart daemon, and stream logs (combines build + restart + logs) |
| `make start` | Build and start daemon in background |
| `make stop` | Stop the running daemon |
| `make status` | Check daemon status |
| `make restart` | Rebuild, stop, and start the daemon |
| `make logs` | Stream daemon logs to stdout |

## 🔐 Permissions

MacOS will prompt for two permissions on first run. Both must be granted for the daemon to function:

| Permission | Required For | Where to Grant |
|---|---|---|
| **Accessibility** | Global hotkey listener (`CGEventTap`) | System Settings → Privacy & Security → Accessibility |
| **Screen Recording** | Screen capture | System Settings → Privacy & Security → Screen Recording |

If Accessibility permission is not granted, the daemon will fail on startup with:

```
CGEventTapCreate failed -- grant Accessibility permission to this app
```

## 📦 Dependencies

| Dependency | Purpose |
|---|---|
| **Standard Library** | `sync` (WaitGroup for concurrent Phase 2 analysis), `context` (cancellation and timeouts), `image` (RGBA image buffers), `time` (deadlines and polling), `encoding/json` (IPC protocol) |
| [`anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) | Official Anthropic Go SDK for Claude AI API |
| [`joho/godotenv`](https://github.com/joho/godotenv) | Loads `.env` files into environment variables at startup |
| [`pterm/pterm`](https://github.com/pterm/pterm) | Terminal UI — colorized output for CLI commands (`start`, `stop`, `status`, `logs`, `restart`) |
| [`spf13/cobra`](https://github.com/spf13/cobra) | CLI command framework — powers all `lensd` subcommands |

**Built-in frameworks**: CoreGraphics (screen capture via CGo), Apple Vision framework (OCR via CGo), AppleScript (window detection). No external OCR libraries required.

### 🛠️ Tools

The following analysis and code generation tools are installed via `make tools` and invoked during development and CI:

| Tool | Purpose |
|---|---|
| `go vet` | Static analysis — detects suspicious constructs and common mistakes |
| `gofmt` | Formatter — enforces standard Go code formatting |
| [`golangci-lint`](https://github.com/golangci/golangci-lint) | Linter aggregator — runs multiple linters in one pass |
| [`govulncheck`](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) | Vulnerability scanner — checks dependencies against the Go vulnerability database |
| [`mockgen`](https://pkg.go.dev/go.uber.org/mock/mockgen) | Mock generator — generates interface mocks for unit tests (via `go.uber.org/mock`) |

---

## 📄 License

This project is licensed under the **GNU General Public License v3.0** — see the [LICENSE](LICENSE) file for details.

You are free to use, modify, and distribute this software under the terms of the GPLv3 license. Any derivative works must also be licensed under GPLv3.
