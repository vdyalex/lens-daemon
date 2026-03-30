# lens-daemon Improvements

This document catalogs every concrete improvement identified by comparing lens-daemon against
ghost-daemon, which was built from this codebase and applied better Go practices throughout.
Each item is actionable and traceable to a specific file.

---

## 1. Architecture

### 1.1 Remove `daemon.Options.LogPath` — adopt socket-only logging

**What**: `daemon.Options` carries a `LogPath string` field that optionally redirects the
daemon's stdout/stderr to a file during re-exec (`daemonize.go:48–55`). The field defaults to
empty string → `/dev/null`, so in practice no logs are ever written to disk in normal operation.

**Why this is a problem**:
- Creates a dual output path (file + IPC socket) that is inconsistently documented.
- Any caller that sets `LogPath` bypasses the IPC broker entirely — logs go to a file, not
  to `lensd logs`.
- Violates the single responsibility principle: the IPC broker already owns log distribution.

**Ghost's approach**: `daemon.Options` has no `LogPath`. The daemon's stderr is hardcoded to
`/dev/null`. All log output flows exclusively through the IPC broker. Persistence is the
caller's responsibility: `lensd logs > output.txt`.

**Files to change**:
- `src/daemon/types.go` — remove `LogPath` field
- `src/daemon/daemonize.go` — remove file-open logic (lines 48–55), hardcode `/dev/null`
- `src/cmd/start.go` — remove any `LogPath` population if present
- `src/cmd/daemon.go` — verify `Options{LogPath: ...}` not used

---

### 1.2 Restructure `src/ipc/` into subdirectories

**What**: The `ipc` package is flat — six files coexist in the same directory: `types.go`,
`server.go`, `client.go`, `handler.go`, `log_broker.go`, `frame.go`.

**Why this is a problem**:
- `log_broker.go` is 179 LOC. Combined with `server.go` (140) and `client.go` (123), the
  package is dense and hard to navigate.
- Single package means all symbols are visible to each other — no encapsulation between
  server-side and client-side concerns.
- Adding new IPC commands means touching a shared `handler.go` that also imports server state.

**Ghost's approach**: Four subdirectories, each with a single responsibility:

| Subdirectory | Responsibility |
|---|---|
| `ipc/broker/` | Log fan-out, slog line parser, subscriber management |
| `ipc/handler/` | Command dispatch (status, shutdown, log.subscribe) |
| `ipc/server/` | Unix socket listener, connection lifecycle |
| `ipc/client/` | Dial, send, subscribe streaming |

Shared types (`Request`, `Response`, `LogEvent`, `Handler` interface) stay in `ipc/types.go`.
Protocol primitives (`WriteFrame`, `ReadFrame`) stay in `ipc/frame.go`.

**Files to change** (restructure, not rewrite — logic is identical):
- Create `src/ipc/broker/broker.go` from `src/ipc/log_broker.go`
- Create `src/ipc/handler/handler.go` from `src/ipc/handler.go`
- Create `src/ipc/server/server.go` from `src/ipc/server.go`
- Create `src/ipc/client/client.go` from `src/ipc/client.go`
- Keep `src/ipc/types.go` and `src/ipc/frame.go` at top level
- Update import paths everywhere the old flat packages were imported

---

### 1.3 Extract `DaemonPath()` helper — eliminate path duplication

**What**: The path construction logic `$TMPDIR/lensd-<uid>.<ext>` is duplicated in two places:

- `src/daemon/pid.go:100–107` → `DefaultPIDPath()`
- `src/ipc/server.go:133–140` → `DefaultSocketPath()`

Both contain identical `os.TempDir()` + `user.Current()` + `filepath.Join` logic.

**Why this is a problem**: DRY violation. If the path scheme changes (e.g., adding a
`$XDG_RUNTIME_DIR` fallback or changing the prefix), it must be updated in two places.

**Ghost's approach**: `src/utils/paths/paths.go` exports a single `DaemonPath(extension string) string`
function. `DefaultPIDPath()` and `DefaultSocketPath()` delegate to it:

```go
func DefaultPIDPath() string    { return paths.DaemonPath("pid") }
func DefaultSocketPath() string { return paths.DaemonPath("sock") }
```

