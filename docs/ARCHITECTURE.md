# Architecture

## CLI and Daemon Architecture

The application uses a **single binary, multiple subcommands** architecture:

**CLI Layer** (`src/cmd/`) provides user-facing commands via Cobra:

- **`daemon`** -- runs the full pipeline with IPC server (called by `start` command)
- **`start`** -- daemonizes the process by re-execing `<binary> daemon` in a new session (uses `syscall.SysProcAttr{Setsid: true}`)
- **`stop`** -- sends SIGTERM to the daemon via PID file
- **`status`** -- queries daemon status (PID, uptime, subscriber count, last window) via IPC, one-shot output
- **`logs`** -- subscribes to IPC log stream and streams colorized log output to stdout
- **`restart`** -- stops and starts the daemon with config flags

**Daemon Lifecycle** (`src/daemon/`) handles process management:

- **PID file** -- stores daemon PID for status checks and shutdown signals (path: `$TMPDIR/<binary>-<uid>.pid`, where `<binary>` is the compiled binary name)
- **Daemonize** -- re-execs current binary with `Setsid: true` for full detachment from terminal
- Config flags are forwarded as environment variables to the re-exec'd child process

**IPC Layer** (`src/ipc/`) enables inter-process communication:

- **Unix domain socket** at `$TMPDIR/<binary>-<uid>.sock` with `0600` permissions
- **Length-prefixed JSON** wire format (4-byte big-endian length + UTF-8 JSON body)
- **Log broker** -- fan-out io.Writer that distributes slog text lines to subscribed IPC clients
- **Handler** -- dispatches IPC commands: `status`, `shutdown`, `log.subscribe`

## Pipeline Design

The pipeline orchestrates the full workflow with a **two-phase design**:

- **Phase 1 (Capture)**: One goroutine per hotkey trigger; captures complete independently and in parallel
- **Phase 2 (Analyse)**: Results queue up and are processed concurrently (one goroutine per queued result) for OCR-AI-Teleprompter-Telegram

This separation decouples fast Phase 1 captures from slow Phase 2 analysis, preventing missed captures when the pipeline is busy.

**Modules** handle MacOS-specific operations:

- **Listener** -- global hotkey detection, bounds tracking, teleprompter toggle, position rotation, and opacity adjustment via `CGEventTap` (cgo)
- **Capturer** -- foreground window detection (AppleScript) and screenshot capture, using `src/bridges/core_graphics` for CoreGraphics calls
- **Extractor** -- OCR text extraction interface consumed by the pipeline
- **Teleprompter** -- stealth overlay management (visibility toggle, text display) delegating to the appkit bridge

**Adapters** integrate with external services:

- **AI** -- Claude AI client using the official Anthropic Go SDK with structured tool calls returning `short` and `detailed` response branches (package `ai`)
- **IM** -- Telegram Bot API sender with message chunking and MarkdownV2 formatting (package `im`). Optional: when `TELEGRAM_BOT_TOKEN` is absent, a `NoopBroadcaster` and `NoopPoller` are used instead
  - **Poller** -- Telegram long-polling for subscriber management (`/start`, `/stop` commands) (package `im/poller`)
  - **Store** -- plain-text file-backed persistence for subscriber chat IDs (package `im/store`)
- **OCR** -- Apple Vision framework adapter wrapping `src/bridges/vision` (package `ocr`)

**Bridges** separate CGo boundaries into dedicated packages:

- **`src/bridges/vision`** -- Objective-C wrapper for Apple Vision framework OCR (`visionRecognizeText`)
- **`src/bridges/core_graphics`** -- Objective-C wrappers for CoreGraphics screen capture and display queries (`captureScreenRect`, `getMainDisplayWidth`, `getMainDisplayHeight`)
- **`src/bridges/appkit`** -- Objective-C wrappers for AppKit overlay window (NSWindow creation, text rendering, positioning) and NSApplication run loop management
- **`src/bridges/browser`** -- Pure Go package that derives the browser content-area rectangle from window geometry, excluding the browser toolbar (address bar, tab bar). Supports Safari (74 px), Google Chrome (88 px), and Firefox (90 px). Returns nil for unrecognised apps so callers fall back to the full window

