package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Logging settings
	LogLevel slog.Level // Minimum log level (LOG_LEVEL, default: info)

	// OCR settings
	VisionLanguage string // VISION_LANG: Vision language (BCP 47, e.g., "en-US", default: "en-US")
	VisionAccuracy string // VISION_ACCURACY: OCR accuracy level ("accurate" or "fast", default: "accurate")

	// Claude AI settings
	AnthropicAPIKey         string
	ClaudeModel             string
	SystemPrompt            string
	ClaudeMaxResponseTokens int // CLAUDE_MAX_RESPONSE_TOKENS: max tokens per Claude API call (default: 1024)

	// Telegram settings
	TelegramBotToken          string
	TelegramChatID            int64         // Optional: seed subscriber (legacy single-chat mode, default: 0)
	SubscriberStorePath       string        // File path for subscriber list (default: "tmp/subscribers")
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
	TimeoutAgentProcess      time.Duration // TIMEOUT_AGENT_PROCESS: Claude API call timeout (default: 60s)
	TimeoutTelegramBroadcast time.Duration // TIMEOUT_TELEGRAM_BROADCAST: broadcast to subscribers timeout (default: 30s)

	// Event listener settings
	EventTapPollInterval time.Duration // EVENT_TAP_POLL_INTERVAL: CFRunLoop polling interval (default: 500ms)
	HotkeyTriggerKeycode int           // Resolved from HOTKEY_TRIGGER_KEYNAME env var (default: RightShift)
	HotkeyBoundsKeycode  int           // Resolved from HOTKEY_BOUNDS_KEYNAME env var (default: RightOption)

	// Worker settings
	WorkerQueueCapacity int // WORKER_QUEUE_CAPACITY: capture queue buffer size (default: 1)
}

// Load reads application configuration from environment variables.
// Required env vars: ANTHROPIC_API_KEY, TELEGRAM_BOT_TOKEN.
// Optional env vars are loaded with sensible defaults (see Config struct field comments).
// Returns ConfigMissingAPIKeyException if ANTHROPIC_API_KEY is not set.
// Returns ConfigMissingBotTokenException if TELEGRAM_BOT_TOKEN is not set.
func Load() (*Config, error) {
	cfg := &Config{
		LogLevel:                  envLogLevel("LOG_LEVEL", slog.LevelInfo),
		VisionLanguage:            envStr("VISION_LANG", "en-US"),
		VisionAccuracy:            envStr("VISION_ACCURACY", "accurate"),
		AnthropicAPIKey:           envStr("ANTHROPIC_API_KEY", ""),
		ClaudeModel:               envStr("CLAUDE_MODEL", "claude-sonnet-4-6"),
		SystemPrompt:              envStr("SYSTEM_PROMPT", "You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency."),
		ClaudeMaxResponseTokens:   envInt("CLAUDE_MAX_RESPONSE_TOKENS", constants.ClaudeMaxResponseTokens),
		TelegramBotToken:          envStr("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:            int64(envInt("TELEGRAM_CHAT_ID", 0)),
		SubscriberStorePath:       envStr("SUBSCRIBER_STORE_PATH", "tmp/subscribers"),
		TelegramMessageChunkSize:  envInt("TELEGRAM_MESSAGE_CHUNK_SIZE", constants.TelegramMessageChunkSize),
		TelegramMaxRetries:        envInt("TELEGRAM_MAX_RETRIES", constants.TelegramMaxRetries),
		TelegramLongPollTimeout:   envDuration("TELEGRAM_LONG_POLL_TIMEOUT", constants.TimeoutTelegramLongPoll),
		TelegramPollerTimeout:     envDuration("TELEGRAM_POLLER_TIMEOUT", constants.TimeoutTelegramPoller),
		TelegramHTTPClientTimeout: envDuration("TELEGRAM_HTTP_CLIENT_TIMEOUT", constants.TimeoutTelegramHTTPClient),
		TimeoutPipelineOverall:    envDuration("TIMEOUT_PIPELINE_OVERALL", constants.TimeoutPipelineOverall),
		TimeoutForegroundWindow:   envDuration("TIMEOUT_FOREGROUND_WINDOW", constants.TimeoutForegroundWindow),
		TimeoutCapture:            envDuration("TIMEOUT_CAPTURE", constants.TimeoutCapture),
		TimeoutOCRExtract:         envDuration("TIMEOUT_OCR_EXTRACT", constants.TimeoutOCRExtract),
		TimeoutAgentProcess:       envDuration("TIMEOUT_AGENT_PROCESS", constants.TimeoutAgentProcess),
		TimeoutTelegramBroadcast:  envDuration("TIMEOUT_TELEGRAM_BROADCAST", constants.TimeoutTelegramBroadcast),
		EventTapPollInterval:      envDuration("EVENT_TAP_POLL_INTERVAL", constants.EventTapPollInterval),
		WorkerQueueCapacity:       envInt("WORKER_QUEUE_CAPACITY", constants.WorkerQueueCapacity),
	}

	if cfg.AnthropicAPIKey == "" {
		return nil, exceptions.ConfigMissingAPIKeyException
	}
	if cfg.TelegramBotToken == "" {
		return nil, exceptions.ConfigMissingBotTokenException
	}

	// Load and validate hotkey names
	triggerKeyName := envStr("HOTKEY_TRIGGER_KEYNAME", constants.HotkeyTriggerKeyName)
	boundsKeyName := envStr("HOTKEY_BOUNDS_KEYNAME", constants.HotkeyBoundsKeyName)

	triggerKeycode, ok := constants.HotkeyKeycodes[triggerKeyName]
	if !ok {
		return nil, exceptions.ConfigInvalidHotkeyException
	}
	boundsKeycode, ok := constants.HotkeyKeycodes[boundsKeyName]
	if !ok {
		return nil, exceptions.ConfigInvalidHotkeyException
	}

	cfg.HotkeyTriggerKeycode = triggerKeycode
	cfg.HotkeyBoundsKeycode = boundsKeycode

	return cfg, nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// envLogLevel parses the named environment variable as a log level string.
// Accepted values (case-insensitive): "debug", "info", "warn", "error".
// Returns fallback if the variable is absent or unrecognised.
func envLogLevel(key string, fallback slog.Level) slog.Level {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return fallback
	}
	return level
}

// envDuration parses the named environment variable as a time.Duration.
// Accepted formats: "300ms", "1.5h", "2h45m", etc. (see time.ParseDuration).
// Returns fallback if the variable is absent or invalid.
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
