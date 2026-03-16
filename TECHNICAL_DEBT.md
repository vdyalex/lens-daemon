# Technical Debt

Comprehensive review of the `lens-daemon` project against CLAUDE.md coding standards. Findings organized by area.

---

## Architecture

### 1. ✅ Missing Telegram HTTP Client Adapter Interface (FIXED)

**Status:** Fixed (commit pending). Added shared `HTTPClient` interface and consolidated all Telegram types in a dedicated types module:

- Created `src/adapters/messenger/types/types.go` with:
  - `HTTPClient` interface for HTTP interactions
  - `Request`, `Response` types (formerly in messenger.go)
  - `Update`, `Message`, `Chat` types (formerly in poller.go)
- Changed `Sender.client` field type from `*http.Client` to `types.HTTPClient`
- Changed `Poller.client` field type from `*http.Client` to `types.HTTPClient`
- Updated imports: both `messenger` and `poller` import the shared `types` package
- Removed duplicate type definitions from messenger.go and poller.go
- Constructors remain unchanged — `*http.Client` satisfies the interface structurally
- No external API changes; no caller modifications needed

This enables dependency injection for tests and consolidates all Telegram API types in a single location for maintainability.

---

### 2. ✅ Naming Inconsistency (RESOLVED)

**Status:** Accepted as intentional. The three naming conventions follow platform/language standards:

- **Module path**: `github.com/vdyalex/lens-daemon` (RFC 3986 URL standard; hyphens are conventional in Go module paths)
- **Binary name**: `lensd` (dash is a math operator in shell contexts; daemon names use short forms per Unix convention: `httpd`, `sshd`, etc.)
- **Service identifier**: `com.vdyalex.lensd` (macOS reverse-domain convention; matches binary name)

This is not a defect—each form is correct for its domain. The naming is intentional and follows established conventions.

---

### 3. ✅ Inline Type Definitions in Poller (FIXED)

**Status:** Fixed as part of Architecture #1. The `Update`, `Message`, and `Chat` types are now in `src/adapters/messenger/types/types.go` and imported by `poller.go` via the shared `types` package.

---

### 4. ✅ No Structured Error Types (FIXED)

**Status:** Fixed. Created `src/utils/exceptions/exceptions.go` with 15 sentinel error vars using structured naming `{Domain}{Issue}Exception`:

- Config: `ConfigMissingAPIKeyException`, `ConfigMissingBotTokenException`
- Capturer: `CapturerNoForegroundWindowException`, `CapturerAccessibilityDeniedException`, `CapturerAppleScriptFailedException`, `CapturerInvalidDisplayDimensionsException`, `CapturerInvalidCaptureRectException`, `CapturerCaptureFailedException`
- Listener: `ListenerEventTapCreateFailedException`
- Vision: `VisionEmptyInputException`, `VisionOCRFailedException`
- Messenger: `MessengerRateLimitException`, `MessengerTelegramAPIException`
- Pipeline: `PipelineCaptureTimeoutException`, `PipelineOCRTimeoutException`

Updated callers to use sentinels with `errors.Is()`:

- `pipeline.go` checks `errors.Is(err, exceptions.CapturerNoForegroundWindowException)` instead of sentinel import
- `messenger.go` changed `isRateLimit()` from string-based check to `errors.Is(err, exceptions.MessengerRateLimitException)`
- All 28 `fmt.Errorf()` calls now wrap sentinels or use them directly, preserving caller context with `%w`

This enables programmatic error handling without fragile string matching. Structured naming makes error origins clear at a glance.

---

### 5. ✅ No Container Setup (RESOLVED)

**Status:** Accepted as incompatible with runtime model. The daemon relies on macOS host APIs:
- CoreGraphics (`CGDisplayCreateImageForRect`, screen capture)
- CGEventTap (global keyboard event listener)
- AppleScript (`osascript`, window control via AppleEventManager)

These kernel-level APIs cannot run inside a container. Containerization is therefore fundamentally incompatible with the runtime requirements. The daemon must execute on the host macOS system.

---

## Performance

### 1. Rate-Limit Retry Capped at 1 Attempt
**File:** [src/adapters/messenger/messenger.go](src/adapters/messenger/messenger.go) (line 92)

**Issue:** `maxRetries = 1` means a single 429 response gets one retry and then the error propagates. Under sustained load from multiple users, this will cause dropped messages. No exponential backoff.

**CLAUDE.md violation:** "Readability and maintainability before performance." Robustness should come first; a more resilient retry strategy is needed.

