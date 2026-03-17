# Lens

A MacOS daemon that captures your screen on demand via a global hotkey, extracts text using OCR, processes it through Claude AI (Anthropic), and broadcasts the response to Telegram subscribers. All operations happen in-memory with zero disk writes for screenshots.

## ⚡ How it works

```
Hotkey (configurable) → Screen Capture → OCR → Claude AI → Telegram
```

1. **Hotkey** — global keyboard listener via MacOS `CGEventTap`. Default: `RightShift` key. Customizable via `HOTKEY_TRIGGER_KEYNAME` environment variable
2. **Capture** — grabs the entire active window via AppleScript and CoreGraphics (direct Objective-C bridge via `src/bridges/core_graphics`, no external libraries). Default: hold `RightOption` key to define custom bounds. Customizable via `HOTKEY_BOUNDS_KEYNAME`
3. **OCR** — extracts text from the image using Apple Vision framework, entirely in-memory
4. **AI** — sends extracted text to Claude with a configurable system prompt (max 1024 response tokens)
5. **Notify** — broadcasts Claude's response to all Telegram subscribers, auto-chunking messages exceeding 4096 runes

Captures happen **only** when you press the hotkey. No continuous polling or background recording.

For detailed documentation, see:

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — architecture, design decisions, and pipeline flow
- [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) — application requirements
- [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md) — implementation details

## 📋 Pre-requisites

