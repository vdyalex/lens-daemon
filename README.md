# ccat-assistant

A macOS daemon that captures your screen on demand via a global hotkey, extracts text using OCR, processes it through Claude AI (Anthropic), and broadcasts the response to Telegram subscribers. All operations happen in-memory with zero disk writes for screenshots.

## How It Works

```
Right Option Key -> Screen Capture -> OCR -> Claude AI -> Telegram
```

1. **Hotkey** -- listens for the **right Option key** (`kVK_RightOption`, keycode `0x3D`) using a macOS `CGEventTap` global keyboard listener
2. **Capture** -- grabs the center 60% of the active window via AppleScript (window detection) and the `kbinani/screenshot` library (pixel capture). Optionally uses custom bounds set via the right Shift key
3. **OCR** -- extracts text from the captured image using Apple Vision framework, entirely in-memory (RGBA -> PNG encode -> Vision API, no temp files)
4. **AI** -- sends the extracted text to Claude with a configurable system prompt (max 1024 response tokens)
5. **Notify** -- broadcasts Claude's response to all Telegram subscribers via the Bot API, formatting messages in MarkdownV2 and auto-chunking messages that exceed 4096 characters

Captures happen **only** when you press the hotkey. There is no continuous screen polling or background recording.

## Architecture

### Pipeline Design

The application follows a clean layered architecture with interface-based design, separating concerns into modules (platform-specific capabilities) and adapters (external service integrations), orchestrated by a central pipeline.

**Modules** handle macOS-specific operations:
- **Listener** -- global hotkey detection and bounds tracking via `CGEventTap` (cgo)
- **Capturer** -- foreground window detection (AppleScript) and screenshot capture
- **Extractor** -- OCR text extraction interface consumed by the pipeline

**Adapters** integrate with external services:
- **Agent** -- Claude AI client using the official Anthropic Go SDK
- **Messenger** -- Telegram Bot API sender with message chunking and MarkdownV2 formatting
- **Poller** -- Telegram long-polling for subscriber management (`/start`, `/stop` commands)
- **Subscriber Store** -- JSON-backed persistence for subscriber chat IDs
- **Vision** -- Objective-C bridge to Apple's Vision framework for OCR

**Pipeline** orchestrates the full workflow, wiring all components together at startup and processing each hotkey trigger through the sequential capture-OCR-AI-Telegram flow.

### Pipeline Flow

```
main()
 |-- config.Load()                   Load and validate env vars
 |-- pipeline.New(cfg, logger)       Wire all components:
 |    |-- extractor.New(lang)          Initialize Vision extractor
 |    |-- subscriber.NewStore(...)     Load subscriber list from JSON
 |    |-- messenger.New(...)           Create Telegram broadcaster
 |    |-- poller.New(...)              Create Telegram subscriber poller
 |    |-- capturer.New()               Create macOS capturer
 |    `-- agent.New(...)               Create Anthropic SDK client
 |-- signal.Notify(SIGINT, SIGTERM)  Register shutdown handler
 `-- pipeline.Run(ctx)               Main event loop:
      |-- listener.Listen(ctx)         Start CGEventTap on dedicated OS thread
      |-- poller.Run(ctx)              Start Telegram subscriber poller (background)
      |-- bounds tracker               Listen for right-Shift bounds updates
      `-- worker loop:
           |-- <-triggers             Wait for right Option key press
           `-- pipeline.process():
                |-- ForegroundWindow()   AppleScript -> window info (5s timeout)
                |-- CaptureCenter()      Screenshot center 60% or custom bounds (30s timeout)
                |-- extractor.Extract()  RGBA -> PNG -> Vision API -> text (30s timeout)
                |-- agent.Process()      Text -> Claude API -> response (60s timeout)
                `-- messenger.Broadcast() Response -> all Telegram subscribers (30s timeout)