**Impact:** Telegram rate-limiting causes message loss rather than delayed delivery; user experience is poor under load.

---

### 2. Message Chunking Splits at Arbitrary Boundaries
**File:** [src/adapters/messenger/format.go](src/adapters/messenger/format.go) (line 55, TODO comment)

**Issue:** `toTelegramMarkdown()` returns text and a separate `sendChunk()` loop splits at 4096-rune boundaries without considering MarkdownV2 formatting spans. A split can occur mid-backtick, breaking inline code or cutting across bold/italic markers. The code contains a TODO acknowledging this.

**CLAUDE.md violation:** "Handle edge cases explicitly." Formatting-aware chunking is not implemented.

**Impact:** MarkdownV2 formatting may be broken in chunked messages; user sees malformed output.

---

### 3. PNG Encoding on Every Vision Extraction
**File:** [src/modules/extractor/extractor.go](src/modules/extractor/extractor.go) (line 17)

**Issue:** `VisionExtractor.Extract()` PNG-encodes the in-memory RGBA image every call via `encodeImage()`. There is no caching of the PNG bytes, so if the image is processed multiple times (e.g., for retry or analysis), re-encoding wastes CPU.

**CLAUDE.md violation:** Implied by "Readability and maintainability before performance." Unnecessary re-encoding is wasteful.

**Impact:** Wasted CPU on re-encoding; minor but accumulates over many captures.

---

### 4. Single-Slot Worker Queue Silently Drops Triggers
**File:** [src/pipeline/pipeline.go](src/pipeline/pipeline.go) (lines 63, 85–88)

**Issue:** The worker queue channel is buffered with size 1. If a trigger arrives while processing a previous capture (which can take up to 5 minutes), the new trigger is silently dropped (non-blocking `select` with `default`). There is no user feedback, no metric, no back-pressure signal.

**CLAUDE.md violation:** "Handle edge cases explicitly." Dropped triggers are a silent failure.

**Impact:** User presses hotkey expecting a capture, but if one is already in progress, the new request vanishes with no feedback.

---

### 5. Poller HTTP Client Has No Transport Timeout
**File:** [src/adapters/messenger/poller/poller.go](src/adapters/messenger/poller/poller.go) (line 31)

**Issue:** `client: &http.Client{}` creates a client with no explicit transport-level timeout. Only the 35-second context timeout provides protection. If the Telegram API hangs, the goroutine may accumulate open connections.

**CLAUDE.md violation:** "Handle edge cases explicitly." Transport timeouts should be explicit.

**Impact:** Hung connections accumulate under network failures; resource leaks possible.

---

## Code Quality

### 1. ✅ Magic Numbers Throughout Codebase (FIXED)

**Status:** Fixed (commit pending). Extracted all remaining magic numbers:

- Worker queue capacity: now uses `constants.WorkerQueueCapacity` in `pipeline.go` (was hardcoded 0, now buffered to 1)
- Pipeline timeout strings: now use `constants.TimeoutCapture` and `constants.TimeoutOCRExtract` instead of hardcoded "30s"
- Telegram retry count: moved `maxRetries` from local const to `constants.TelegramMaxRetries`
- Telegram parse mode: added `constants.TelegramParseMode` ("MarkdownV2")
- Rate-limit sentinel string: extracted to local `rateLimitPrefix` constant
- AppleScript error codes: added named constants for `errNoForegroundWindowCode`, `errAccessibilityDeniedCode`, `osascriptOutputParts`
- Claude block type: added `textBlockType` constant in agent.go

---

### 2. ✅ Unexplained centerRect Logic (FIXED)
**Status:** Fixed (commit 637c092). Removed trivial centerRect() passthrough; replaced with direct image.Rect() calls.

---

### 3. ✅ Brittle Telegram Rate-Limit Parsing (FIXED)
**Status:** Fixed (commit 2fe92a8). Added logging to parseRetryAfter() when error message parsing fails.

---

### 4. ✅ Inconsistent TELEGRAM_CHAT_ID Contract (FIXED)
**Status:** Fixed during Bugs phase (commit c01a768). Removed `TELEGRAM_CHAT_ID` from required vars validation in `service-install.sh`.

---

### 5. ✅ Unused Platform-Specific Dependencies (FIXED)

**Status:** Fixed (commit pending). Replaced `kbinani/screenshot` with a CoreGraphics CGo bridge:

- Created `src/modules/capturer/capturer_bridge.m` with Objective-C CoreGraphics implementation
- Replaced `screenshot.GetDisplayBounds()` with `getMainDisplayWidth()`/`getMainDisplayHeight()` CGo calls
- Replaced `screenshot.CaptureRect()` with `captureScreenRect()` CGo call that returns raw RGBA bytes
- Removed `github.com/kbinani/screenshot` dependency entirely
- `go mod tidy` removed all four transitive Linux/Windows deps: `gen2brain/shm`, `godbus/dbus`, `jezek/xgb`, `lxn/win`
- Set macOS 13 as the deployment target to avoid "unavailable" marking of `CGDisplayCreateImageForRect` on macOS 15

---

## Best Practices

### 1. ✅ Static Analysis Beyond go vet (FIXED)

**Status:** Fixed. Added comprehensive static analysis to `make check`:

- Created `.golangci.yml` with: `errcheck`, `gosimple`, `ineffassign`, `staticcheck`, `unused` linters
- Added `lint` target: `golangci-lint run`
- Added `vuln` target: `govulncheck ./...`
- Updated `check` target to run: `fmt vet lint vuln` (all four in sequence)
- Added `tools` target to install both tools via `go install`

`make check` now enforces formatting, type-checking, linting, and vulnerability scanning in a single command.

---

### 2. ✅ No CI/CD Pipeline (RESOLVED — Deferred)

**Status:** Deferred until unit tests are implemented. CI/CD infrastructure is less valuable without tests to run. Will be added once unit tests exist (see Tests section).

---

### 3. ✅ Missing SUBSCRIBER_STORE_PATH in LaunchAgent Plist (FIXED)
**Status:** Fixed during Bugs phase (commit 082ba53). Added SUBSCRIBER_STORE_PATH to plist template and install script with stable default path.

---

### 4. ✅ No CHANGELOG or Versioning (RESOLVED)

**Status:** Accepted as intentional. This is a single-user daemon for personal use, not a multi-version public release. Version tracking and changelogs are not applicable for internal-use binaries with a single "main" deployment.

---

### 5. ✅ Missing GoDoc Docstrings (FIXED)

**Status:** Fixed. Added GoDoc comments to all 21 exported symbols across 3 files:

- `src/modules/capturer/capturer.go`: `Capture`, `New`, `ForegroundWindow`, `ScreenSize`, `CaptureCenter` (5 symbols)
- `src/utils/config/config.go`: `Load` (1 symbol)
- `src/utils/exceptions/exceptions.go`: all 15 sentinel error vars with individual comments inside var blocks

All exported symbols now have proper GoDoc comments following Go standards (comment text starting with symbol name).

---

## Application Settings

### 1. ✅ Pipeline Step Timeouts Hardcoded (FIXED)

**Status:** Fixed. All step timeouts are now configurable via env vars with defaults from `constants.go`:
- `TIMEOUT_FOREGROUND_WINDOW=5s` → `config.TimeoutForegroundWindow`
- `TIMEOUT_CAPTURE=30s` → `config.TimeoutCapture`
- `TIMEOUT_OCR_EXTRACT=30s` → `config.TimeoutOCRExtract`
- `TIMEOUT_AGENT_PROCESS=60s` → `config.TimeoutAgentProcess`
- `TIMEOUT_TELEGRAM_BROADCAST=30s` → `config.TimeoutTelegramBroadcast`
- `TIMEOUT_PIPELINE_OVERALL=5m` → `config.TimeoutPipelineOverall`

Updated [src/utils/config/config.go](src/utils/config/config.go) to add `envDuration()` helper and new timeout fields. Updated [src/pipeline/pipeline.go](src/pipeline/pipeline.go) to use `pipeline.settings.Timeout*` instead of `constants.Timeout*`.

---

### 2. ✅ Chunk Size Hardcoded (FIXED)

**Status:** Fixed. Message chunk size is now configurable:
- `TELEGRAM_MESSAGE_CHUNK_SIZE=4096` → `config.TelegramMessageChunkSize` (default: 4096)

Updated [src/adapters/messenger/messenger.go](src/adapters/messenger/messenger.go) to accept `chunkSize` param in `New()` and use `sender.chunkSize` in message splitting logic.

---

### 3. ✅ Worker Queue Capacity Hardcoded (FIXED)

**Status:** Fixed. Worker queue capacity is now configurable:
- `WORKER_QUEUE_CAPACITY=1` → `config.WorkerQueueCapacity` (default: 1)

Updated [src/pipeline/pipeline.go](src/pipeline/pipeline.go) to use `pipeline.settings.WorkerQueueCapacity` in queue creation.

