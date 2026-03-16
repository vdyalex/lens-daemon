package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Logging settings
	LogLevel slog.Level // Minimum log level (LOG_LEVEL, default: info)

	// OCR settings
	VisionLanguage string // VISION_LANG: Vision language (BCP 47, e.g., "en-US", default: "en-US")

	// Claude AI settings
	AnthropicAPIKey string
	ClaudeModel     string
	SystemPrompt    string

	// Telegram settings
	TelegramBotToken    string
	TelegramChatID      int64  // Optional: seed subscriber (legacy single-chat mode, default: 0)
	SubscriberStorePath string // File path for subscriber list (default: "tmp/subscribers")
}

func Load() (*Config, error) {
	cfg := &Config{
		LogLevel:            envLogLevel("LOG_LEVEL", slog.LevelInfo),
		VisionLanguage:      envStr("VISION_LANG", "en-US"),
		AnthropicAPIKey:     envStr("ANTHROPIC_API_KEY", ""),
		ClaudeModel:         envStr("CLAUDE_MODEL", "claude-sonnet-4-6"),
		SystemPrompt:        envStr("SYSTEM_PROMPT", "You're a questionnaire assistant. Provide quick, accurate responses with maximum efficiency."),
		TelegramBotToken:    envStr("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:      int64(envInt("TELEGRAM_CHAT_ID", 0)),
		SubscriberStorePath: envStr("SUBSCRIBER_STORE_PATH", "tmp/subscribers"),
	}

	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}

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