```

Each pipeline run has an overall timeout of 5 minutes. Individual step timeouts are enforced to prevent any single operation from stalling the daemon indefinitely.

### Subscriber Management

The daemon supports multiple Telegram subscribers through a dynamic subscription system:

- Users send `/start` to the Telegram bot to subscribe and receive responses
- Users send `/stop` to unsubscribe
- The subscriber list is persisted to a JSON file (default: `subscribers.json`) and survives daemon restarts
- All responses are broadcast to every active subscriber
- The optional `TELEGRAM_CHAT_ID` environment variable can seed an initial subscriber (legacy single-chat mode)

The Telegram poller runs in the background, long-polling for updates with a 30-second timeout and 5-second retry backoff on errors.

### Custom Capture Bounds

By default, the daemon captures the center 60% of the active window (20% margin on each side). You can override this with custom screen-coordinate bounds:

1. **Hold the right Shift key** and move your mouse to define a rectangular region
2. The daemon tracks the minimum and maximum coordinates of your mouse movement while the key is held
3. **Release right Shift** to lock in the bounds
4. All subsequent captures will use the custom bounds instead of the default center crop

For fullscreen windows (width and height >= screen dimensions), the daemon captures the center 60% of the entire display.

### Key Design Decisions

- **CGEventTap in listen-only mode**: The event tap observes keyboard and mouse events without modifying or consuming them. Other applications continue to receive all events normally. On shutdown, the listener disables the tap and releases all C resources before the goroutine exits.
- **Non-blocking hotkey channel**: The C callback sends to a buffered channel with a non-blocking select, ensuring the `CFRunLoop` is never stalled by a slow pipeline execution.
- **Async worker processing**: A dedicated worker goroutine processes captures sequentially (limited to 1 concurrent run) while the main loop stays responsive to new hotkey triggers. If the queue is full, additional triggers are silently dropped.
- **In-memory image pipeline**: Images flow as `*image.RGBA` through the pipeline and are PNG-encoded into a byte buffer only when passed to the Vision API. No files are created at any point.
- **Atomic subscriber persistence**: The subscriber store writes to a temporary file and uses `os.Rename()` for atomic updates, protected by a read-write mutex for concurrent access.
- **MarkdownV2 formatting**: All Telegram messages are converted to MarkdownV2 format, escaping special characters for proper rendering.
- **Message chunking**: Telegram's 4096-character message limit is handled by splitting at rune boundaries (respecting Unicode character width) and sending sequential chunks. Rate-limited responses (HTTP 429) are retried with the server-specified backoff.

## Functional Requirements

- **Global hotkey detection**: Listen for the right Option key system-wide using macOS `CGEventTap` in listen-only mode. The event tap runs on a dedicated OS thread with its own `CFRunLoop` and automatically re-enables itself if the system disables it due to timeout or user input.
- **Custom bounds selection**: Track mouse movement while the right Shift key is held to define a custom capture rectangle. The bounds persist until the daemon is restarted or new bounds are set.
- **Active window detection**: Identify the frontmost application window (name, position, size) via AppleScript and `System Events`. Unparseable coordinates in the osascript output are treated as errors and surface through the pipeline's non-fatal error path (logged, hotkey listener continues).
- **Screen capture**: Capture the center 60% of the active window (20% margin on each side), or use custom bounds if set. For fullscreen windows (width >= screen width AND height >= screen height), capture the center 60% of the entire display instead.
- **OCR text extraction**: Convert the captured image to text using Apple Vision framework. The image is PNG-encoded in memory and passed directly to the Vision API via byte buffer -- no intermediate files touch the disk.
- **AI processing**: Send extracted text to Claude AI with a configurable system prompt. Empty OCR results are silently skipped (no API call made).
- **Telegram delivery**: Broadcast Claude's response to all active subscribers. Messages exceeding Telegram's 4096-character limit are automatically split into sequential chunks. Empty AI responses are silently skipped. The HTTP client enforces a 30-second timeout per request; a non-responsive Telegram API will not stall the pipeline indefinitely.
- **Subscriber management**: Support dynamic subscriber registration via Telegram `/start` and `/stop` bot commands. Persist the subscriber list to a JSON file for durability across restarts.
- **Graceful shutdown**: Handle `SIGINT` and `SIGTERM` signals to cleanly shut down the event tap, poller, and extractor before exiting.
- **Non-fatal runtime errors**: Pipeline errors (capture failure, OCR failure, API errors) are logged but do not terminate the daemon. The hotkey listener continues running for the next trigger.

## Non-Functional Requirements

- **Platform**: macOS only. Uses macOS-specific APIs (`CGEventTap`, `CFRunLoop`, `CoreGraphics`, `CoreFoundation`, `Vision framework`) via cgo and AppleScript via `osascript`.
- **In-memory processing**: No screenshots, intermediate images, or temporary files are written to disk at any point in the pipeline.
- **Daemon operation**: Runs as a background CLI daemon. Does not appear in the Dock or Cmd-Tab (pure CLI process with no GUI elements).
- **Event-driven**: Idle until the hotkey is pressed. No polling, no timers, no periodic screen checks.
- **Single-trigger pipeline**: Each hotkey press executes a full sequential pipeline (capture -> OCR -> AI -> Telegram). The pipeline does not overlap; if a capture is already in progress, additional triggers are dropped.
- **Language**: Go 1.24+ with cgo (for `CoreGraphics`/`CoreFoundation` bindings and Vision framework).
- **Settings**: All settings via environment variables. No config files, no CLI flags.
- **Logging**: Structured log output to stderr using Go's slog TextHandler (time, level, message, and key-value fields). Log verbosity is controlled by `LOG_LEVEL`.
- **External dependencies**: No external OCR dependencies required (uses built-in Apple Vision framework).
- **Security permissions**: Requires macOS Accessibility and Screen Recording permissions granted to the terminal or binary.

## Prerequisites

- **macOS** (uses CoreGraphics CGEventTap, Vision framework, and AppleScript)
- **Go 1.24+** (with cgo support)
- **Anthropic API key** -- get one at [console.anthropic.com](https://console.anthropic.com)
- **Telegram bot** -- create one via [@BotFather](https://t.me/BotFather)

## Installation

```bash
git clone https://github.com/vdyalex/ccat-assistant.git
cd ccat-assistant
make build
```

This produces the `networkd` binary in the project root.

## Configuration

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
| `SUBSCRIBER_STORE_PATH` | `subscribers.json` | File path for the subscriber list (persists users who sent `/start`) |
| `LOG_LEVEL` | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `VISION_LANG` | `en-US` | OCR language (BCP 47 code, e.g., `en-US`, `fr-FR`, `de-DE`, `zh-Hans`, `ja`, `ko`) |
| `CLAUDE_MODEL` | `claude-sonnet-4-6` | Claude model ID to use for AI processing |
| `SYSTEM_PROMPT` | *(built-in)* | System prompt sent to Claude with each request |

The built-in default system prompt is:

> You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency.

## Usage

### Run Manually

```bash
# Export required variables
export ANTHROPIC_API_KEY="sk-ant-..."
export TELEGRAM_BOT_TOKEN="123456:ABC..."

