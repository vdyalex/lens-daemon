package constants

import "time"

// Timeout constants for pipeline steps.
// These define the maximum duration allowed for each operation.
const (
	// TimeoutPipelineOverall is the total time allowed for a complete capture-to-broadcast cycle.
	TimeoutPipelineOverall = 5 * time.Minute

	// TimeoutForegroundWindow is the deadline for detecting the active window via AppleScript.
	TimeoutForegroundWindow = 5 * time.Second

	// TimeoutCapture is the deadline for taking a screenshot.
	TimeoutCapture = 30 * time.Second

	// TimeoutOCRExtract is the deadline for running Vision OCR on the captured image.
	TimeoutOCRExtract = 30 * time.Second

	// TimeoutAgentProcess is the deadline for Claude API call and response.
	TimeoutAgentProcess = 60 * time.Second

	// TimeoutTelegramBroadcast is the deadline for sending messages to all subscribers.
	TimeoutTelegramBroadcast = 30 * time.Second
)

// Telegram-related timeouts.
const (
	// TimeoutTelegramHTTPClient is the per-request HTTP timeout for Telegram API calls.
	TimeoutTelegramHTTPClient = 30 * time.Second

	// TimeoutTelegramPoller is the context timeout for Telegram polling operations.
	// Set 5s longer than the server-side timeout to allow for network jitter.
	TimeoutTelegramPoller = 35 * time.Second

	// TimeoutTelegramLongPoll is the server-side long-poll timeout sent to Telegram API.
	// Telegram will wait this long for new updates before responding.
	TimeoutTelegramLongPoll = 30 * time.Second

	// TimeoutTelegramRetryBackoff is the delay before retrying a failed Telegram request.
	TimeoutTelegramRetryBackoff = 5 * time.Second

	// TimeoutTelegramRetryFallback is the fallback retry delay if parsing the server's
	// retry-after header fails (e.g., due to API format change).
	TimeoutTelegramRetryFallback = 1 * time.Second
)

// Message formatting constants.
const (
	// TelegramMessageChunkSize is the maximum number of UTF-8 runes per Telegram message.
	// Telegram API enforces a 4096-character limit; we split on rune boundaries.
	TelegramMessageChunkSize = 4096

	// TelegramParseMode is the Telegram message parsing mode for formatted text.
	// MarkdownV2 enables bold, italic, inline code, code blocks, and other formatting.
	TelegramParseMode = "MarkdownV2"

	// TelegramMaxRetries is the maximum number of retry attempts for failed Telegram API calls.
	// Set to 1 to retry once on rate-limit or transient failure; 0 means no retries.
	TelegramMaxRetries = 1
)

// Claude API constants.
const (
	// ClaudeMaxResponseTokens is the maximum tokens requested from Claude API per call.
	// Higher values allow longer responses but consume more tokens.
	ClaudeMaxResponseTokens = 1024
)

// Event listener constants.
const (
	// EventTapPollInterval is the CFRunLoop polling interval for detecting keyboard events.
	// Smaller values increase responsiveness but consume more CPU; 500ms is a reasonable balance.
	EventTapPollInterval = 500 * time.Millisecond
)

// Worker queue constants.
const (
	// WorkerQueueCapacity is the buffer size for the capture trigger channel.
	// Only 1 concurrent capture is allowed; additional triggers are dropped if full.
	WorkerQueueCapacity = 1
)
