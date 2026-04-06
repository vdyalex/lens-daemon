// Package config loads and manages application configuration from environment variables.
package config

import (
	"log/slog"
	"os"
	"slices"
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

// getStr retrieves a string environment variable, then fallback.
// Precedence: env var > fallback
func (l *loader) getStr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getInt retrieves an integer environment variable, then fallback.
// Precedence: env var > fallback
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
// Precedence: env var > fallback
// Returns fallback if the variable is absent or unrecognised.
func (l *loader) getLogLevel(key string, fallback slog.Level) slog.Level {
	if value := os.Getenv(key); value != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(value)); err == nil {
			return level
		}
		l.logger.Warn("invalid log level value, using default", "key", key, "value", value, "fallback", fallback.String())
	}
	return fallback
}

// getFloat retrieves a float64 environment variable, then fallback.
// Precedence: env var > fallback
// Logs a warning if the env var value cannot be parsed as a float.
func (l *loader) getFloat(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
		l.logger.Warn("invalid float value, using default", "key", key, "value", value, "fallback", fallback)
	}
	return fallback
}

// getBool retrieves a boolean environment variable, then fallback.
// Accepted values: "true", "false", "1", "0", "t", "f" (see strconv.ParseBool).
// Precedence: env var > fallback
// Logs a warning if the env var value cannot be parsed as a boolean.
func (l *loader) getBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
		l.logger.Warn("invalid boolean value, using default", "key", key, "value", value, "fallback", fallback)
	}
	return fallback
}

// getDuration parses the named environment variable as a time.Duration.
// Accepted formats: "300ms", "1.5h", "2h45m", etc. (see time.ParseDuration).
// Precedence: env var > fallback
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

// getKey resolves a hotkey name environment variable to a MacOS virtual keycode.
// Returns 0 if the key name is not found in the HotkeyKeycodes mapping.
// key: environment variable name.
// fallback: default key name when the variable is not set.
func (l *loader) getKey(key string, fallback string) int {
	name := l.getStr(key, fallback)
	return constants.HotkeyKeycodes[name]
}

