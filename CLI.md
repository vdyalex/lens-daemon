# CLI: Cobra CLI + Detached Daemon Architecture

## Status: ✅ Complete

**Last Updated**: 2026-03-17
**Status**: Production ready — all features implemented, tested, and documented
**Overall Progress**: 100% complete

---

## Context

The previous implementation used a BubbleTea TUI for configuration and log viewing. This proved
overly complex and difficult to use. The goal is to replace it with a simple, scriptable Cobra CLI:

- Configuration is passed as flags on `start`/`daemon` commands (no TOML file)
- Logs are streamed to stdout via `lensd logs` using IPC (same broker, simpler consumer)
- Status is a one-shot command printing a formatted message
- The daemon lifecycle (start/stop/restart) remains unchanged

---

## Architecture: Single Binary, Multiple Subcommands

```
lensd daemon   — run pipeline (used by LaunchAgent); accepts config flags
lensd start    — daemonize (re-exec "lensd daemon" detached); accepts config flags
lensd stop     — send SIGTERM via PID file
lensd status   — probe PID file + IPC, print one-shot status message
lensd logs     — subscribe to IPC log stream; print level-colorized lines until Ctrl-C
lensd restart  — stop + start; accepts config flags
```

### Why re-exec daemonize (not double-fork)

macOS + CGo makes traditional POSIX `fork()` unsafe (Go runtime panics in child). Re-exec with
`syscall.SysProcAttr{Setsid: true}` is the canonical macOS-safe approach. The parent exits after
`cmd.Start()`; the child (`lensd daemon`) runs in a new session, fully detached from the terminal.

Config flags are forwarded by setting them as environment variables on the child process before
re-exec, so the daemon picks them up via the existing `config.Load()` env-var layer.

---

## IPC: Unix Domain Socket + Length-Prefixed JSON

Unchanged from current implementation. Only the commands still in use are kept.

- Socket at `$TMPDIR/lensd-<uid>.sock`
- Wire format: 4-byte big-endian length prefix + UTF-8 JSON body
- Permissions: `0600`

**Commands (trimmed):**

| Command | Purpose | Used by |
|---|---|---|
| `status` | PID, uptime, last capture, subscriber count | `lensd status` |
| `shutdown` | Graceful stop | `lensd stop` |
| `log.subscribe` | Stream log events (server-push) | `lensd logs` |

`config.get` and `config.set` are removed — no caller remains after TUI removal.

---

## Config Management

**Load precedence:** CLI flags (passed as env vars to daemon) > env vars > `.env` > compiled defaults

No TOML file. No live config reload. All config is set at daemon start time.

**Cobra flags on `daemon` / `start` / `restart`:**

| Flag | Env var mapped | Description |
|---|---|---|
| `--model` | `ANTHROPIC_MODEL` | Anthropic model name |
| `--system-prompt` | `ANTHROPIC_SYSTEM_PROMPT` | AI system prompt |
| `--max-tokens` | `ANTHROPIC_MAX_RESPONSE_TOKENS` | Max response tokens |
| `--log-level` | `LOG_LEVEL` | debug / info / warn / error |
| `--api-key` | `ANTHROPIC_API_KEY` | Anthropic API key |
| `--bot-token` | `TELEGRAM_BOT_TOKEN` | Telegram bot token |

`start` passes set flags as explicit env vars in `daemon.Options.ExtraEnv` before re-exec.

---

## File Changes

### Files Deleted

```
src/tui/                          — entire directory (BubbleTea panels, models, views) ✅
src/cmd/tui.go                    — BubbleTea entry point ✅
src/utils/config/file.go          — TOML load/save ✅
src/utils/config/patch.go         — live-config patch apply ✅
src/utils/config/patch_test.go    — tests for deleted file ✅
```

### Files Added

```
src/cmd/logs.go     — IPC log.subscribe consumer; uses slog with colorization ✅
```

### Files Modified

| File | Change | Status |
|---|---|---|
| `src/cmd/root.go` | Cobra root command with `SilenceUsage: true` | ✅ |
| `src/cmd/daemon.go` | Cobra command with config flags applied via `applyFlags()` | ✅ |
| `src/cmd/start.go` | Cobra command, forwards flags to daemon via `ExtraEnv` | ✅ |
| `src/cmd/stop.go` | Cobra command wrapper | ✅ |
| `src/cmd/status.go` | Cobra command, one-shot IPC query + formatted output | ✅ |
| `src/cmd/restart.go` | Cobra command, calls `runStop()` then `runStart()` | ✅ |
| `src/ipc/handler.go` | Removed `config.get`/`config.set` dispatch | ✅ |
| `src/ipc/protocol.go` | Removed `CommandConfigGet/Set`, `ConfigSetPayload` | ✅ |
| `src/ipc/log_broker.go` | Enhanced parser: proper escape sequence handling (`\n`, `\t`, etc.) | ✅ |
| `src/pipeline/pipeline.go` | Removed `SetLiveConfig()` method | ✅ |
| `src/adapters/ai/ai.go` | Removed `SetSystemPrompt()`, `SetModel()`, `SetMaxTokens()` | ✅ |
| `src/adapters/ai/types.go` | Removed `LiveUpdater` interface, `sync.RWMutex` field | ✅ |
| `src/utils/config/config.go` | Removed TOML load layer, simplified to env vars + .env | ✅ |
| `src/utils/config/types.go` | Removed `Patch` struct | ✅ |
| `src/main.go` | Call `cmd.Execute()` instead of `cmd.Dispatch(os.Args)` | ✅ |
| `Makefile` | Removed `tui` target; added `logs` target; `run` accepts args | ✅ |

