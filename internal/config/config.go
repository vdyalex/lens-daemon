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
	LogLevel slog.Level // Minimum log level (CCAT_LOG_LEVEL, default: info)

	// OCR settings
	TesseractLang string // Tesseract language pack (e.g., "eng")

	// Claude AI settings
	ClaudeAPIKey string
	ClaudeModel  string
	SystemPrompt string

	// Telegram settings
	TelegramBotToken string
	TelegramChatID   int64
}

func Load() (*Config, error) {
	cfg := &Config{
		LogLevel:         envLogLevel("CCAT_LOG_LEVEL", slog.LevelInfo),
		TesseractLang:    envStr("CCAT_TESSERACT_LANG", "eng"),
		ClaudeAPIKey:     envStr("ANTHROPIC_API_KEY", ""),
		ClaudeModel:      envStr("CCAT_CLAUDE_MODEL", "claude-sonnet-4-6"),
		SystemPrompt:     envStr("CCAT_SYSTEM_PROMPT", "You are a helpful assistant that processes screen content. Analyze the following text extracted from a user's screen and provide a concise summary or relevant instructions based on the content."),
		TelegramBotToken: envStr("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   int64(envInt("TELEGRAM_CHAT_ID", 0)),
	}

	if cfg.ClaudeAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	if cfg.TelegramChatID == 0 {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID environment variable is required")
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