# Run in foreground
./networkd

# Or build and run
make run
```

Once running:

1. **Subscribe**: Send `/start` to your Telegram bot to begin receiving responses
2. **Capture**: Press the **right Option key** at any time to trigger a capture
3. **Custom bounds** (optional): Hold the **right Shift key**, move your mouse to define a region, then release
4. The daemon captures the screen, runs OCR, sends the text to Claude, and broadcasts the response to all subscribers

Send `/stop` to the Telegram bot to unsubscribe. Press `Ctrl+C` to stop the daemon. It handles `SIGINT`/`SIGTERM` gracefully, releasing all resources before exiting.

### Run as a Service (Background Daemon)

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
- Log output to `~/Library/Logs/ccat/stdout.log` and `~/Library/Logs/ccat/stderr.log`

**Service management commands:**

```bash
make service-logs       # View real-time logs
make service-stop       # Stop the service
make service-start      # Start the service (if already installed)
make service-uninstall  # Remove the service
```

The service is managed as a macOS LaunchAgent (`com.vdyalex.assistant`) and environment variables from your `.env` file are embedded into the LaunchAgent plist at installation time.

After installation, you may need to re-grant **Accessibility** and **Screen Recording** permissions in System Settings if they don't automatically persist. See [macOS Permissions](#macos-permissions) below.

## macOS Permissions

macOS will prompt for two permissions on first run. Both must be granted for the daemon to function:

| Permission | Required For | Where to Grant |
|---|---|---|
| **Accessibility** | Global hotkey listener (`CGEventTap`) | System Settings -> Privacy & Security -> Accessibility |
| **Screen Recording** | Screen capture | System Settings -> Privacy & Security -> Screen Recording |

If Accessibility permission is not granted, the daemon will fail on startup with:

```
CGEventTapCreate failed -- grant Accessibility permission to this app
```

## Dependencies

| Dependency | Purpose |
|---|---|
| [`anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) | Official Anthropic Go SDK for Claude AI API |
| [`kbinani/screenshot`](https://github.com/kbinani/screenshot) | Screen capture (macOS backend) |

Indirect dependencies (`tidwall/gjson`, `tidwall/sjson`, `golang.org/x/sync`, `golang.org/x/sys`) are pulled in by the Anthropic SDK. The Vision framework is built-in to macOS.

## Build Commands

```bash
make build              # Compile the binary
make run                # Build and run in foreground
make clean              # Remove the binary

make check              # Run gofmt and go vet (fmt + vet)
make fmt                # Format source files with gofmt
make vet                # Run go vet static analysis

make service-install    # Install as a background service (autostart on login)
make service-uninstall  # Uninstall the background service
make service-start      # Start the service (if already installed)
make service-stop       # Stop the running service
make service-logs       # View real-time service logs
```

## License

See [LICENSE](LICENSE) for details.