**Files to change**:
- Create `src/utils/paths/paths.go` with `DaemonPath(extension string) string`
- `src/daemon/pid.go` — simplify `DefaultPIDPath()` to delegate
- `src/ipc/server.go` (or `src/ipc/server/server.go` after 1.2) — simplify `DefaultSocketPath()`

---

## 2. Code Quality

### 2.1 Split `src/modules/listener/listener.go` (211 LOC → under 200)

**What**: `listener.go` is 211 lines, 11 lines over the 200 LOC limit.

**Why**: The file mixes CGo preamble (C struct setup, callback registration) with Go orchestration
(RunLoop management, channel fan-out). These are two distinct concerns.

**Suggested split**:
- `listener.go` — Go-side orchestration: `New()`, `Listen()`, RunLoop lifecycle (~130 LOC)
- `eventcallback.go` — CGo preamble + `eventCallback` C function setup (~80 LOC)

**Files to change**:
- `src/modules/listener/listener.go` — split off CGo preamble

---

### 2.2 Add GoDoc to `src/main.go` and `src/cmd/restart.go`

**What**: Two files are missing package-level or function-level GoDoc:

- `src/main.go` — no package doc
- `src/cmd/restart.go` — no GoDoc on `runRestart`

**Fix**:

```go
// src/main.go
// Package main is the entry point for the lens/lensd binary.
package main
```

```go
// src/cmd/restart.go
// runRestart stops the running daemon and starts a new one.
// Returns an error if stop or start fail.
func runRestart(cmd *cobra.Command, args []string) error { ... }
```

**Files to change**:
- `src/main.go`
- `src/cmd/restart.go`

---

### 2.3 Adopt ghost's constants naming conventions

**What**: lens-daemon's `constants.go` uses inconsistent naming for related groups. Ghost
establishes clearer prefix patterns:

| Prefix | Scope | Example |
|---|---|---|
| `Default*` | Config defaults | `DefaultBehaviorMinWordsPerMinute` |
| `Timeout*` | All timeouts | `TimeoutIPCClient`, `TimeoutDaemonStartup` |
| `Interval*` | Polling intervals | `IntervalDaemonStartupPoll` |
| `Permission*` | File modes | `PermissionPIDFile`, `PermissionSocket` |
| `IPC*` | IPC protocol | `IPCMaxFrameSize`, `IPCLogSubscriberBuffer` |

Review `constants.go` against this convention and rename any constants that don't follow the
prefix pattern. Avoid abbreviations in constant names.

**Files to change**:
- `src/utils/constants/constants.go`

---

### 2.4 Add CI coverage reporting

**What**: The CI pipeline (`pipeline.yml`) runs `go test` but does not publish a coverage
report or enforce a threshold.

**Ghost's pattern**: `make coverage` generates an HTML report. CI can enforce a minimum using
`go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`.