**Pipeline Components** (`src/pipeline/`) orchestrate the workflow:

- **`pipeline.go`** -- constructors and public interface (`New`, `NewWithDependencies`, `Status`, `Run`)
- **`process.go`** -- implementation of the sequential process steps (fetch window, derive canvas bounds, capture, extract, process with AI, display on teleprompter, broadcast to Telegram)
- **`run.go`** -- event loop and goroutine orchestration (`Run` method), including visibility toggle and position tracking

## Startup and Daemon Flow

```mermaid
flowchart TD
    A([binary start<br/>user command]) --> B[daemon.Daemonize<br/>re-exec with Setsid]
    B --> C[child: binary daemon<br/>new session]
    A --> D[poll for PID file<br/>startup confirmation]

    C --> E[config.Load<br/>Load and validate env vars]
    E --> F[daemon.WritePID<br/>Write PID file]
    F --> G[pipeline.New<br/>Wire all components]
    G --> H[ipc.NewServer<br/>Listen on Unix socket]

    subgraph Init[Initialization]
        G --> I[extractor.New<br/>Vision OCR extractor]
        G --> J[store.NewStore<br/>Load subscriber list<br/>skipped if no bot token]
        G --> K[im.New or NoopBroadcaster<br/>Telegram broadcaster]
        G --> L[poller.New or NoopPoller<br/>Telegram subscriber poller]
        G --> M[capturer.New<br/>MacOS capturer]
        G --> N[ai.New<br/>Anthropic SDK client<br/>structured tool calls]
        G --> O2[teleprompter.New<br/>Stealth overlay window]
    end

    H --> O[ipc.NewLogBroker<br/>Create log fan-out]
    O --> P[Start IPC server<br/>background goroutine]
    P --> Q[signal.Notify<br/>SIGINT / SIGTERM]
    Q --> R([pipeline.Run])

    subgraph Runtime[Runtime - concurrent goroutines]
        R --> S[listener.Listen<br/>CGEventTap on OS thread]
        R --> T[poller.Run<br/>Telegram subscriber poller]
        R --> U[bounds tracker<br/>Bounds hotkey updates]
        R --> U2[visibility tracker<br/>Toggle hotkey]
        R --> U3[position tracker<br/>Arrow key repositioning]
        R --> V{Wait for<br/>trigger hotkey or<br/>IPC commands}
    end

    V -->|status/logs request| W[ipc.Handler<br/>Query pipeline or stream logs]
    V -->|hotkey pressed| X([Phase 1: Capture<br/>5 min overall timeout])

    subgraph Phase1[Phase 1: Capture - Sequential per trigger]
        X --> Y[ForegroundWindow<br/>AppleScript - window info<br/>5 s timeout]
        Y --> Y2[browser.CanvasBounds<br/>Derive content-area rect<br/>from app name + window geometry]
        Y2 --> Z[CaptureCenter<br/>captureBounds or canvasBounds or full window<br/>30 s timeout]
    end

    Phase1 --> X2{Enqueue to<br/>analyseQueue}
    X2 -->|queue not full| X3[Phase 2: Analyse<br/>One goroutine per result]

    subgraph Phase2[Phase 2: Analyse - Concurrent per result]
        X3 --> AA[extractor.Extract<br/>RGBA - PNG - Vision API - text<br/>30 s timeout]
        AA --> AB[agent.Process<br/>Text - Claude API - response<br/>60 s timeout]
        AB --> AC[messenger.Broadcast<br/>Response - all Telegram subscribers<br/>30 s timeout]
    end

    AC --> V
    W --> V
```

Each pipeline run has an overall timeout of 5 minutes. Individual step timeouts are enforced to prevent any single operation from stalling the daemon indefinitely.

