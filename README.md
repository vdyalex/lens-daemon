# ccat-assistant

A macOS daemon that captures your screen on demand via a global hotkey, extracts text using OCR, processes it through a local LLM (Ollama), and delivers the response to Telegram. All operations happen in-memory with zero disk writes.

## How It Works

```
Right Option Key -> Screen Capture -> OCR -> Claude AI -> Telegram
```

1. **Hotkey** -- listens for the **right Option key** (`kVK_RightOption`, keycode `0x3D`) using a macOS `CGEventTap` global keyboard listener
2. **Capture** -- grabs the center 60% of the active window via AppleScript (window detection) and the `kbinani/screenshot` library (pixel capture)
3. **OCR** -- extracts text from the captured image using Tesseract, entirely in-memory (RGBA -> PNG encode -> Tesseract, no temp files)
4. **AI** -- sends the extracted text to Claude with a configurable system prompt (max 1024 response tokens)
5. **Notify** -- delivers Claude's response to a Telegram chat via the Bot API, auto-chunking messages that exceed 4096 characters

Captures happen **only** when you press the hotkey. There is no continuous screen polling or background recording.

## Functional Requirements

- **Global hotkey detection**: Listen for the right Option key system-wide using macOS `CGEventTap` in listen-only mode. The event tap runs on a dedicated OS thread with its own `CFRunLoop` and automatically re-enables itself if the system disables it due to timeout or user input.
- **Active window detection**: Identify the frontmost application window (name, position, size) via AppleScript and `System Events`. Unparseable coordinates in the osascript output are treated as errors and surface through the pipeline's non-fatal error path (logged, hotkey listener continues).
- **Screen capture**: Capture the center 60% of the active window (20% margin on each side). For fullscreen windows (width >= screen width AND height >= screen height), capture the center 60% of the entire display instead.
- **OCR text extraction**: Convert the captured image to text using Tesseract OCR. The image is PNG-encoded in memory and passed directly to Tesseract via byte buffer -- no intermediate files touch the disk.
- **AI processing**: Send extracted text to Claude AI with a configurable system prompt. Empty OCR results are silently skipped (no API call made).
- **Telegram delivery**: Send Claude's response to a configured Telegram chat. Messages exceeding Telegram's 4096-character limit are automatically split into sequential chunks. Empty AI responses are silently skipped. The HTTP client enforces a 30-second timeout per request; a non-responsive Telegram API will not stall the pipeline indefinitely.
- **Graceful shutdown**: Handle `SIGINT` and `SIGTERM` signals to cleanly shut down the event tap, release Tesseract resources, and exit.
- **Non-fatal runtime errors**: Pipeline errors (capture failure, OCR failure, API errors) are logged but do not terminate the daemon. The hotkey listener continues running for the next trigger.

## Non-Functional Requirements

- **Platform**: macOS only. Uses macOS-specific APIs (`CGEventTap`, `CFRunLoop`, `CoreGraphics`, `CoreFoundation`) via cgo and AppleScript via `osascript`.
- **In-memory processing**: No screenshots, intermediate images, or temporary files are written to disk at any point in the pipeline.
- **Daemon operation**: Runs as a background CLI daemon. Does not appear in the Dock or Cmd-Tab (pure CLI process with no GUI elements).
- **Event-driven**: Idle until the hotkey is pressed. No polling, no timers, no periodic screen checks.
- **Single-trigger pipeline**: Each hotkey press executes a full sequential pipeline (capture -> OCR -> AI -> Telegram). The pipeline does not overlap or queue multiple triggers; the hotkey channel has a buffer of 1 with non-blocking sends.
- **Language**: Go 1.24+ with cgo (for `CoreGraphics`/`CoreFoundation` bindings and Tesseract C library).
- **Settings**: All settings via environment variables. No config files, no CLI flags.
- **Logging**: Structured log output to stderr using Go's slog TextHandler (time, level, message, and key-value fields). Log verbosity is controlled by `LOG_LEVEL`.
- **External dependencies**: Tesseract OCR engine must be installed on the host system (e.g., via Homebrew).
- **Security permissions**: Requires macOS Accessibility and Screen Recording permissions granted to the terminal or binary.

## Prerequisites

- **macOS** (uses CoreGraphics CGEventTap and AppleScript)
- **Go 1.24+** (with cgo support)
- **Tesseract OCR** -- install via Homebrew:
  ```bash
  brew install tesseract
  ```
