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

// Telegram API connectivity constants.
const (
	// TelegramAPIHost is the hostname for the Telegram Bot API.
	TelegramAPIHost = "api.telegram.org"

	// TelegramAPIBaseURL is the full base URL for the Telegram Bot API.
	TelegramAPIBaseURL = "https://" + TelegramAPIHost

	// TelegramPathSendMessage is the URL path template for the sendMessage endpoint.
	// Callers must fmt.Sprintf the bot token into the %s placeholder.
	TelegramPathSendMessage = "/bot%s/sendMessage"

	// TelegramPathGetUpdates is the URL path template for the getUpdates endpoint.
	// Callers must fmt.Sprintf the bot token into the %s placeholder.
	TelegramPathGetUpdates = "/bot%s/getUpdates"

	// TelegramCommandStart is the bot command to subscribe a chat to broadcasts.
	TelegramCommandStart = "/start"

	// TelegramCommandStop is the bot command to unsubscribe a chat from broadcasts.
	TelegramCommandStop = "/stop"

	// TelegramAllowedUpdates is the JSON-encoded update type filter sent to getUpdates.
	TelegramAllowedUpdates = `["message"]`
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

	// EventTapRunLoopTimeout is the timeout for each CFRunLoopRunInMode call.
	// Kept short (50ms) to ensure responsive context cancellation on shutdown.
	EventTapRunLoopTimeout = 50 * time.Millisecond

	// ListenerTriggerChannelBuffer is the capacity of the hotkey trigger event channel.
	// Buffered to absorb rapid keypress bursts without stalling the CGo callback.
	ListenerTriggerChannelBuffer = 10
)

// Worker queue constants.
const (
	// Deprecated: use AnalyseQueueCapacity instead. TODO: remove after config migration.
	// WorkerQueueCapacity is retained for backward compatibility only.
	WorkerQueueCapacity = 1
)

// Pipeline phase constants.
const (
	// TimeoutCapturePhase is the total context budget for a Phase 1 goroutine.
	// Covers TimeoutForegroundWindow (5s) + TimeoutCapture (30s) + scheduling headroom.
	TimeoutCapturePhase = 40 * time.Second

	// TimeoutAnalysePhase is the total context budget for a Phase 2 iteration.
	// Covers TimeoutOCRExtract (30s) + TimeoutAIProcess (60s) + TelegramBroadcastTimeout (30s) + headroom.
	TimeoutAnalysePhase = 5 * time.Minute

	// AnalyseQueueCapacity is the default buffer depth for the analyse channel.
	// 5 slots keeps a tight backlog; excess captures are dropped with a warning.
	AnalyseQueueCapacity = 5
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

// Daemon startup polling constants.
const (
	// TimeoutDaemonStartup is the max wait for a PID file after re-exec.
	TimeoutDaemonStartup = 3 * time.Second
	// IntervalDaemonStartupPoll is how often to check for the PID file.
	IntervalDaemonStartupPoll = 100 * time.Millisecond
	// TimeoutDaemonStop is the max wait for the daemon process to exit after SIGTERM.
	TimeoutDaemonStop = 5 * time.Second
	// IntervalDaemonStopPoll is how often to re-check the PID file during shutdown.
	IntervalDaemonStopPoll = 100 * time.Millisecond
)

// IPC layer constants.
const (
	// TimeoutIPCClient is the default dial/operation timeout for IPC Client.
	TimeoutIPCClient = 5 * time.Second
	// IPCMaxFrameSize is the 4 MiB safety limit for length-prefixed frames.
	IPCMaxFrameSize = 4 * 1024 * 1024
	// IPCLogSubscriberBuffer is the channel capacity for log event subscribers.
	IPCLogSubscriberBuffer = 64
	// IPCReadTimeout is the per-connection read deadline for server and client frames.
	IPCReadTimeout = 5 * time.Second
	// IPCLogRingBuffer is the number of recent log events replayed to new subscribers.
	IPCLogRingBuffer = 100
	// SubcommandDaemon is the argv[1] used when re-execing the daemon child.
	SubcommandDaemon = "daemon"
)

// Slog text handler field keys used by the log broker parser.
const (
	// SlogFieldTime is the key emitted by slog.TextHandler for the event timestamp.
	SlogFieldTime = "time"
	// SlogFieldLevel is the key emitted by slog.TextHandler for the log level.
	SlogFieldLevel = "level"
	// SlogFieldMessage is the key emitted by slog.TextHandler for the log message.
	SlogFieldMessage = "msg"
)

// Log streaming constants.
const (
	// LogsReconnectInterval is the delay before the logs command retries after a daemon disconnect.
	LogsReconnectInterval = 3 * time.Second

	// RecentLogLineCount is the number of trailing log lines printed on daemon startup failure.
	RecentLogLineCount = 10
)

// File permission modes.
const (
	// PermissionPIDDirectory is the mode for the PID file parent directory.
	PermissionPIDDirectory = 0700
	// PermissionPIDFile is the mode for the PID file itself.
	PermissionPIDFile = 0600
	// PermissionSocket is the mode for the Unix domain socket file.
	PermissionSocket = 0600
	// PermissionLogFile is the mode for the daemon log redirect file.
	PermissionLogFile = 0600
	// PermissionSubscribersDirectory is the mode for the daemon subscriber store parent directory.
	PermissionSubscribersDirectory = 0700
	// PermissionSubscribersFile is the mode for the daemon store the list of subscribers.
	PermissionSubscribersFile = 0600
)

// Image capture constants.
const (
	// BytesPerPixelRGBA is the number of bytes per pixel in a raw RGBA image buffer.
	BytesPerPixelRGBA = 4
)

// Application configuration defaults.
const (
	// DefaultVisionLanguage is the BCP-47 tag passed to Vision framework.
	DefaultVisionLanguage = "en-US"
	// DefaultVisionAccuracy is the Vision framework accuracy level.
	DefaultVisionAccuracy = "accurate"
	// DefaultAnthropicModel is the Anthropic model used when unconfigured.
	DefaultAnthropicModel = "claude-sonnet-4-6"
	// DefaultAnthropicSystemPrompt is the system prompt used when unconfigured.
	DefaultAnthropicSystemPrompt = "You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency."
	// DefaultStorePath is the subscriber store file path used when unconfigured.
	DefaultStorePath = "tmp/subscribers"
)
