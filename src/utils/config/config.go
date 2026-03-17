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
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
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
// - Service (MacOS LaunchAgent): env vars embedded in plist take precedence, .env is supplementary
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
		LogLevel:                   getLogLevel("LOG_LEVEL", slog.LevelInfo),
		VisionLanguage:             getStr("VISION_LANG", "en-US"),
		VisionAccuracy:             getStr("VISION_ACCURACY", "accurate"),
		AnthropicAPIKey:            getStr("ANTHROPIC_API_KEY", ""),
		AnthropicModel:             getStr("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
		AnthropicSystemPrompt:      getStr("ANTHROPIC_SYSTEM_PROMPT", "You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency."),
		AnthropicMaxResponseTokens: getInt("ANTHROPIC_MAX_RESPONSE_TOKENS", constants.AnthropicMaxResponseTokens),
		TelegramBotToken:           getStr("TELEGRAM_BOT_TOKEN", ""),
		SubscriberStorePath:        getStr("SUBSCRIBER_STORE_PATH", "tmp/subscribers"),
		TelegramMessageChunkSize:   getInt("TELEGRAM_MESSAGE_CHUNK_SIZE", constants.TelegramMessageChunkSize),
		TelegramMaxRetries:         getInt("TELEGRAM_MAX_RETRIES", constants.TelegramMaxRetries),
		TelegramLongPollTimeout:    getDuration("TELEGRAM_LONG_POLL_TIMEOUT", constants.TimeoutTelegramLongPoll),
		TelegramPollerTimeout:      getDuration("TELEGRAM_POLLER_TIMEOUT", constants.TimeoutTelegramPoller),
		TelegramHTTPClientTimeout:  getDuration("TELEGRAM_HTTP_CLIENT_TIMEOUT", constants.TimeoutTelegramHTTPClient),
		TimeoutPipelineOverall:     getDuration("TIMEOUT_PIPELINE_OVERALL", constants.TimeoutPipelineOverall),
		TimeoutForegroundWindow:    getDuration("TIMEOUT_FOREGROUND_WINDOW", constants.TimeoutForegroundWindow),
		TimeoutCapture:             getDuration("TIMEOUT_CAPTURE", constants.TimeoutCapture),
		TimeoutOCRExtract:          getDuration("TIMEOUT_OCR_EXTRACT", constants.TimeoutOCRExtract),
		TimeoutAIProcess:           getDuration("TIMEOUT_AI_PROCESS", constants.TimeoutAIProcess),
		TelegramBroadcastTimeout:   getDuration("TELEGRAM_BROADCAST_TIMEOUT", constants.TelegramBroadcastTimeout),
		EventTapPollInterval:       getDuration("EVENT_TAP_POLL_INTERVAL", constants.EventTapPollInterval),
		WorkerQueueCapacity:        getInt("WORKER_QUEUE_CAPACITY", constants.WorkerQueueCapacity),
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