// Load reads application configuration from environment variables and .env file.
//
// Precedence Chain (highest to lowest):
//  1. Environment variables (either shell-set or passed from flags)
//  2. .env file variables (loaded via godotenv.Load)
//  3. Compiled default values
//
// Required env vars: ANTHROPIC_API_KEY.
// Optional env vars: TELEGRAM_BOT_TOKEN (when absent, Telegram broadcasting is disabled).
// Returns ErrConfigMissingAPIKey if ANTHROPIC_API_KEY is not set.
// Returns ErrConfigInvalidHotkey if HOTKEY_TRIGGER_KEYNAME, HOTKEY_BOUNDS_KEYNAME, or HOTKEY_TOGGLE_KEYNAME are invalid.
func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if present; no-op if absent or already-set vars are preserved

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ldr := &loader{logger: logger}

	cfg := &Config{
		// Logging
		LogLevel: ldr.getLogLevel("LOG_LEVEL", slog.LevelInfo),

		// Output method
		OutputMethod: ldr.getStr("OUTPUT_METHOD", constants.OutputMethod),

		// Hotkeys
		HotkeyTriggerKeycode: ldr.getKey("HOTKEY_TRIGGER_KEYNAME", constants.HotkeyTriggerKeyName),
		HotkeyBoundsKeycode:  ldr.getKey("HOTKEY_BOUNDS_KEYNAME", constants.HotkeyBoundsKeyName),
		HotkeyToggleKeycode:  ldr.getKey("HOTKEY_TOGGLE_KEYNAME", constants.HotkeyToggleKeyName),

		// Vision
		VisionLanguage: ldr.getStr("VISION_LANG", ""),
		VisionAccuracy: ldr.getStr("VISION_ACCURACY", constants.VisionAccuracy),

		// Anthropic AI
		AnthropicAPIKey:            ldr.getStr("ANTHROPIC_API_KEY", ""),
		AnthropicModel:             ldr.getStr("ANTHROPIC_MODEL", constants.AnthropicModel),
		AnthropicSystemPrompt:      ldr.getStr("ANTHROPIC_SYSTEM_PROMPT", constants.AnthropicSystemPrompt),
		AnthropicMaxResponseTokens: ldr.getInt("ANTHROPIC_MAX_RESPONSE_TOKENS", constants.AnthropicMaxResponseTokens),
		AnthropicCacheTTL:          ldr.getStr("ANTHROPIC_CACHE_TTL", constants.AnthropicCacheTTL),

		// Telegram
		TelegramBotToken:            ldr.getStr("TELEGRAM_BOT_TOKEN", ""),
		TelegramSubscriberStorePath: ldr.getStr("TELEGRAM_SUBSCRIBER_STORE_PATH", constants.TelegramSubscriberStorePath),
		TelegramMessageChunkSize:    ldr.getInt("TELEGRAM_MESSAGE_CHUNK_SIZE", constants.TelegramMessageChunkSize),
		TelegramMaxRetries:          ldr.getInt("TELEGRAM_MAX_RETRIES", constants.TelegramMaxRetries),
		TelegramLongPollTimeout:     ldr.getDuration("TELEGRAM_LONG_POLL_TIMEOUT", constants.TimeoutTelegramLongPoll),
		TelegramPollerTimeout:       ldr.getDuration("TELEGRAM_POLLER_TIMEOUT", constants.TimeoutTelegramPoller),
		TelegramHTTPClientTimeout:   ldr.getDuration("TELEGRAM_HTTP_CLIENT_TIMEOUT", constants.TimeoutTelegramHTTPClient),
		TelegramBroadcastTimeout:    ldr.getDuration("TELEGRAM_BROADCAST_TIMEOUT", constants.TelegramBroadcastTimeout),

		// Teleprompter appearance
		TeleprompterFontFamily:    ldr.getStr("TELEPROMPTER_FONT_FAMILY", ""),
		TeleprompterFontWeight:    ldr.getStr("TELEPROMPTER_FONT_WEIGHT", constants.TeleprompterFontWeight),
		TeleprompterFontSize:      ldr.getFloat("TELEPROMPTER_FONT_SIZE", constants.TeleprompterFontSize),
		TeleprompterOpacity:       ldr.getFloat("TELEPROMPTER_OPACITY", constants.TeleprompterOpacity),
		TeleprompterVisible:       ldr.getBool("TELEPROMPTER_VISIBLE", false),
		TeleprompterAlignment:     ldr.getStr("TELEPROMPTER_ALIGNMENT", constants.TeleprompterAlignment),
		TeleprompterAdaptiveColor: ldr.getBool("TELEPROMPTER_ADAPTIVE_COLOR", constants.TeleprompterAdaptiveColor),
		TeleprompterFadeDuration:  ldr.getFloat("TELEPROMPTER_FADE_DURATION", constants.TeleprompterFadeDuration),

		// Teleprompter grid positioning and window tracking
		TeleprompterGridMoveDebounceDuration: ldr.getDuration("TELEPROMPTER_GRID_MOVE_DEBOUNCE_DURATION", constants.TeleprompterGridMoveDebounceDuration),
		TeleprompterGridStep:                 ldr.getFloat("TELEPROMPTER_GRID_STEP", constants.TeleprompterGridStep),
		TeleprompterGridInitialCol:           ldr.getFloat("TELEPROMPTER_GRID_INITIAL_COL", constants.TeleprompterGridInitialCol),
		TeleprompterGridInitialRow:           ldr.getFloat("TELEPROMPTER_GRID_INITIAL_ROW", constants.TeleprompterGridInitialRow),
		TeleprompterWindowMonitorInterval:    ldr.getDuration("TELEPROMPTER_WINDOW_MONITOR_INTERVAL", constants.TeleprompterWindowMonitorInterval),
		TeleprompterWindowStabilizeDelay:     ldr.getDuration("TELEPROMPTER_WINDOW_STABILIZE_DELAY", constants.TeleprompterWindowStabilizeDelay),

		// Pipeline timeouts
		TimeoutPipelineOverall:  ldr.getDuration("TIMEOUT_PIPELINE_OVERALL", constants.TimeoutPipelineOverall),
		TimeoutForegroundWindow: ldr.getDuration("TIMEOUT_FOREGROUND_WINDOW", constants.TimeoutForegroundWindow),
		TimeoutCapture:          ldr.getDuration("TIMEOUT_CAPTURE", constants.TimeoutCapture),
		TimeoutOCRExtract:       ldr.getDuration("TIMEOUT_OCR_EXTRACT", constants.TimeoutOCRExtract),
		TimeoutAIProcess:        ldr.getDuration("TIMEOUT_AI_PROCESS", constants.TimeoutAIProcess),
		TimeoutCapturePhase:     ldr.getDuration("TIMEOUT_CAPTURE_PHASE", constants.TimeoutCapturePhase),
		TimeoutAnalysePhase:     ldr.getDuration("TIMEOUT_ANALYSE_PHASE", constants.TimeoutAnalysePhase),

		// Event listener
		EventTapPollInterval: ldr.getDuration("EVENT_TAP_POLL_INTERVAL", constants.EventTapPollInterval),

		// Pipeline phase
		AnalyseQueueCapacity: ldr.getInt("ANALYSE_QUEUE_CAPACITY", constants.AnalyseQueueCapacity),
	}

	if cfg.AnthropicAPIKey == "" {
		return nil, exceptions.ErrConfigMissingAPIKey
	}

	keycodes := []int{cfg.HotkeyTriggerKeycode, cfg.HotkeyBoundsKeycode, cfg.HotkeyToggleKeycode}
	if slices.Contains(keycodes, 0) {
		return nil, exceptions.ErrConfigInvalidHotkey
	}

	return cfg, nil
}
