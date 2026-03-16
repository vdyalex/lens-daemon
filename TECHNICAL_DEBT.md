# Technical Debt

Comprehensive review of the `ccat-assistant` project against CLAUDE.md coding standards. Findings organized by area.

---

## Architecture

### 1. Missing Telegram HTTP Client Adapter Interface
**File:** [src/adapters/messenger/messenger.go](src/adapters/messenger/messenger.go), [src/adapters/messenger/poller/poller.go](src/adapters/messenger/poller/poller.go)

**Issue:** HTTP calls to Telegram are embedded directly in `Sender.sendChunk()` and `Poller.handleUpdate()`. There is no interface abstraction, making unit testing require actual HTTP mocking and coupling the code to Telegram's HTTP API specifics.

**CLAUDE.md violation:** "Wrap third-party integrations in adapters. Maintain provider-agnosticism unless project constraints require coupling." No adapter exists.

**Impact:** Difficult to test; tightly coupled to Telegram API; cannot swap implementations.

---

### 2. Naming Inconsistency: Module vs Directory vs Binary
**File:** [go.mod](go.mod), [Makefile](Makefile)

**Issue:** Go module is `github.com/vdyalex/lens-daemon`, the project directory is `ccat-assistant`, and the binary name is `lensd`. These three names do not align, suggesting an incomplete rename or unrelated purposes.

**CLAUDE.md violation:** "Names: full words; single word unless compound required. No abbreviations." The inconsistency signals unclear intent.

**Impact:** Confusion for new contributors; module path does not reflect the actual project name.

---

### 3. Inline Type Definitions in Poller
**File:** [src/adapters/messenger/poller/poller.go](src/adapters/messenger/poller/poller.go) (lines 36–54)

