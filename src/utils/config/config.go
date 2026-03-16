package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

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

// getStr retrieves a string environment variable, returning fallback if not set.
func getStr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getInt retrieves an integer environment variable, returning fallback if not set or invalid.
// Logs a warning if the env var value cannot be parsed as an integer.
func getInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		logger.Warn("Invalid environment variable value, using default", "key", key, "value", value, "fallback", fallback)
	}
	return fallback
}

// getLogLevel parses the named environment variable as a log level string.
// Accepted values (case-insensitive): "debug", "info", "warn", "error".
// Returns fallback if the variable is absent or unrecognised.
func getLogLevel(key string, fallback slog.Level) slog.Level {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		logger.Warn("Invalid log level value, using default", "key", key, "value", value, "fallback", fallback.String())
		return fallback
	}
	return level
}

// getDuration parses the named environment variable as a time.Duration.
// Accepted formats: "300ms", "1.5h", "2h45m", etc. (see time.ParseDuration).
// Returns fallback if the variable is absent or invalid.
func getDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
		logger.Warn("Invalid duration value, using default", "key", key, "value", value, "fallback", fallback.String())
	}
	return fallback
}

// Load reads application configuration from environment variables.
//
// Environment Variable Precedence and Loading:
//  1. godotenv.Load() loads variables from .env file (safe variant: does NOT override already-set vars)
//  2. os.Getenv() reads variables in this order (first match wins):
//     a. Variables already set in the shell environment (highest priority)
//     b. Variables loaded from .env by godotenv (only if not in shell env)
//     c. Default fallback values defined in config struct
//
// Typical usage patterns:
// - Development: .env file contains config, loaded automatically on startup
// - Service (macOS LaunchAgent): env vars embedded in plist take precedence, .env is supplementary
// - CI/Container: env vars injected before process start, .env file optional
//
// Required env vars: ANTHROPIC_API_KEY, TELEGRAM_BOT_TOKEN.
// Optional env vars are loaded with sensible defaults (see Config struct field comments).
// Returns ConfigMissingAPIKeyException if ANTHROPIC_API_KEY is not set.
// Returns ConfigMissingBotTokenException if TELEGRAM_BOT_TOKEN is not set.
// Returns ConfigInvalidHotkeyException if HOTKEY_TRIGGER_KEYNAME or HOTKEY_BOUNDS_KEYNAME are invalid.
func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if present; no-op if absent or already-set vars are preserved

	cfg := &Config{
		LogLevel:                  getLogLevel("LOG_LEVEL", slog.LevelInfo),
		VisionLanguage:            getStr("VISION_LANG", "en-US"),
		VisionAccuracy:            getStr("VISION_ACCURACY", "accurate"),
		AnthropicAPIKey:           getStr("ANTHROPIC_API_KEY", ""),
		ClaudeModel:               getStr("CLAUDE_MODEL", "claude-sonnet-4-6"),
		SystemPrompt:              getStr("SYSTEM_PROMPT", "You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency."),
		ClaudeMaxResponseTokens:   getInt("CLAUDE_MAX_RESPONSE_TOKENS", constants.ClaudeMaxResponseTokens),
		TelegramBotToken:          getStr("TELEGRAM_BOT_TOKEN", ""),
		SubscriberStorePath:       getStr("SUBSCRIBER_STORE_PATH", "tmp/subscribers"),
		TelegramMessageChunkSize:  getInt("TELEGRAM_MESSAGE_CHUNK_SIZE", constants.TelegramMessageChunkSize),
		TelegramMaxRetries:        getInt("TELEGRAM_MAX_RETRIES", constants.TelegramMaxRetries),
		TelegramLongPollTimeout:   getDuration("TELEGRAM_LONG_POLL_TIMEOUT", constants.TimeoutTelegramLongPoll),
		TelegramPollerTimeout:     getDuration("TELEGRAM_POLLER_TIMEOUT", constants.TimeoutTelegramPoller),
		TelegramHTTPClientTimeout: getDuration("TELEGRAM_HTTP_CLIENT_TIMEOUT", constants.TimeoutTelegramHTTPClient),
		TimeoutPipelineOverall:    getDuration("TIMEOUT_PIPELINE_OVERALL", constants.TimeoutPipelineOverall),
		TimeoutForegroundWindow:   getDuration("TIMEOUT_FOREGROUND_WINDOW", constants.TimeoutForegroundWindow),
		TimeoutCapture:            getDuration("TIMEOUT_CAPTURE", constants.TimeoutCapture),
		TimeoutOCRExtract:         getDuration("TIMEOUT_OCR_EXTRACT", constants.TimeoutOCRExtract),
		TimeoutAgentProcess:       getDuration("TIMEOUT_AGENT_PROCESS", constants.TimeoutAgentProcess),
		TimeoutTelegramBroadcast:  getDuration("TIMEOUT_TELEGRAM_BROADCAST", constants.TimeoutTelegramBroadcast),
		EventTapPollInterval:      getDuration("EVENT_TAP_POLL_INTERVAL", constants.EventTapPollInterval),
		WorkerQueueCapacity:       getInt("WORKER_QUEUE_CAPACITY", constants.WorkerQueueCapacity),
	}

	if cfg.AnthropicAPIKey == "" {
		return nil, exceptions.ConfigMissingAPIKeyException
	}
	if cfg.TelegramBotToken == "" {
		return nil, exceptions.ConfigMissingBotTokenException
	}

	// Load and validate hotkey names
	triggerKeyName := getStr("HOTKEY_TRIGGER_KEYNAME", constants.HotkeyTriggerKeyName)
	boundsKeyName := getStr("HOTKEY_BOUNDS_KEYNAME", constants.HotkeyBoundsKeyName)

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
