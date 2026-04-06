# Requirements

## Daemon Lifecycle

- **CLI interface**: Provide Cobra-based CLI with subcommands for daemon control: `daemon` (run pipeline with IPC), `start` (daemonize), `stop` (shutdown), `status` (check status), `logs` (stream logs), `restart` (stop + start).
- **Daemonization**: Implement safe macOS-compatible process detachment via re-exec with `syscall.SysProcAttr{Setsid: true}`. The parent process exits after child process starts; child runs in a new session fully detached from the terminal.
- **PID file**: Store daemon PID at `$TMPDIR/<binary>-<uid>.pid` for status checks and graceful shutdown. PID file is removed on daemon exit.
- **Startup confirmation**: CLI `start` command polls for PID file to confirm daemon has started within a timeout (default: 3 seconds).
- **IPC socket**: Create Unix domain socket at `$TMPDIR/<binary>-<uid>.sock` with permissions `0600` for inter-process communication.
- **Config flags**: Accept command-line flags on `daemon`, `start`, and `restart` commands (--api-key, --bot-token, --model, --system-prompt, --max-tokens, --log-level, --store-path) and forward them as environment variables to child processes.
- **Graceful shutdown**: Handle `SIGINT` and `SIGTERM` signals from `lensd stop` command or direct signals. Cleanly shut down the event tap, poller, extractor, and IPC server before exiting.

## IPC Communication

- **Log streaming**: Implement `log.subscribe` IPC command that streams slog log events to clients. Parse slog TextHandler output into structured LogEvent structs (time, level, message, attributes) and send to all subscribed clients via fan-out broker.
- **Status query**: Implement `status` IPC command that returns daemon PID, uptime (seconds), last capture time, and last window title. Used by `lensd status` and startup confirmation polling.
- **Shutdown command**: Implement `shutdown` IPC command that gracefully stops the daemon via context cancellation. Used by `lensd stop`.
- **Log colorization**: Apply level-based colors to streamed log output: DEBUG (gray), INFO (cyan), WARN (yellow), ERROR (red/bold). Output to terminal with proper escape sequence handling.

## Hotkey and Capture

- **Global hotkey detection**: Listen for a configurable trigger hotkey system-wide (default: `RightShift`, customizable via `HOTKEY_TRIGGER_KEYNAME`) using MacOS `CGEventTap` in listen-only mode. The event tap runs on a dedicated OS thread with its own `CFRunLoop` and automatically re-enables itself if the system disables it due to timeout or user input.
- **Custom bounds selection**: Track mouse movement while a configurable bounds hotkey is held (default: `RightOption`, customizable via `HOTKEY_BOUNDS_KEYNAME`) to define a custom capture rectangle. Bounds are only recorded when the mouse actually moves and no grid/opacity keys were pressed during the hold session. The bounds persist until the daemon is restarted or new bounds are set.
- **Teleprompter toggle**: Toggle the stealth overlay visibility via a configurable hotkey (default: `RightCommand`, customizable via `HOTKEY_TOGGLE_KEYNAME`). Show and hide transitions use a configurable fade animation. Toggle during a grid move defers the visual change to the move's completion.
- **Teleprompter grid positioning**: While the bounds hotkey is held, arrow keys (Up/Down/Left/Right) move the teleprompter on a percentage-based grid with configurable step (`GRID_STEP`, default 1%) and initial position (`GRID_INITIAL_COL`/`GRID_INITIAL_ROW`, default 0.5). Position wraps circularly. Rapid presses debounce with a configurable delay (`GRID_MOVE_DEBOUNCE_DURATION`, default 300ms).
- **Dynamic text alignment**: `TELEPROMPTER_ALIGNMENT` supports `left`, `center`, `right`, and `dynamic` (default). In dynamic mode, alignment adapts based on grid column position.
- **Teleprompter opacity adjustment**: While the bounds hotkey is held, minus/plus keys decrease/increase text opacity by 0.01 per step (clamped to 0.0–1.0). Press hotkey + 0 to reset to the configured default. Text opacity and overlay visibility (animation) are independent controls.
- **Window evasion**: The teleprompter fades out when the captured window moves or resizes, and restores at the current grid spot after the window stabilises. Polls via `CGWindowListCopyWindowInfo` (metadata only, no screen-capture indicator). Tracks by PID so switching apps does not affect the teleprompter.
- **Active window detection**: Identify the frontmost application window (name, position, size) via AppleScript and `System Events`. Unparseable coordinates in the AppleScript output are treated as errors and surface through the pipeline's non-fatal error path (logged, hotkey listener continues).
- **Screen capture**: Capture the full active window (overlay hidden synchronously during capture), then crop to canvas bounds in Go for browser windows. Use custom bounds if manually set. For fullscreen windows (width >= screen width AND height >= screen height), capture the entire display instead.

## Processing Pipeline

- **OCR text extraction**: Convert the captured image to text using Apple Vision framework. The image is PNG-encoded in memory and passed directly to the Vision API via byte buffer -- no intermediate files touch the disk.
- **AI processing**: Send extracted text to Claude AI using structured tool calls that return `short` (concise answer) and `detailed` (answer + reason) branches. Empty OCR results are silently skipped (no API call made).
- **Teleprompter display**: Show the short answer on a stealth macOS overlay window excluded from screen sharing (`NSWindowSharingNone`). The overlay is configurable via environment variables (font family, weight, size, opacity, position, initial visibility, adaptive color, fade duration). Text updates cross-fade with configurable duration. The teleprompter text is cleared when a new capture is triggered.
- **Adaptive text color**: When enabled, the overlay captures the screen behind the text strip on each text update, inverts every pixel, and uses the result as the text color pattern so each glyph pixel contrasts with the background directly beneath it. Sampling is event-gated (co-timed with the OCR hotkey) to minimize the macOS screen-recording privacy indicator footprint.
- **Telegram delivery** (optional): Broadcast the detailed response to all active subscribers. When `TELEGRAM_BOT_TOKEN` is not set, Telegram is disabled and the daemon runs in teleprompter-only mode. Messages exceeding Telegram's 4096-character limit are automatically split into sequential chunks. Empty AI responses are silently skipped. The HTTP client enforces a 30-second timeout per request; a non-responsive Telegram API will not stall the pipeline indefinitely.
- **Subscriber management**: Support dynamic subscriber registration via Telegram `/start` and `/stop` bot commands. Persist the subscriber list to a plain-text file (one chat ID per line) for durability across restarts.
- **Non-fatal runtime errors**: Pipeline errors (capture failure, OCR failure, API errors) are logged but do not terminate the daemon. The hotkey listener continues running for the next trigger.