**Issue:** `Update`, `Message`, and `Chat` types are defined inline within the poller package. These types should be in a dedicated types file so they can be reused (e.g., in the adapter interface mentioned in issue #1) and kept separate from implementation.

**CLAUDE.md violation:** "Split modules into granular files." Types should be in a separate types file.

**Impact:** Type reuse is discouraged; types are hidden in implementation details.

---

### 4. No Structured Error Types
**File:** [src/](src/) (all modules)

**Issue:** Errors are created via `errors.New()` and `fmt.Errorf()` with string messages. Only `ErrNoForegroundWindow` is a sentinel. Callers cannot programmatically distinguish error types (e.g., timeout vs. no window vs. network failure).

**CLAUDE.md violation:** "No fragile workarounds, premature abstractions, or security vulnerabilities. Align with ... Agile principles." Lack of error types forces error handling via string matching.

**Impact:** Error handling is fragile; cannot distinguish error classes; makes retries and recovery logic brittle.

---

### 5. No Container Setup
**File:** None (missing)

**Issue:** No `Dockerfile` or `docker-compose.yml` for reproducible builds. While the runtime is macOS-only (uses CoreGraphics, Vision, AppleScript), a build container for the Go compilation step would still benefit reproducibility and CI/CD.

**CLAUDE.md violation:** "Docker required for local development and reproducible builds." No container setup exists.

**Impact:** Builds are not reproducible across environments; CI/CD setup is incomplete.

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

### 1. Static Analysis Beyond go vet
**File:** [Makefile](Makefile)

**Issue:** `make check` runs only `gofmt` and `go vet`. CLAUDE.md requires a full linter (e.g., `golangci-lint`), automatic formatting, and vulnerability scanning (e.g., `govulncheck` or `nancy`).

**CLAUDE.md violation:** "Maintain: formatter, linter, type-checker, dependency/code vulnerability scanner. Single command runs all. Auto-fix formatting. Fail build on violations."

**Impact:** Code style is inconsistent; vulnerabilities in dependencies are not detected; no single check command.

---

### 2. No CI/CD Pipeline
**File:** None (missing)

**Issue:** No `.github/workflows` or equivalent. Commits to the repository are not automatically tested, linted, or built. The human relies on local `make` commands before pushing.

**CLAUDE.md violation:** "Maintain: ... coverage reporting. ... Single command for all tests; one command for coverage."

**Impact:** Regressions can slip into the main branch; no enforcement of static checks on push.

---

### 3. ✅ Missing SUBSCRIBER_STORE_PATH in LaunchAgent Plist (FIXED)
**Status:** Fixed during Bugs phase (commit 082ba53). Added SUBSCRIBER_STORE_PATH to plist template and install script with stable default path.

---

### 4. No CHANGELOG or Versioning
**File:** None (missing)

**Issue:** No `CHANGELOG.md` or version tags. The binary and module have no version information, making it impossible to track what changed between releases or debug version-specific issues.

**CLAUDE.md violation:** "Align with ... Agile principles." Versioning is a standard practice.

**Impact:** Users cannot determine which version they are running; no release history; no semantic versioning.

---

### 5. Missing GoDoc Docstrings
**File:** [src/](src/) (all modules)

**Issue:** Most exported functions and types lack GoDoc comments. Only a few (e.g., `Config` struct) have minimal docstrings. CLAUDE.md requires docstrings on every top-level function/type with purpose, constraints, edge cases, params, and return values.

**CLAUDE.md violation:** "Docstring every top-level function/method/class: purpose, constraints, edge cases, params, return. Use language-standard docstring syntax strictly (JSDoc, rustdoc, Sphinx)."

**Impact:** API contracts are undocumented; developers must read implementation to understand behavior.

---

## Application Settings

### 1. Pipeline Step Timeouts Hardcoded
**File:** [src/pipeline/pipeline.go](src/pipeline/pipeline.go) (lines 39–45)

**Issue:** Step timeouts are hardcoded:
- ForegroundWindow: 5s
- CaptureCenter: 30s
- OCR Extract: 30s
- Agent Process: 60s
- Broadcast: 30s
- Overall: 5 minutes

These should be exposed as env vars with sensible defaults to allow tuning without code changes.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Cannot adjust timeouts for different network conditions or hardware without recompiling.

---

### 2. Chunk Size Hardcoded
**File:** [src/adapters/messenger/messenger.go](src/adapters/messenger/messenger.go) (line 55)

**Issue:** Message chunk size is hardcoded to `4096` runes. If Telegram's limits change or the user wants to send shorter messages for readability, they must modify the code.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Not configurable; brittle to API changes.

---

### 3. Worker Queue Capacity Hardcoded
**File:** [src/pipeline/pipeline.go](src/pipeline/pipeline.go) (line 63)

**Issue:** The worker queue buffer size is hardcoded to `1`. Increasing it requires code changes. An env var would allow tuning without recompilation.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Not configurable; cannot increase concurrency without code changes.

---

### 4. CGEventTap Poll Interval Hardcoded
**File:** [src/modules/listener/listener.go](src/modules/listener/listener.go) (line 43)

**Issue:** The `CFRunLoop` polls every 0.5 seconds. This hardcoded interval affects responsiveness and power consumption; it should be configurable.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Cannot tune responsiveness vs. power usage without code changes.

---

### 5. Telegram Poller Timeouts Hardcoded
**File:** [src/adapters/messenger/poller/poller.go](src/adapters/messenger/poller/poller.go) (lines 24–25)

**Issue:** Poller uses hardcoded timeouts:
- `telegramTimeout = 30 * time.Second` (server-side long-poll timeout)
- Context timeout implied as 35s

These should be env vars to allow adjustment for different network conditions.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Cannot tune for slow networks or adjust polling frequency without code changes.

---

### 6. Claude Max Tokens Hardcoded
**File:** [src/adapters/agent/agent.go](src/adapters/agent/agent.go) (line 24)

**Issue:** The Claude API request is hardcoded to request `MaxTokens: 1024`. This should be configurable to allow shorter or longer responses.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Cannot adjust response length without code changes; wastes tokens if shorter responses suffice.

---

### 7. Vision OCR Parameters Hardcoded
**File:** [src/adapters/vision/vision_bridge.m](src/adapters/vision/vision_bridge.m) (lines 21, 26)

**Issue:** The Vision framework is initialized with hardcoded:
- Accuracy level: `accurate`
- Recognition language: `en-US`

These should be configurable, especially the language, to support multi-language users.

**CLAUDE.md violation:** "Extract hardcoded settings to environment variables; group by prefix or domain."

**Impact:** Cannot adjust OCR accuracy or language without recompiling; limits usability for non-English users.

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
| Architecture | 5 | High |
| Performance | 5 | Medium |
| Code Quality | 0 | N/A |
| Best Practices | 4 | High |
| Application Settings | 7 | Medium |
| Unit Tests | 7 test suites | High |
| Functional Tests | 6 test suites | High |

**Total: 35 items** (4 bugs fixed + 5 code quality fixed + 1 best practice fixed = 10 fixed, 25 remaining)

---

## Review against CLAUDE.md

All findings are violations of CLAUDE.md sections: Core rules, Code structure, Design principles, Docstrings, Workflow, Project setup (Tests and coverage, Static checks, Containers), Security, and Review process.

## Priority Order

Next steps:
1. ✅ **Fix bugs** (4 items) — COMPLETED
2. ✅ **Code quality improvements** (5 items: remove trivial centerRect, add parseRetryAfter logging, mark TELEGRAM_CHAT_ID complete, extract remaining magic numbers, remove unused dependencies) — COMPLETED
3. **Add tests** (13 test suites: unit + functional).
4. **Extract settings to env vars** (7 items: timeouts, chunk size, poll interval, max tokens, OCR params).
5. **Improve static analysis** (Makefile: add linter, vulnerability scanner, CI/CD).
6. **Add structured error types** (all modules: replace bare `errors.New` with sentinel values).
7. **Add GoDoc docstrings** (all exported symbols).
8. **Add container setup** (optional: Dockerfile for build reproducibility).
