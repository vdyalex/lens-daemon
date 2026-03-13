package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Capture settings
	PollInterval      time.Duration // How often to check for screen changes
	DiffThreshold     float64       // Percentage of pixels that must differ (0.0-1.0)
	MaxHistory        int           // Number of screenshots to keep in ring buffer
	ScreenshotQuality int           // JPEG quality for in-memory encoding (1-100)

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
		PollInterval:      envDuration("TEST_POLL_INTERVAL", 2*time.Second),
		DiffThreshold:     envFloat("TEST_DIFF_THRESHOLD", 0.01),
		MaxHistory:        envInt("TEST_MAX_HISTORY", 50),
		ScreenshotQuality: envInt("TEST_SCREENSHOT_QUALITY", 80),
		TesseractLang:     envStr("TEST_TESSERACT_LANG", "eng"),
		ClaudeAPIKey:      envStr("ANTHROPIC_API_KEY", ""),
		ClaudeModel:       envStr("TEST_CLAUDE_MODEL", "claude-sonnet-4-6"),
		SystemPrompt:      envStr("TEST_SYSTEM_PROMPT", "You are a helpful assistant that processes screen content. Analyze the following text extracted from a user's screen and provide a concise summary or relevant instructions based on the content."),
		TelegramBotToken:  envStr("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:    int64(envInt("TELEGRAM_CHAT_ID", 0)),
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

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
