# Requirements

## Daemon Lifecycle

- **CLI interface**: Provide Cobra-based CLI with subcommands for daemon control: `daemon` (run pipeline with IPC), `start` (daemonize), `stop` (shutdown), `status` (check status), `logs` (stream logs), `restart` (stop + start).
- **Daemonization**: Implement safe macOS-compatible process detachment via re-exec with `syscall.SysProcAttr{Setsid: true}`. The parent process exits after child process starts; child runs in a new session fully detached from the terminal.
- **PID file**: Store daemon PID at `$TMPDIR/lensd-<uid>.pid` for status checks and graceful shutdown. PID file is removed on daemon exit.
- **Startup confirmation**: CLI `start` command polls for PID file to confirm daemon has started within a timeout (default: 5 seconds).
- **IPC socket**: Create Unix domain socket at `$TMPDIR/lensd-<uid>.sock` with permissions `0600` for inter-process communication.
- **Config flags**: Accept command-line flags on `daemon`, `start`, and `restart` commands (--api-key, --bot-token, --model, --system-prompt, --max-tokens, --log-level) and forward them as environment variables to child processes.
- **Graceful shutdown**: Handle `SIGINT` and `SIGTERM` signals from `lensd stop` command or direct signals. Cleanly shut down the event tap, poller, extractor, and IPC server before exiting.

## IPC Communication

- **Log streaming**: Implement `log.subscribe` IPC command that streams slog log events to clients. Parse slog TextHandler output into structured LogEvent structs (time, level, message, attributes) and send to all subscribed clients via fan-out broker.
- **Status query**: Implement `status` IPC command that returns daemon PID, uptime (seconds), last capture time, and last window title. Used by `lensd status` and startup confirmation polling.
- **Shutdown command**: Implement `shutdown` IPC command that gracefully stops the daemon via context cancellation. Used by `lensd stop`.
- **Log colorization**: Apply level-based colors to streamed log output: DEBUG (gray), INFO (cyan), WARN (yellow), ERROR (red/bold). Output to terminal with proper escape sequence handling.

## Hotkey and Capture

- **Global hotkey detection**: Listen for a configurable trigger hotkey system-wide (default: `RightShift`, customizable via `HOTKEY_TRIGGER_KEYNAME`) using MacOS `CGEventTap` in listen-only mode. The event tap runs on a dedicated OS thread with its own `CFRunLoop` and automatically re-enables itself if the system disables it due to timeout or user input.
- **Custom bounds selection**: Track mouse movement while a configurable bounds hotkey is held (default: `RightOption`, customizable via `HOTKEY_BOUNDS_KEYNAME`) to define a custom capture rectangle. The bounds persist until the daemon is restarted or new bounds are set.
- **Active window detection**: Identify the frontmost application window (name, position, size) via AppleScript and `System Events`. Unparseable coordinates in the AppleScript output are treated as errors and surface through the pipeline's non-fatal error path (logged, hotkey listener continues).
- **Screen capture**: Capture the entire active window, or use custom bounds if set. For fullscreen windows (width >= screen width AND height >= screen height), capture the entire display instead.

## Processing Pipeline

- **OCR text extraction**: Convert the captured image to text using Apple Vision framework. The image is PNG-encoded in memory and passed directly to the Vision API via byte buffer -- no intermediate files touch the disk.
- **AI processing**: Send extracted text to Claude AI with a configurable system prompt. Empty OCR results are silently skipped (no API call made).
- **Telegram delivery**: Broadcast Claude's response to all active subscribers. Messages exceeding Telegram's 4096-character limit are automatically split into sequential chunks. Empty AI responses are silently skipped. The HTTP client enforces a 30-second timeout per request; a non-responsive Telegram API will not stall the pipeline indefinitely.
- **Subscriber management**: Support dynamic subscriber registration via Telegram `/start` and `/stop` bot commands. Persist the subscriber list to a plain-text file (one chat ID per line) for durability across restarts.
- **Non-fatal runtime errors**: Pipeline errors (capture failure, OCR failure, API errors) are logged but do not terminate the daemon. The hotkey listener continues running for the next trigger.