- **MacOS** (uses CoreGraphics CGEventTap, Vision framework, and AppleScript)
- **Go 1.24+** (with cgo support)
- **Anthropic API key** — get one at [console.anthropic.com](https://console.anthropic.com)
- **Telegram bot** — create one via [@BotFather](https://t.me/BotFather)

## 🔧 Installation

```bash
git clone https://github.com/vdyalex/lens-daemon.git
cd lens-daemon
make build
```

This produces the `lensd` binary in the `bin` directory.

## ⚙️ Configuration

All configuration is done through environment variables. Copy `.env.example` to `.env` and fill in your values, or export them directly.

### Required

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Anthropic API key (e.g., `sk-ant-...`) |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token from @BotFather (e.g., `123456:ABC-DEF...`) |

### Optional

| Variable | Default | Description |
|---|---|---|
| `SUBSCRIBER_STORE_PATH` | `tmp/subscribers`* | File path for the subscriber list (persists users who sent `/start`) |
| `VISION_ACCURACY` | `accurate` | Vision accuracy level: `accurate` (slower, higher quality) or `fast` (faster, lower quality) |
| `ANTHROPIC_MAX_RESPONSE_TOKENS` | `1024` | Maximum tokens per Anthropic API response |
| `TELEGRAM_MESSAGE_CHUNK_SIZE` | `4096` | Maximum runes per Telegram message (auto-splits longer responses) |
| `TELEGRAM_MAX_RETRIES` | `1` | Retry attempts on Telegram rate limit (HTTP 429) |
| `TELEGRAM_LONG_POLL_TIMEOUT` | `30s` | Server-side long-poll timeout for Telegram updates |
| `TELEGRAM_POLLER_TIMEOUT` | `35s` | Client-side timeout for Telegram poller (5s longer than server timeout for network jitter) |
| `TELEGRAM_HTTP_CLIENT_TIMEOUT` | `30s` | Per-request HTTP timeout for Telegram API calls |
| `TIMEOUT_PIPELINE_OVERALL` | `5m` | Total deadline for a complete capture-to-broadcast cycle |
| `TIMEOUT_FOREGROUND_WINDOW` | `5s` | Deadline for detecting the active window via AppleScript |
| `TIMEOUT_CAPTURE` | `30s` | Deadline for taking a screenshot |
| `TIMEOUT_OCR_EXTRACT` | `30s` | Deadline for OCR text extraction via Vision framework |
| `TIMEOUT_AI_PROCESS` | `60s` | Deadline for Claude AI API call and response |
| `TELEGRAM_BROADCAST_TIMEOUT` | `30s` | Deadline for broadcasting to all subscribers |
| `EVENT_TAP_POLL_INTERVAL` | `500ms` | CFRunLoop polling interval for keyboard event detection |
| `WORKER_QUEUE_CAPACITY` | `1` | Buffer size for capture queue (only 1 concurrent capture allowed; additional triggers are dropped) |
| `LOG_LEVEL` | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `VISION_LANG` | `en-US` | Vision language (BCP 47 code, e.g., `en-US`, `fr-FR`, `de-DE`, `zh-Hans`, `ja`, `ko`) |
| `ANTHROPIC_MODEL` | `claude-sonnet-4-6` | Anthropic model ID to use for AI processing |
| `ANTHROPIC_SYSTEM_PROMPT` | *(built-in)* | System prompt sent to Anthropic with each request |
| `HOTKEY_TRIGGER_KEYNAME` | `RightShift` | Hotkey to trigger capture (see [Hotkey Configuration](#-hotkey-configuration) below) |
| `HOTKEY_BOUNDS_KEYNAME` | `RightOption` | Hotkey to define custom capture bounds (see [Hotkey Configuration](#-hotkey-configuration) below) |

**Note:** `*` When using `make service-install`, the `SUBSCRIBER_STORE_PATH` defaults to `$HOME/Library/Application Support/lensd/subscribers` instead of `tmp/subscribers`.

The built-in default system prompt is:

> You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency.

### 🎛️ Hotkey Configuration

Configure which keys trigger captures and bounds selection via `HOTKEY_TRIGGER_KEYNAME` and `HOTKEY_BOUNDS_KEYNAME`.

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

### Manually

```bash
# Export required variables
export ANTHROPIC_API_KEY="sk-ant-..."
export TELEGRAM_BOT_TOKEN="123456:ABC..."

# Run in foreground
./bin/lensd

# Or build and run
make run
```

Once running:

1. **Subscribe**: Send `/start` to your Telegram bot to begin receiving responses
2. **Capture**: Press the configured trigger hotkey (default: `RightShift`) at any time to trigger a capture
3. **Custom bounds** (optional): Hold the configured bounds hotkey (default: `RightOption`), move your mouse to define a region, then release
4. The daemon captures the screen, runs OCR, sends the text to Claude, and broadcasts the response to all subscribers

Send `/stop` to the Telegram bot to unsubscribe. Press `Ctrl+C` to stop the daemon.

### As a Background Service

To run the daemon continuously as a background service that starts on login:

```bash
# Set up .env with your credentials
cp .env.example .env
# Edit .env and fill in your API keys

# Install and start the service
make service-install
```

This builds the binary to `$GOBIN` (`${GOPATH:-$HOME/go}/bin/lensd`), not the project root, and installs it as a MacOS LaunchAgent.

The service will:

- Start automatically on login
- Restart automatically if it crashes
- Run in the background with all hotkey functionality
- Log output to `~/Library/Logs/lens/stdout.log` and `~/Library/Logs/lens/stderr.log`

**Service management:**

| Command | Purpose |
|---|---|
| `make service-start` | Start the service (if already installed) |
| `make service-stop` | Stop the running service |
| `make service-logs` | View real-time service logs |
| `make service-uninstall` | Uninstall and remove the service |

The service is managed as a MacOS LaunchAgent (`com.vdyalex.lensd`). The following environment variables from your `.env` file are embedded into the LaunchAgent plist at installation time: `ANTHROPIC_API_KEY`, `TELEGRAM_BOT_TOKEN`, `LOG_LEVEL`, `ANTHROPIC_MODEL`, `VISION_LANG`, `ANTHROPIC_SYSTEM_PROMPT`, `SUBSCRIBER_STORE_PATH`. Other optional variables (timeouts, accuracy modes, etc.) will use their built-in defaults unless you modify the plist directly.

After installation, you may need to re-grant **Accessibility** and **Screen Recording** permissions in System Settings if they don't automatically persist. See [Permissions](#-permissions) below.

### Build Commands

| Command | Purpose |
|---|---|
| `make build` | Compile the binary |
| `make run` | Build and run in foreground |
| `make clean` | Remove the binary |
| `make check` | Run all static checks (fmt, vet, lint, vuln) |
| `make fmt` | Format source files with gofmt |
| `make vet` | Run go vet static analysis |
| `make lint` | Run golangci-lint static analysis |
| `make vuln` | Run govulncheck vulnerability scanner |
| `make tools` | Install golangci-lint and govulncheck tools |

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
| [`anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) | Official Anthropic Go SDK for Claude AI API |

**Indirect dependencies** (pulled in by the Anthropic SDK): `tidwall/gjson`, `tidwall/sjson`, `tidwall/match`, `tidwall/pretty`, `golang.org/x/sync`.

**Built-in frameworks**: CoreGraphics (screen capture via CGo), Apple Vision framework (OCR via CGo), AppleScript (window detection). No external OCR libraries required.