**Suggested addition** to `pipeline.yml`:
```yaml
- name: Coverage
  run: go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

Optionally gate on a threshold (e.g., 80%) with `grep total coverage.out`.

**Files to change**:
- `.github/workflows/pipeline.yml`

---

## 3. Patterns to Adopt

### 3.1 `waitWithInterrupt` — eliminate repeated select blocks

**What**: In any future event-loop code (e.g., pipeline phases with configurable timeouts),
repeated `select { case <-timer: case <-ctx.Done(): }` patterns should be extracted into a
named helper following ghost's pattern:

```go
// waitWithInterrupt waits for duration, returning early on context cancellation.
// Returns (false, nil) on normal completion, (false, ctx.Err()) if cancelled.
// Uses time.NewTimer to avoid goroutine leaks on early exit.
func waitWithInterrupt(ctx context.Context, duration time.Duration) (bool, error) {
	t := time.NewTimer(duration)
	defer t.Stop()
	select {
	case <-t.C:
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
```

**Where this currently matters in lens**: `pipeline/phases.go` has repeated timeout context
patterns (`ctxWithTimeout, cancel := context.WithTimeout(...)`) that could benefit from a
single helper for consistent cancellation handling.

---

### 3.2 Explicit adapter structs for CGo bridge wrapping

**What**: Ghost wraps its CGo bridge package-level functions in private zero-value structs that
implement interfaces, so the bridge is never directly imported by the engine — only through the
interface:

```go
// pipeline/types.go
type clipboardBridge struct{}

func (clipboardBridge) GetText() (string, bool) { return clipboard.GetText() }
```

This keeps CGo confined to the pipeline wiring layer and makes the engine pure Go and testable.

**Lens already uses interfaces** (`capturer.Service`, `extractor.Service`, etc.) and mocks them.
The improvement is to verify that CGo bridges (`vision.go`, `core_graphics.go`) are only
accessed through adapter structs at the pipeline boundary — never imported directly by modules.

**Files to review**:
- `src/adapters/ocr/ocr.go` — confirm it wraps `bridges/vision` via adapter struct, not direct call
- `src/modules/capturer/capturer.go` — confirm it wraps `bridges/core_graphics` via adapter struct

---

### 3.3 Dual constructor pattern for pipeline testability

**What**: Ghost uses two constructors:
- `New(config, logger)` — wires real dependencies (production path)
- `NewWithDependencies(config, listenerService, ..., logger)` — accepts injected dependencies (test path)

This eliminates the need for build tags or global state replacement in tests.

**Verify lens has this pattern** in `src/pipeline/pipeline.go`. If `New()` builds all dependencies
internally and tests replace them via other means, migrate to the dual-constructor pattern.

**Files to review**:
- `src/pipeline/pipeline.go` — check constructor signature(s)
- `src/pipeline/run_test.go` — check how dependencies are provided in tests

---

## 4. Technical Debt

### 4.1 `src/cmd/start.go:78` — extract sleep constant

**What**: `time.Sleep(10 * time.Millisecond)` is a hardcoded magic number in the daemon startup
polling loop.

**Fix**: Add `IntervalDaemonStartupPoll = 10 * time.Millisecond` to `constants.go` and reference it.

**Files to change**:
- `src/utils/constants/constants.go` — add constant
- `src/cmd/start.go:78` — use constant

---

### 4.2 Document the markdown chunking limitation

**What**: `src/adapters/im/helpers/format.go:25` has a TODO noting that markdown conversion
may break across chunk boundaries. Telegram message chunking can split markdown spans (bold,
italic) across two messages, producing broken markup.

**Fix**: Either address the root cause (chunk after markdown conversion, not before) or promote
the TODO to a `// DEBT:` comment with a concrete explanation of the limitation and its impact.

**Files to change**:
- `src/adapters/im/helpers/format.go:25`

---

## Summary

| # | Category | Change | Files | Effort |
|---|---|---|---|---|
| 1.1 | Architecture | Remove `daemon.Options.LogPath` | daemon/types.go, daemonize.go, cmd/start.go | S |
| 1.2 | Architecture | Restructure `ipc/` into subdirectories | 6 files → 4 subdirs | M |
| 1.3 | Architecture | Extract `DaemonPath()` to `utils/paths/` | pid.go, server.go, new paths.go | S |
| 2.1 | Code quality | Split `listener.go` (211 LOC) | listener.go → listener.go + eventcallback.go | S |
| 2.2 | Code quality | Add missing GoDoc | main.go, cmd/restart.go | XS |
| 2.3 | Code quality | Align constants naming conventions | constants.go | S |
| 2.4 | Code quality | Add CI coverage reporting | .github/workflows/pipeline.yml | XS |
| 3.1 | Pattern | `waitWithInterrupt` helper | pipeline/phases.go or new helper | S |
| 3.2 | Pattern | Verify CGo adapter struct isolation | adapters/ocr, modules/capturer | S |
| 3.3 | Pattern | Verify dual constructor pattern | pipeline/pipeline.go | S |
| 4.1 | Debt | Extract sleep constant | constants.go, cmd/start.go | XS |
| 4.2 | Debt | Document markdown chunking limitation | adapters/im/helpers/format.go | XS |

---

## Verification

After applying all changes:

1. `make build` — compiles cleanly (CGo + Go)
2. `make check` — format, vet, lint, vulnerabilities, tests all pass
3. `lensd start && lensd logs` — daemon starts and log stream works over socket (no file)
4. `lensd start && lensd status` — status command resolves via IPC
5. `lensd logs > /tmp/test.log` — file persistence works via pipe

---

*Document generated by comparing lens-daemon and ghost-daemon codebases. Ghost-daemon was
derived from lens-daemon and applies these improvements. See ghost-daemon source as the
reference implementation.*
