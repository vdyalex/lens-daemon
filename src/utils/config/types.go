package config

import (
	"log/slog"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Logging settings
	LogLevel slog.Level // Minimum log level (LOG_LEVEL, default: info)

	// Output method: "telegram" or "teleprompter" (OUTPUT_METHOD, default: teleprompter)
	OutputMethod string

	// Vision settings
	VisionLanguage string // VISION_LANG: Vision framework language (BCP 47, e.g., "en-US", default: "en-US")
	VisionAccuracy string // VISION_ACCURACY: Vision accuracy level ("accurate" or "fast", default: "accurate")

	// Anthropic AI settings
	AnthropicAPIKey            string
	AnthropicModel             string
	AnthropicSystemPrompt      string
	AnthropicMaxResponseTokens int    // ANTHROPIC_MAX_RESPONSE_TOKENS: max tokens per Anthropic API call (default: 1024)
	AnthropicCacheTTL          string // ANTHROPIC_CACHE_TTL: prompt caching TTL for system prompt and tools ("5m" or "1h", default: "1h")

	// Telegram settings
	TelegramBotToken            string        // TELEGRAM_BOT_TOKEN: bot token from @BotFather
	TelegramSubscriberStorePath string        // TELEGRAM_SUBSCRIBER_STORE_PATH: file path for subscriber list (default: "tmp/subscribers")
	TelegramMessageChunkSize    int           // TELEGRAM_MESSAGE_CHUNK_SIZE: max runes per message (default: 4096)
	TelegramMaxRetries          int           // TELEGRAM_MAX_RETRIES: retry attempts on rate limit (default: 1)
	TelegramLongPollTimeout     time.Duration // TELEGRAM_LONG_POLL_TIMEOUT: server-side long-poll timeout (default: 30s)
	TelegramPollerTimeout       time.Duration // TELEGRAM_POLLER_TIMEOUT: context timeout for poller (default: 35s)
	TelegramHTTPClientTimeout   time.Duration // TELEGRAM_HTTP_CLIENT_TIMEOUT: HTTP client timeout (default: 0, disabled)
	TelegramBroadcastTimeout    time.Duration // TELEGRAM_BROADCAST_TIMEOUT: broadcast to subscribers timeout (default: 30s)

	// Teleprompter appearance
	TeleprompterFontFamily    string  // TELEPROMPTER_FONT_FAMILY: font family name (default: "" = system font)
	TeleprompterFontWeight    string  // TELEPROMPTER_FONT_WEIGHT: font weight name (default: "ultralight")
	TeleprompterFontSize      float64 // TELEPROMPTER_FONT_SIZE: font size in points (default: 14.0)
	TeleprompterOpacity       float64 // TELEPROMPTER_OPACITY: text opacity 0.0-1.0 (default: 0.05)
	TeleprompterVisible       bool    // TELEPROMPTER_VISIBLE: initial visibility (default: false)
	TeleprompterAlignment     string  // TELEPROMPTER_ALIGNMENT: "left", "center", "right", or "dynamic" (default: "dynamic")
	TeleprompterAdaptiveColor bool    // TELEPROMPTER_ADAPTIVE_COLOR: enable background-adaptive text color (default: true)
	TeleprompterFadeDuration  float64 // TELEPROMPTER_FADE_DURATION: fade animation duration in seconds (default: 0.75)

	// Teleprompter grid positioning and window tracking
	TeleprompterGridMoveDebounceDuration time.Duration // TELEPROMPTER_GRID_MOVE_DEBOUNCE_DURATION: idle delay before snap commit (default: 300ms)
	TeleprompterGridStep                 float64       // TELEPROMPTER_GRID_STEP: percentage per arrow press, 0.0–1.0 (default: 0.01)
	TeleprompterGridInitialCol           float64       // TELEPROMPTER_GRID_INITIAL_COL: initial horizontal position, 0.0–1.0 (default: 0.5)
	TeleprompterGridInitialRow           float64       // TELEPROMPTER_GRID_INITIAL_ROW: initial vertical position, 0.0–1.0 (default: 0.5)
	TeleprompterWindowMonitorInterval    time.Duration // TELEPROMPTER_WINDOW_MONITOR_INTERVAL: window-bounds poll interval (default: 200ms)
	TeleprompterWindowStabilizeDelay     time.Duration // TELEPROMPTER_WINDOW_STABILIZE_DELAY: stable-window delay before restoring overlay (default: 500ms)

	// Pipeline timeouts
	TimeoutPipelineOverall  time.Duration // TIMEOUT_PIPELINE_OVERALL: total capture-to-broadcast time (default: 5m)
	TimeoutForegroundWindow time.Duration // TIMEOUT_FOREGROUND_WINDOW: window detection timeout (default: 5s)
	TimeoutCapture          time.Duration // TIMEOUT_CAPTURE: screenshot capture timeout (default: 30s)
	TimeoutOCRExtract       time.Duration // TIMEOUT_OCR_EXTRACT: OCR extraction timeout (default: 30s)
	TimeoutAIProcess        time.Duration // TIMEOUT_AI_PROCESS: Claude API call timeout (default: 60s)
	TimeoutCapturePhase     time.Duration // TIMEOUT_CAPTURE_PHASE: Phase 1 total budget (default: 40s)
	TimeoutAnalysePhase     time.Duration // TIMEOUT_ANALYSE_PHASE: Phase 2 total budget (default: 5m)

	// Event listener settings
	EventTapPollInterval time.Duration // EVENT_TAP_POLL_INTERVAL: CFRunLoop polling interval (default: 500ms)
	HotkeyTriggerKeycode int           // Resolved from HOTKEY_TRIGGER_KEYNAME env var (default: RightShift)
	HotkeyBoundsKeycode  int           // Resolved from HOTKEY_BOUNDS_KEYNAME env var (default: RightOption)
	HotkeyToggleKeycode  int           // Resolved from HOTKEY_TOGGLE_KEYNAME env var (default: RightCommand)

	// Pipeline phase settings
	AnalyseQueueCapacity int // ANALYSE_QUEUE_CAPACITY: analyse channel buffer (default: 5)
}