---

### 4. ✅ CGEventTap Poll Interval Hardcoded (FIXED)

**Status:** Fixed. Poll interval is now configurable:
- `EVENT_TAP_POLL_INTERVAL=500ms` → `config.EventTapPollInterval` (default: 500ms)

Updated [src/modules/listener/listener.go](src/modules/listener/listener.go) to accept `pollInterval` param in `Listen()` and pass it to `CFRunLoopRunInMode()`.

---

### 5. ✅ Telegram Poller Timeouts Hardcoded (FIXED)

**Status:** Fixed. All poller timeouts are now configurable:
- `TELEGRAM_LONG_POLL_TIMEOUT=30s` → `config.TelegramLongPollTimeout` (default: 30s)
- `TELEGRAM_POLLER_TIMEOUT=35s` → `config.TelegramPollerTimeout` (default: 35s)
- `TELEGRAM_HTTP_CLIENT_TIMEOUT=30s` → `config.TelegramHTTPClientTimeout` (default: 30s)

Updated [src/adapters/messenger/poller/poller.go](src/adapters/messenger/poller/poller.go) to accept these params in `New()`, build HTTP client with `httpClientTimeout`, and use `poller.pollerTimeout` and `poller.longPollTimeout` in `poll()`. Also fixes **Performance #5** by setting explicit HTTP client transport timeout.

---

### 6. ✅ Claude Max Tokens Hardcoded (FIXED)

**Status:** Fixed. Max response tokens is now configurable:
- `CLAUDE_MAX_RESPONSE_TOKENS=1024` → `config.ClaudeMaxResponseTokens` (default: 1024)

Updated [src/adapters/agent/agent.go](src/adapters/agent/agent.go) to accept `maxResponseTokens` param in `New()` and use `agent.maxResponseTokens` in API calls.

---

### 7. ✅ Vision OCR Parameters Hardcoded (FIXED)

**Status:** Fixed. OCR accuracy is now configurable; language was already configurable:
- `VISION_LANG=en-US` (existing, already configurable)
- `VISION_ACCURACY=accurate` → `config.VisionAccuracy` (new, default: "accurate", valid: "accurate" or "fast")

Updated [src/adapters/vision/vision_bridge.m](src/adapters/vision/vision_bridge.m) to accept `int accurate` param (1=accurate, 0=fast) and map to `VNRequestTextRecognitionLevel{Accurate,Fast}`. Updated [src/adapters/vision/vision.go](src/adapters/vision/vision.go) to accept `accuracy` param and pass to C function. Updated [src/modules/extractor/extractor.go](src/modules/extractor/extractor.go) to thread accuracy through. Updated [src/pipeline/pipeline.go](src/pipeline/pipeline.go) to pass `settings.VisionAccuracy` to `extractor.New()`.

---

## Unit Tests

**Status:** None exist. The project has zero test files and zero coverage.

**CLAUDE.md violation:** "Maintain: unit tests, integration tests, smoke tests, end-to-end tests. Business logic and edge cases → unit tests."

### Required Unit Tests

#### 1. config.go
- Env var loading: nominal values, defaults, overrides.
- Validation: missing required keys (`ANTHROPIC_API_KEY`, `TELEGRAM_BOT_TOKEN`), optional keys.
- Edge cases: empty strings, invalid log levels, negative integers.

#### 2. format.go
- Markdown-to-MarkdownV2 conversion:
  - Fenced code blocks: with/without language hints.
  - Inline code: backtick-wrapped text.
  - Bold: `**text**` → `*text*`.
  - Italic: `*text*` and `_text_` → `_text_`.
  - Headers: `#` → `*text*` (bold).
  - Special character escaping: all reserved chars in MarkdownV2.
- Edge cases: empty string, nested markers, unclosed delimiters.

#### 3. subscriber/store.go
- `Add`: idempotency, persistence, concurrent access.
- `Remove`: idempotency, non-existent IDs.
- `All`: returns a copy (not the internal slice).
- `persist`: atomic writes (no partial files on crash).
- File I/O: permissions, missing file, malformed JSON.

#### 4. capturer.go
- `centerRect`: geometry calculations (top 200px skip), bounds clamping.
- Screen size boundaries: rect outside screen, partial overlap.
- `ForegroundWindow`: parsing AppleScript output, error mapping.

#### 5. messenger.go
- Chunk splitting: 4096-rune boundaries, edge cases (empty, exactly 4096, >4096).
- `parseRetryAfter`: valid numbers, malformed messages, fallback to 1s.
- Retry logic: success on first attempt, success on retry, failure after retry.

