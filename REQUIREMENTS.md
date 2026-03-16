# Requirements

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
      |-- bounds tracker               Listen for configured bounds hotkey updates
      `-- worker loop:
           |-- <-triggers             Wait for configured trigger hotkey press
           `-- pipeline.process():\
                |-- ForegroundWindow()   AppleScript -> window info (5s timeout)
                |-- CaptureCenter()      Screenshot entire window or custom bounds (30s timeout)
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

By default, the daemon captures the entire active window. You can override this with custom screen-coordinate bounds:

1. **Hold the configured bounds hotkey** (default: `RightOption`, customizable via `HOTKEY_BOUNDS_KEYNAME`) and move your mouse to define a rectangular region
2. The daemon tracks the minimum and maximum coordinates of your mouse movement while the key is held
3. **Release the bounds hotkey** to lock in the bounds
4. All subsequent captures will use the custom bounds instead of capturing the full window

For fullscreen windows (width and height >= screen dimensions), the daemon captures the entire display.

### Key Design Decisions

- **CGEventTap in listen-only mode**: The event tap observes keyboard and mouse events without modifying or consuming them. Other applications continue to receive all events normally. On shutdown, the listener disables the tap and releases all C resources before the goroutine exits.
- **Non-blocking hotkey channel**: The C callback sends to a buffered channel with a non-blocking select, ensuring the `CFRunLoop` is never stalled by a slow pipeline execution.
- **Async worker processing**: A dedicated worker goroutine processes captures sequentially (limited to 1 concurrent run) while the main loop stays responsive to new hotkey triggers. If the queue is full, additional triggers are silently dropped.
- **In-memory image pipeline**: Images flow as `*image.RGBA` through the pipeline and are PNG-encoded into a byte buffer only when passed to the Vision API. No files are created at any point.
- **Atomic subscriber persistence**: The subscriber store writes to a temporary file and uses `os.Rename()` for atomic updates, protected by a read-write mutex for concurrent access.
- **MarkdownV2 formatting**: All Telegram messages are converted to MarkdownV2 format, escaping special characters for proper rendering.
- **Message chunking**: Telegram's 4096-character message limit is handled by splitting at rune boundaries (respecting Unicode character width) and sending sequential chunks. Rate-limited responses (HTTP 429) are retried with the server-specified backoff.

## Functional Requirements

- **Global hotkey detection**: Listen for a configurable trigger hotkey system-wide (default: `RightShift`, customizable via `HOTKEY_TRIGGER_KEYNAME`) using macOS `CGEventTap` in listen-only mode. The event tap runs on a dedicated OS thread with its own `CFRunLoop` and automatically re-enables itself if the system disables it due to timeout or user input.
- **Custom bounds selection**: Track mouse movement while a configurable bounds hotkey is held (default: `RightOption`, customizable via `HOTKEY_BOUNDS_KEYNAME`) to define a custom capture rectangle. The bounds persist until the daemon is restarted or new bounds are set.
- **Active window detection**: Identify the frontmost application window (name, position, size) via AppleScript and `System Events`. Unparseable coordinates in the AppleScript output are treated as errors and surface through the pipeline's non-fatal error path (logged, hotkey listener continues).
- **Screen capture**: Capture the entire active window, or use custom bounds if set. For fullscreen windows (width >= screen width AND height >= screen height), capture the entire display instead.
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
