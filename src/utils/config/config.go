// Package config loads and manages application configuration from environment variables.
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

// loader holds configuration loading state and injects the logger dependency.
type loader struct {
	logger *slog.Logger
}

// getStr retrieves a string environment variable, returning fallback if not set.
func (l *loader) getStr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getInt retrieves an integer environment variable, returning fallback if not set or invalid.
// Logs a warning if the env var value cannot be parsed as an integer.
func (l *loader) getInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		l.logger.Warn("invalid environment variable value, using default", "key", key, "value", value, "fallback", fallback)
	}
	return fallback
}

// getLogLevel parses the named environment variable as a log level string.
// Accepted values (case-insensitive): "debug", "info", "warn", "error".
// Returns fallback if the variable is absent or unrecognised.
func (l *loader) getLogLevel(key string, fallback slog.Level) slog.Level {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		l.logger.Warn("invalid log level value, using default", "key", key, "value", value, "fallback", fallback.String())
		return fallback
	}
	return level
}

// getDuration parses the named environment variable as a time.Duration.
// Accepted formats: "300ms", "1.5h", "2h45m", etc. (see time.ParseDuration).
// Returns fallback if the variable is absent or invalid.
func (l *loader) getDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		l.logger.Warn("invalid duration value, using default", "key", key, "value", value, "fallback", fallback.String())
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
// Returns ErrConfigMissingAPIKey if ANTHROPIC_API_KEY is not set.
// Returns ErrConfigMissingBotToken if TELEGRAM_BOT_TOKEN is not set.
// Returns ErrConfigInvalidHotkey if HOTKEY_TRIGGER_KEYNAME or HOTKEY_BOUNDS_KEYNAME are invalid.
func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if present; no-op if absent or already-set vars are preserved

	ldr := &loader{logger: slog.New(slog.NewTextHandler(os.Stderr, nil))}

	cfg := &Config{
		LogLevel:                   ldr.getLogLevel("LOG_LEVEL", slog.LevelInfo),
		VisionLanguage:             ldr.getStr("VISION_LANG", "en-US"),
		VisionAccuracy:             ldr.getStr("VISION_ACCURACY", "accurate"),
		AnthropicAPIKey:            ldr.getStr("ANTHROPIC_API_KEY", ""),
		AnthropicModel:             ldr.getStr("ANTHROPIC_MODEL", "claude-sonnet-4-6"),
		AnthropicSystemPrompt:      ldr.getStr("ANTHROPIC_SYSTEM_PROMPT", "You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency."),
		AnthropicMaxResponseTokens: ldr.getInt("ANTHROPIC_MAX_RESPONSE_TOKENS", constants.AnthropicMaxResponseTokens),
		TelegramBotToken:           ldr.getStr("TELEGRAM_BOT_TOKEN", ""),
		StorePath:                  ldr.getStr("SUBSCRIBER_STORE_PATH", "tmp/subscribers"),
		TelegramMessageChunkSize:   ldr.getInt("TELEGRAM_MESSAGE_CHUNK_SIZE", constants.TelegramMessageChunkSize),
		TelegramMaxRetries:         ldr.getInt("TELEGRAM_MAX_RETRIES", constants.TelegramMaxRetries),
		TelegramLongPollTimeout:    ldr.getDuration("TELEGRAM_LONG_POLL_TIMEOUT", constants.TimeoutTelegramLongPoll),
		TelegramPollerTimeout:      ldr.getDuration("TELEGRAM_POLLER_TIMEOUT", constants.TimeoutTelegramPoller),
		TelegramHTTPClientTimeout:  ldr.getDuration("TELEGRAM_HTTP_CLIENT_TIMEOUT", constants.TimeoutTelegramHTTPClient),
		TimeoutPipelineOverall:     ldr.getDuration("TIMEOUT_PIPELINE_OVERALL", constants.TimeoutPipelineOverall),
		TimeoutForegroundWindow:    ldr.getDuration("TIMEOUT_FOREGROUND_WINDOW", constants.TimeoutForegroundWindow),
		TimeoutCapture:             ldr.getDuration("TIMEOUT_CAPTURE", constants.TimeoutCapture),
		TimeoutOCRExtract:          ldr.getDuration("TIMEOUT_OCR_EXTRACT", constants.TimeoutOCRExtract),
		TimeoutAIProcess:           ldr.getDuration("TIMEOUT_AI_PROCESS", constants.TimeoutAIProcess),
		TelegramBroadcastTimeout:   ldr.getDuration("TELEGRAM_BROADCAST_TIMEOUT", constants.TelegramBroadcastTimeout),
		EventTapPollInterval:       ldr.getDuration("EVENT_TAP_POLL_INTERVAL", constants.EventTapPollInterval),
		WorkerQueueCapacity:        ldr.getInt("WORKER_QUEUE_CAPACITY", constants.WorkerQueueCapacity),
	}

	if cfg.AnthropicAPIKey == "" {
		return nil, exceptions.ErrConfigMissingAPIKey
	}
	if cfg.TelegramBotToken == "" {
		return nil, exceptions.ErrConfigMissingBotToken
	}

	// Load and validate hotkey names
	triggerKeyName := ldr.getStr("HOTKEY_TRIGGER_KEYNAME", constants.HotkeyTriggerKeyName)
	boundsKeyName := ldr.getStr("HOTKEY_BOUNDS_KEYNAME", constants.HotkeyBoundsKeyName)

	triggerKeycode, ok := constants.HotkeyKeycodes[triggerKeyName]
	if !ok {
		return nil, exceptions.ErrConfigInvalidHotkey
	}
	boundsKeycode, ok := constants.HotkeyKeycodes[boundsKeyName]
	if !ok {
		return nil, exceptions.ErrConfigInvalidHotkey
	}

	cfg.HotkeyTriggerKeycode = triggerKeycode
	cfg.HotkeyBoundsKeycode = boundsKeycode

	return cfg, nil
}
