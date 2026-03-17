package config

import (
	"log/slog"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Logging settings
	LogLevel slog.Level // Minimum log level (LOG_LEVEL, default: info)

	// Vision settings
	VisionLanguage string // VISION_LANG: Vision framework language (BCP 47, e.g., "en-US", default: "en-US")
	VisionAccuracy string // VISION_ACCURACY: Vision accuracy level ("accurate" or "fast", default: "accurate")

	// Anthropic AI settings
	AnthropicAPIKey            string
	AnthropicModel             string
	AnthropicSystemPrompt      string
	AnthropicMaxResponseTokens int // ANTHROPIC_MAX_RESPONSE_TOKENS: max tokens per Anthropic API call (default: 1024)

	// Telegram settings
	TelegramBotToken          string
	StorePath                 string        // File path for subscriber list (default: "tmp/subscribers")
	TelegramMessageChunkSize  int           // TELEGRAM_MESSAGE_CHUNK_SIZE: max runes per message (default: 4096)
	TelegramMaxRetries        int           // TELEGRAM_MAX_RETRIES: retry attempts on rate limit (default: 1)
	TelegramLongPollTimeout   time.Duration // TELEGRAM_LONG_POLL_TIMEOUT: server-side long-poll timeout (default: 30s)
	TelegramPollerTimeout     time.Duration // TELEGRAM_POLLER_TIMEOUT: context timeout for poller (default: 35s)
	TelegramHTTPClientTimeout time.Duration // TELEGRAM_HTTP_CLIENT_TIMEOUT: HTTP client timeout (default: 30s)

	// Pipeline timeouts
	TimeoutPipelineOverall   time.Duration // TIMEOUT_PIPELINE_OVERALL: total capture-to-broadcast time (default: 5m)
	TimeoutForegroundWindow  time.Duration // TIMEOUT_FOREGROUND_WINDOW: window detection timeout (default: 5s)
	TimeoutCapture           time.Duration // TIMEOUT_CAPTURE: screenshot capture timeout (default: 30s)
	TimeoutOCRExtract        time.Duration // TIMEOUT_OCR_EXTRACT: OCR extraction timeout (default: 30s)
	TimeoutAIProcess         time.Duration // TIMEOUT_AI_PROCESS: Claude API call timeout (default: 60s)
	TelegramBroadcastTimeout time.Duration // TELEGRAM_BROADCAST_TIMEOUT: broadcast to subscribers timeout (default: 30s)

	// Event listener settings
	EventTapPollInterval time.Duration // EVENT_TAP_POLL_INTERVAL: CFRunLoop polling interval (default: 500ms)
	HotkeyTriggerKeycode int           // Resolved from HOTKEY_TRIGGER_KEYNAME env var (default: RightShift)
	HotkeyBoundsKeycode  int           // Resolved from HOTKEY_BOUNDS_KEYNAME env var (default: RightOption)

	// Worker settings
	WorkerQueueCapacity int // WORKER_QUEUE_CAPACITY: capture queue buffer size (default: 1)
}