---

## Dependencies

### Removed ✅

| Dependency | Reason |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI removed |
| `github.com/charmbracelet/bubbles` | TUI removed |
| `github.com/charmbracelet/lipgloss` | TUI removed |
| `github.com/BurntSushi/toml` | TOML config removed |

### Added ✅

| Dependency | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework with structured subcommand dispatch |
| `github.com/fatih/color` | Terminal log colorization in `lensd logs` |

---

## `lensd logs` Implementation

### Behavior

- Dials IPC socket, sends `log.subscribe` request
- Streams `LogEvent` frames until Ctrl-C or daemon disconnect
- Uses `slog.TextHandler` with custom `ColoredLogWriter` for output
- Applies level-based colors to slog text output:
  - DEBUG → gray (faint white)
  - INFO → cyan
  - WARN → yellow
  - ERROR → red (bold)
- Properly interprets escape sequences in messages: `\n` → newline, `\t` → tab, etc.
- Logs retain full slog format with timestamps and attributes

### Log Output Format

```
time=2026-03-17T15:04:05.123Z level=INFO msg="lensd started" pid=12345 component=daemon
time=2026-03-17T15:04:05.456Z level=DEBUG msg="pipeline running" component=capturer
```

All slog-formatted output is preserved, with level-based colorization applied by `ColoredLogWriter`.

---

## Log Parser Enhancements

The `log_broker.go` parser now correctly handles:

1. **Quoted values with spaces** — `msg="hello world"` is parsed as a single message
2. **Escape sequences** — `\n` is converted to actual newline, `\t` to tab, `\\` to backslash, etc.
3. **Trailing whitespace** — slog output lines are trimmed before parsing
4. **Multiline messages** — messages with `\n` escapes are displayed across multiple terminal lines
5. **All slog attributes** — non-standard key=value pairs are preserved in `event.Attrs`

---

## Usage Examples

### Basic Commands

```bash
# Build
make build

# Start daemon with custom model
make run start --model claude-opus-4-6

# Check status
make run status
# → Daemon running | PID 12345 | Uptime 2m34s | Last window: Xcode

# Stream logs
make run logs
# → time=2026-03-17T15:04:05.123Z level=INFO msg="lensd started" pid=12345

# Restart with new config
make run restart --log-level debug

# Stop
make run stop
```

### Direct Binary Calls

```bash
./bin/lensd start --api-key YOUR_KEY --bot-token YOUR_TOKEN
./bin/lensd status
./bin/lensd logs
./bin/lensd stop
```

### Make Targets with Arguments

The `Makefile` supports passing arguments directly via `make run`:

```bash
make run logs           # Stream logs
make run status         # Check status
make run start          # Start daemon
make run restart        # Restart daemon
make run start --model claude-haiku-4-5-20251001  # Start with flags
```

**Note:** `make build` only compiles the binary and does not start the daemon. To ensure a clean rebuild when switching configurations, use:

```bash
make clean-daemon build   # Stop any running daemon, then build
make run start            # Start with new configuration
```

---

## Improvements Over Previous Implementation

| Aspect | BubbleTea TUI | Cobra CLI |
| --- | --- | --- |
| Configuration | Interactive form in TUI | Command-line flags |
| Log viewing | Scrollable viewport with ring buffer | Streaming to stdout (scriptable) |
| Status display | 5s auto-refresh with dashboard | One-shot command |
| Dependencies | 4 new charmbracelet libs (bulky) | 2 focused libs (cobra, color) |
| Scriptability | Not pipeline-friendly | Unix-friendly, piping support |
| Binary size | ~13 MB | ~13 MB (color lib is small) |
| Testing | Complex TUI testing | Standard CLI testing |
| Maintenance | TUI state machine complexity | Straightforward command dispatch |

---

## Tests

All existing tests pass:

```bash
make test           # 35+ unit tests passing
make test-integration  # Daemon + IPC integration tests
make check          # Full suite (format, lint, vuln scan, tests)
```

---

## Known Limitations

- No live config reload (all config is set at daemon start)
- Log streaming doesn't persist history (use `-o logfile` redirection if needed)
- Status is a point-in-time snapshot, not a dashboard

These are acceptable trade-offs for simplicity and scriptability.

---

## Implementation Complete ✅

All tasks finished:
1. ✅ Removed BubbleTea + TOML infrastructure
2. ✅ Added Cobra CLI framework
3. ✅ Implemented 6 subcommands with proper flag handling
4. ✅ Implemented `lensd logs` with slog + colorization
5. ✅ Fixed log parser for escape sequences and multiline messages
6. ✅ Updated Makefile for flexible argument passing (`make run <command>`)
7. ✅ Silenced help output (only on explicit `--help` or unknown command)
8. ✅ Added `make clean-daemon` target to stop running daemon before build
9. ✅ All tests passing, clean build, production-ready