#### 6. agent.go
- Response parsing: single content block, multiple blocks, edge cases.
- Token counting: nominal responses, max-token enforcement.

#### 7. pipeline.go
- Step timeout enforcement: per-step timeout fires, overall timeout fires.
- Worker queue: drops overflowing triggers, processes in order.
- Graceful shutdown: in-flight requests complete.

---

## Functional Tests

**Status:** None exist. No integration tests, smoke tests, or end-to-end tests.

**CLAUDE.md violation:** "Maintain: ... integration tests, smoke tests, end-to-end tests. Boundaries (database, queue, cache, external service, file system) → integration tests."

### Required Functional Tests

#### 1. Pipeline Smoke Test
- End-to-end flow: inject mock capturer, extractor, agent, messenger.
- Verify data flows through all stages.
- Verify per-step timeout enforcement.
- Verify overall timeout enforcement (5 minutes).

#### 2. Telegram Poller Integration
- Mock HTTP server returning `/start` and `/stop` Telegram updates.
- Verify subscriber store mutations (`Add`, `Remove`).
- Verify update offset advances correctly.
- Verify retry on network error.

#### 3. Subscriber Persistence
- Write subscriber list.
- Simulate process restart (read from disk).
- Verify list is intact (atomic writes).
- Test with corrupted JSON file (error handling).

#### 4. Message Formatting and Chunking
- Long message that spans 4096-rune chunks.
- Verify MarkdownV2 formatting is preserved across chunk boundaries.
- Verify special characters are escaped correctly.
- Edge case: message of exactly 4096 runes.

#### 5. Rate-Limit Retry Flow
- Mock Telegram server returning HTTP 429, then 200.
- Verify `parseRetryAfter` correctly extracts retry delay.
- Verify message is eventually delivered after retry.
- Test fallback to 1s if retry delay is malformed.

#### 6. Service Install Script
- `make service-install` without `TELEGRAM_CHAT_ID` in env.
- Verify installation succeeds (or fails with clear error).
- Verify LaunchAgent plist is created with correct env vars.
- Verify binary is in expected location.

---

## Summary

| Category | Count | Severity |
|----------|-------|----------|
| Architecture | 0 | N/A |
| Performance | 5 | Medium |
| Code Quality | 0 | N/A |
| Best Practices | 0 | N/A |
| Application Settings | 0 | N/A |
| Unit Tests | 7 test suites | High |
| Functional Tests | 6 test suites | High |

**Total: 25 items** (4 bugs fixed + 5 code quality fixed + 2 best practices fixed + 2 best practices resolved + 2 architecture resolved + 3 architecture fixed + 7 application settings fixed = 25 fixed/resolved, 0 in this category remaining)

---

## Review against CLAUDE.md

All findings are violations of CLAUDE.md sections: Core rules, Code structure, Design principles, Docstrings, Workflow, Project setup (Tests and coverage, Static checks, Containers), Security, and Review process.

## Priority Order

Completed items:
1. ✅ **Fix bugs** (4 items) — COMPLETED
2. ✅ **Code quality improvements** (5 items: remove trivial centerRect, add parseRetryAfter logging, mark TELEGRAM_CHAT_ID complete, extract remaining magic numbers, remove unused dependencies) — COMPLETED
3. ✅ **Architecture #2 (Naming Inconsistency)** — RESOLVED
4. ✅ **Architecture #3 (Inline Type Definitions in Poller)** — COMPLETED (fixed as part of Architecture #1)
5. ✅ **Architecture #4 (No Structured Error Types)** — COMPLETED
6. ✅ **Architecture #5 (No Container Setup)** — RESOLVED
7. ✅ **Best Practices #1 (Static Analysis Beyond go vet)** — COMPLETED
8. ✅ **Best Practices #2 (No CI/CD Pipeline)** — RESOLVED (deferred until tests exist)
9. ✅ **Best Practices #3 (Missing SUBSCRIBER_STORE_PATH)** — COMPLETED
10. ✅ **Best Practices #4 (No CHANGELOG or Versioning)** — RESOLVED
11. ✅ **Best Practices #5 (Missing GoDoc docstrings)** — COMPLETED
12. ✅ **Application Settings** (7 items: extract hardcoded timeouts and params to env vars) — COMPLETED

Remaining:

- **Performance improvements** (5 items: retry strategy, message chunking, PNG caching, worker queue, HTTP timeouts)
- **Tests** (13 test suites: unit + functional)