**Phase 1 and Phase 2 are independent**: Phase 1 captures complete quickly and enqueue to `analyseQueue` (capacity 5 by default). Phase 2 processes queued results concurrently—if the queue fills, new captures are dropped with a warning log, preventing unbounded memory growth.

## Sequence Diagrams

### Capture Pipeline

The sequence below shows the capture flow triggered by a hotkey press through to message delivery, including error paths for empty results and queue overflow.

```mermaid
sequenceDiagram
    actor User
    participant Listener
    participant Pipeline
    participant Browser as browser.CanvasBounds
    participant Capturer
    participant Vision
    participant Claude as Claude API
    participant Teleprompter
    participant Telegram

    User->>Listener: press trigger hotkey
    Listener->>Pipeline: trigger (buffered channel)

    alt queue full (capture in progress)
        Pipeline->>Pipeline: drop trigger
    else
        Pipeline->>Capturer: ForegroundWindow
        Capturer->>Capturer: AppleScript / System Events
        Capturer-->>Pipeline: WindowInfo

        Pipeline->>Browser: CanvasBounds(appName, x, y, w, h)
        alt recognised browser
            Browser-->>Pipeline: content-area rect (toolbar excluded)
        else unrecognised app
            Browser-->>Pipeline: nil
        end

        Pipeline->>Capturer: CaptureCenter
        Note over Pipeline,Capturer: bounds priority: captureBounds > canvasBounds > full window
        Capturer->>Capturer: CoreGraphics screenshot
        Capturer-->>Pipeline: image RGBA

        Pipeline->>Vision: Extract text
        Vision->>Vision: PNG encode and Vision API
        Vision-->>Pipeline: text

        alt empty OCR text
            Pipeline->>Pipeline: skip
        else
            Pipeline->>Claude: Process text (structured tool call)
            Claude-->>Pipeline: {short, detailed}

            alt empty response
                Pipeline->>Pipeline: skip
            else
                Pipeline->>Teleprompter: Display short answer
                Pipeline->>Telegram: Broadcast detailed answer
                Note over Pipeline,Telegram: split if larger than 4096 chars
                Telegram-->>User: message chunks
            end
        end
    end
```

### Subscription Flow

This sequence shows how Telegram subscribers are managed via `/start` and `/stop` commands, using long-polling to receive updates from the Telegram API.

```mermaid
sequenceDiagram
    actor User
    participant Telegram
    participant Poller
    participant Store

    loop polling
        Poller->>Telegram: getUpdates

        alt start command
            Telegram-->>Poller: user subscribed
            Poller->>Store: add subscriber
            Store->>Store: persist to disk
        else stop command
            Telegram-->>Poller: user unsubscribed
            Poller->>Store: remove subscriber
            Store->>Store: persist to disk
        else error
            Telegram-->>Poller: error
            Poller->>Poller: retry after delay
        end
    end
```

## Subscriber Management

Telegram integration is optional. When `TELEGRAM_BOT_TOKEN` is not set, the daemon runs in teleprompter-only mode using noop implementations for the broadcaster and poller. When enabled, the daemon supports multiple Telegram subscribers through a dynamic subscription system:

- Users send `/start` to the Telegram bot to subscribe and receive responses
- Users send `/stop` to unsubscribe
- The subscriber list is persisted to a plain-text file (default: `tmp/subscribers`), with one chat ID per line, and survives daemon restarts
- All responses are broadcast to every active subscriber

The Telegram poller runs in the background, long-polling for updates with a 30-second timeout and 5-second retry backoff on errors.

## Custom Capture Bounds

Each capture uses the first bounds available in this priority order:

1. **Custom bounds** (highest priority) — set by holding the bounds hotkey (default: `RightOption`) and moving the mouse to define a rectangular region. The daemon tracks minimum and maximum mouse coordinates while the key is held; bounds are locked on release (ignored if the mouse did not move).
2. **Canvas bounds** — when the focused application is a recognised browser (Safari, Chrome, Firefox), the daemon derives the content-area rectangle from the window geometry by subtracting the browser toolbar height (Safari: 74 px, Chrome: 88 px, Firefox: 90 px). This clips the capture to the page content, excluding the address bar and tab bar. Updated automatically on every hotkey press via `browser.CanvasBounds`.
3. **Full window** (fallback) — when neither custom bounds are set nor a browser is recognised, the entire active window is captured.

While the bounds key is held, **arrow keys** reposition the teleprompter overlay: Left/Right rotate through left-center-right alignment. **Minus/plus keys** adjust the text opacity by ±0.025 per step (clamped to 0.0–1.0).

For fullscreen windows (width and height >= screen dimensions), the daemon captures the entire display.

## Teleprompter Overlay

The teleprompter is a stealth macOS overlay window positioned at the bottom of the screen:

- **Excluded from screen sharing** via `NSWindowSharingNone` — invisible to Zoom, QuickTime, and all capture pipelines
- **Excluded from Mission Control, Cmd+Tab, and Dock** via accessory activation policy and collection behavior flags
- **Click-through** — does not intercept mouse events
- **Configurable appearance** — font family, weight, size, opacity, position, adaptive color, and fade duration via environment variables
- **Adaptive text color** — when enabled (`TELEPROMPTER_ADAPTIVE_COLOR=true`), the overlay captures the background behind the text strip on each text update, inverts every pixel via `kCGBlendModeDifference`, and uses the result as the text color pattern so each glyph pixel contrasts with whatever is beneath it. Sampling is event-gated (on `Display(text)` calls, which co-time with the OCR hotkey) rather than periodic, so the macOS screen-recording privacy indicator is only lit during the ~60s debounce window following each hotkey press — the same exposure the OCR capture already incurs
- **Fade animations** — show, hide, and text updates cross-fade with configurable duration (`TELEPROMPTER_FADE_DURATION`). Animation cancellation uses a generation counter to avoid stale completions
- **Runtime repositioning** — hold bounds key + arrow keys to rotate alignment (left/center/right)
- **Runtime opacity adjustment** — hold bounds key + minus/plus keys to decrease/increase text opacity by 0.025 per step. Press 0 to reset to the configured default
- **Toggle visibility** — press the configured toggle hotkey (default: `RightCommand`) to show/hide with fade animation

The AppKit run loop runs on the main OS thread (pinned via `runtime.LockOSThread`). All daemon logic runs in background goroutines. Window operations are dispatched to the main thread via a channel-based work queue pumped at ~60 Hz.

## Key Design Decisions

- **CGEventTap in listen-only mode**: The event tap observes keyboard and mouse events without modifying or consuming them. Other applications continue to receive all events normally. On shutdown, the listener disables the tap and releases all C resources before the goroutine exits.
- **Non-blocking hotkey channel**: The C callback sends to a buffered channel with a non-blocking select, ensuring the `CFRunLoop` is never stalled by a slow pipeline execution.
- **Two-phase pipeline**: Phase 1 captures are unbounded per trigger (responsive to rapid hotkeys). Phase 2 processes queued results concurrently (one goroutine per result) but bounded by `analyseQueue` capacity (5 by default). If the queue is full, captures are dropped with a warning log.
- **In-memory image pipeline**: Images flow as `*image.RGBA` through the pipeline and are PNG-encoded into a byte buffer only when passed to the Vision API. No files are created at any point.
- **Atomic subscriber persistence**: The subscriber store writes to a temporary file and uses `os.Rename()` for atomic updates, protected by a read-write mutex for concurrent access.
- **MarkdownV2 formatting**: All Telegram messages are converted to MarkdownV2 format, escaping special characters for proper rendering.
- **Message chunking**: Telegram's 4096-character message limit is handled by splitting at rune boundaries (respecting Unicode character width) and sending sequential chunks. Rate-limited responses (HTTP 429) are retried with the server-specified backoff.
