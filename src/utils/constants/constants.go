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

	// TimeoutAIProcess is the deadline for Claude API call and response.
	TimeoutAIProcess = 60 * time.Second

	// TelegramBroadcastTimeout is the deadline for sending messages to all subscribers.
	TelegramBroadcastTimeout = 30 * time.Second
)

// Telegram-related timeouts.
const (
	// TimeoutTelegramHTTPClient is disabled (0) for long-polling.
	// The context-based TimeoutTelegramPoller is the correct bounding mechanism;
	// a per-request HTTP client timeout races with the server-side long-poll timeout
	// and causes spurious "Client.Timeout exceeded" errors.
	TimeoutTelegramHTTPClient = 0

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

// Anthropic API constants.
const (
	// AnthropicMaxResponseTokens is the maximum tokens requested from Anthropic API per call.
	// Higher values allow longer responses but consume more tokens.
	AnthropicMaxResponseTokens = 1024
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

// Hotkey defaults.
const (
	// HotkeyTriggerKeyName is the default key name for the capture-trigger hotkey.
	HotkeyTriggerKeyName = "RightShift"

	// HotkeyBoundsKeyName is the default key name for the bounds-selection hotkey.
	HotkeyBoundsKeyName = "RightOption"
)

// Hotkey keycode mappings (MacOS virtual keycodes).
var HotkeyKeycodes = map[string]int{
	"LeftShift":    0x38, // 56
	"RightShift":   0x3C, // 60
	"LeftControl":  0x3B, // 59
	"RightControl": 0x3E, // 62
	"LeftCommand":  0x37, // 55
	"RightCommand": 0x36, // 54
	"LeftOption":   0x3A, // 58
	"RightOption":  0x3D, // 61
	"Fn":           0x3F, // 63
}