- **Anthropic API key** -- get one at [console.anthropic.com](https://console.anthropic.com)
- **Telegram bot** -- create one via [@BotFather](https://t.me/BotFather) and obtain your chat ID

## Installation

```bash
git clone https://github.com/vdyalex/ccat-assistant.git
cd ccat-assistant
make build
```

This produces the `ccat` binary in the project root.

## Configuration

All configuration is done through environment variables. Copy `.env.example` to `.env` and fill in your values, or export them directly.

### Required

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Anthropic API key (e.g., `sk-ant-...`) |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token from @BotFather (e.g., `123456:ABC-DEF...`) |
| `TELEGRAM_CHAT_ID` | Numeric Telegram chat ID to send messages to |

### Optional

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `TESSERACT_LANG` | `eng` | Tesseract language pack code (e.g., `deu`, `fra`, `rus`) |
| `CLAUDE_MODEL` | `claude-sonnet-4-6` | Claude model ID to use for AI processing |
| `SYSTEM_PROMPT` | *(built-in)* | System prompt sent to Claude with each request |

The built-in default system prompt is:

> You're tasked with assisting with a CCAT examination. You're a logical and practical assistant who only returns quick logical responses with maximum efficiency and accuracy.

## Usage

```bash
# Export required variables
export ANTHROPIC_API_KEY="sk-ant-..."
export TELEGRAM_BOT_TOKEN="123456:ABC..."
export TELEGRAM_CHAT_ID="987654321"

# Run in foreground
./ccat

# Or build and run in background
make run
```

Once running, press the **right Option key** at any time to trigger a capture. The daemon captures the active window, runs OCR, sends the text to Claude, and forwards the response to your Telegram chat.

Press `Ctrl+C` to stop. The daemon handles `SIGINT`/`SIGTERM` gracefully, releasing all resources before exiting.

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

## Project Structure

```
ccat-assistant/
|-- src/
|   |-- main.go                 Entry point, signal handling, graceful shutdown
|   |-- pipeline/
|   |   `-- pipeline.go         Orchestrates the capture -> OCR -> AI -> Telegram flow
|   |-- modules/
|   |   |-- listener/
|   |   |   `-- listener.go     Global hotkey listener (macOS CGEventTap via cgo)
|   |   |-- capturer/
|   |   |   |-- capturer.go     Capturer interface, WindowInfo type, center-rect utility
|   |   |   `-- capturer_macos.go   macOS implementation (AppleScript + screenshot library)
|   |   `-- extractor/
|   |       `-- extractor.go    Tesseract OCR wrapper (in-memory image processing)
|   |-- adapters/
|   |   |-- agent/
|   |   |   `-- agent.go        Claude AI client (Anthropic SDK)
|   |   `-- messenger/
|   |       `-- messenger.go    Telegram Bot API sender with automatic message chunking
|   `-- utils/
|       `-- config.go           Environment variable loader with defaults and validation
|-- Makefile                    Build automation (build, run, clean, check)
|-- go.mod                      Go module definition
|-- go.sum                      Dependency checksums
`-- README.md                   This file
```

## Architecture

### Pipeline Flow

```
main()
 |-- config.Load()                   Load and validate env vars
 |-- pipeline.New(cfg, logger)       Wire all components:
 |    |-- extractor.New(lang)          Initialize Tesseract client
 |    |-- capturer.New()               Create macOS capturer
 |    |-- agent.New(...)               Create Anthropic SDK client
 |    `-- messenger.New(...)           Create Telegram HTTP sender
 |-- signal.Notify(SIGINT, SIGTERM)  Register shutdown handler
 `-- pipeline.Run(ctx)               Main event loop:
      |-- listener.Listen(ctx)         Start CGEventTap on dedicated OS thread
      `-- loop:
           |-- <-triggers             Wait for right Option key press
           `-- pipeline.process():
                |-- ForegroundWindow()   AppleScript -> window info
                |-- CaptureCenter()      Screenshot center 60% region
                |-- extractor.Extract()  RGBA -> PNG -> Tesseract -> text
                |-- agent.Process()      Text -> Claude API -> response
                `-- messenger.Send()      Response -> Telegram (chunked)
```

### Key Design Decisions

- **CGEventTap in listen-only mode**: The event tap (in `modules/listener/`) observes keyboard events without modifying or consuming them. Other applications continue to receive all key events normally. On shutdown, the listener uses a `CFRunLoopRunInMode` polling loop (0.5 s timeout per iteration) to check for context cancellation, then disables the tap and releases all C resources (`CFRelease`) before the goroutine exits.
- **Non-blocking hotkey channel**: The C callback sends to a buffered channel (capacity 1) with a non-blocking select, ensuring the `CFRunLoop` is never stalled by a slow pipeline execution.
- **Center 60% capture**: The capturer (`modules/capturer/`) crops the outer 20% on all sides to focus on the primary content area of the window, reducing noise from toolbars, sidebars, and window chrome.
- **Fullscreen detection**: When the window dimensions match or exceed the screen size, the capture falls back to the full display bounds.
- **In-memory image pipeline**: Images flow as `*image.RGBA` through the pipeline and are PNG-encoded into a byte buffer only when passed to Tesseract (in `modules/extractor/`). No files are created.
- **Message chunking**: Telegram's 4096-character message limit is handled by the messenger adapter (`adapters/messenger/`) by splitting at rune boundaries (respecting Unicode character width) and sending sequential chunks.

## Dependencies

| Dependency | Purpose |
|---|---|
| [`anthropic-sdk-go`](https://github.com/anthropics/anthropic-sdk-go) | Official Anthropic Go SDK for Claude AI API |
| [`kbinani/screenshot`](https://github.com/kbinani/screenshot) | Screen capture (macOS backend) |
| [`otiai10/gosseract`](https://github.com/otiai10/gosseract) | Go bindings for Tesseract OCR engine |

Indirect dependencies (`tidwall/gjson`, `tidwall/sjson`, `golang.org/x/sync`, `golang.org/x/sys`) are pulled in by the Anthropic SDK.

## Build Commands

```bash
make build    # Compile to ./ccat
make run      # Build and run in background
make clean    # Remove the binary
make check    # Run gofmt and go vet (fmt + vet)
make fmt      # Format source files with gofmt
make vet      # Run go vet static analysis
```

## License

See [LICENSE](LICENSE) for details.
