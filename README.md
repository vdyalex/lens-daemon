# Lens

A macOS daemon that captures your screen on demand via a global hotkey, extracts text using OCR, processes it through Claude AI (Anthropic), and broadcasts the response to Telegram subscribers. All operations happen in-memory with zero disk writes for screenshots.

## ⚡ How it works

```
Right Shift Key → Screen Capture → OCR → Claude AI → Telegram
```

1. **Hotkey** — right Shift key (`kVK_RightShift`) triggers a capture via macOS `CGEventTap` global keyboard listener
2. **Capture** — grabs the entire active window via AppleScript and the `kbinani/screenshot` library. Use right Option key to define custom bounds
3. **OCR** — extracts text from the image using Apple Vision framework, entirely in-memory
4. **AI** — sends extracted text to Claude with a configurable system prompt (max 1024 response tokens)
5. **Notify** — broadcasts Claude's response to all Telegram subscribers, auto-chunking messages exceeding 4096 characters

Captures happen **only** when you press the hotkey. No continuous polling or background recording.

For detailed architecture, design decisions, and requirements, see [REQUIREMENTS.md](REQUIREMENTS.md).

## 📋 Pre-requisites

- **macOS** (uses CoreGraphics CGEventTap, Vision framework, and AppleScript)
- **Go 1.24+** (with cgo support)
- **Anthropic API key** — get one at [console.anthropic.com](https://console.anthropic.com)
- **Telegram bot** — create one via [@BotFather](https://t.me/BotFather)

## 🔧 Installation

```bash
git clone https://github.com/vdyalex/lens-daemon.git
cd lens-daemon
make build
```

This produces the `lensd` binary in the project root.

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
| `TELEGRAM_CHAT_ID` | `0` (none) | Seed an initial subscriber chat ID (legacy single-chat mode). If set, this chat is added as the first subscriber on startup |
| `SUBSCRIBER_STORE_PATH` | `tmp/subscribers` | File path for the subscriber list (persists users who sent `/start`) |
| `LOG_LEVEL` | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `VISION_LANG` | `en-US` | OCR language (BCP 47 code, e.g., `en-US`, `fr-FR`, `de-DE`, `zh-Hans`, `ja`, `ko`) |
| `CLAUDE_MODEL` | `claude-sonnet-4-6` | Claude model ID to use for AI processing |
| `SYSTEM_PROMPT` | *(built-in)* | System prompt sent to Claude with each request |

The built-in default system prompt is:

> You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency.

## ▶️ Running

### Manually

```bash
# Export required variables
export ANTHROPIC_API_KEY="sk-ant-..."
export TELEGRAM_BOT_TOKEN="123456:ABC..."

# Run in foreground
./lensd

# Or build and run
make run
```

Once running:

1. **Subscribe**: Send `/start` to your Telegram bot to begin receiving responses
2. **Capture**: Press the **right Shift key** at any time to trigger a capture
3. **Custom bounds** (optional): Hold the **right Option key**, move your mouse to define a region, then release
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

The service is managed as a macOS LaunchAgent (`com.vdyalex.lensd`) and environment variables from your `.env` file are embedded into the LaunchAgent plist at installation time.

After installation, you may need to re-grant **Accessibility** and **Screen Recording** permissions in System Settings if they don't automatically persist. See [Permissions](#-permissions) below.

### Build Commands

| Command | Purpose |
|---|---|
| `make build` | Compile the binary |
| `make run` | Build and run in foreground |
| `make clean` | Remove the binary |
| `make check` | Run gofmt and go vet (fmt + vet) |
| `make fmt` | Format source files with gofmt |
| `make vet` | Run go vet static analysis |

## 🔐 Permissions

macOS will prompt for two permissions on first run. Both must be granted for the daemon to function:

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
| [`kbinani/screenshot`](https://github.com/kbinani/screenshot) | Screen capture (macOS backend) |

Indirect dependencies (`tidwall/gjson`, `tidwall/sjson`, `golang.org/x/sync`, `golang.org/x/sys`) are pulled in by the Anthropic SDK. The Vision framework is built-in to macOS.
