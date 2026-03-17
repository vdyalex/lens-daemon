package config_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

func TestLoad_defaults(t *testing.T) {
	// Clear all optional env vars to test defaults
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("VISION_LANG", "")
	t.Setenv("VISION_ACCURACY", "")
	// Set required env vars
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if configuration == nil {
		t.Fatal("expected config, got nil")
	}

	// Check some defaults
	if configuration.LogLevel != slog.LevelInfo {
		t.Errorf("expected default LogLevel to be Info, got %v", configuration.LogLevel)
	}
	if configuration.VisionLanguage != "en-US" {
		t.Errorf("expected default VisionLanguage to be 'en-US', got %q", configuration.VisionLanguage)
	}
	if configuration.VisionAccuracy != "accurate" {
		t.Errorf("expected default VisionAccuracy to be 'accurate', got %q", configuration.VisionAccuracy)
	}
	if configuration.TelegramMessageChunkSize != constants.TelegramMessageChunkSize {
		t.Errorf("expected default TelegramMessageChunkSize to be %d, got %d", constants.TelegramMessageChunkSize, configuration.TelegramMessageChunkSize)
	}
}

func TestLoad_missingAPIKey(t *testing.T) {
	// Do NOT set ANTHROPIC_API_KEY
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	_, err := config.Load()

	if err == nil {
		t.Errorf("expected ErrConfigMissingAPIKey, got no error")
	}
	if !errors.Is(err, exceptions.ErrConfigMissingAPIKey) {
		t.Errorf("expected ErrConfigMissingAPIKey, got %v", err)
	}
}

func TestLoad_missingBotToken(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	// Do NOT set TELEGRAM_BOT_TOKEN

	_, err := config.Load()

	if err == nil {
		t.Errorf("expected ErrConfigMissingBotToken, got no error")
	}
	if !errors.Is(err, exceptions.ErrConfigMissingBotToken) {
		t.Errorf("expected ErrConfigMissingBotToken, got %v", err)
	}
}

func TestLoad_invalidHotkeyTrigger(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("HOTKEY_TRIGGER_KEYNAME", "BadKey")

	_, err := config.Load()

	if err == nil {
		t.Errorf("expected ErrConfigInvalidHotkey, got no error")
	}
	if !errors.Is(err, exceptions.ErrConfigInvalidHotkey) {
		t.Errorf("expected ErrConfigInvalidHotkey, got %v", err)
	}
}

func TestLoad_invalidHotkeyBounds(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("HOTKEY_BOUNDS_KEYNAME", "InvalidKey")

	_, err := config.Load()

	if err == nil {
		t.Errorf("expected ErrConfigInvalidHotkey, got no error")
	}
	if !errors.Is(err, exceptions.ErrConfigInvalidHotkey) {
		t.Errorf("expected ErrConfigInvalidHotkey, got %v", err)
	}
}

func TestLoad_validHotkeyNames(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("HOTKEY_TRIGGER_KEYNAME", "RightShift")
	t.Setenv("HOTKEY_BOUNDS_KEYNAME", "RightOption")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify keycodes are set
	if configuration.HotkeyTriggerKeycode == 0 {
		t.Errorf("expected HotkeyTriggerKeycode to be non-zero, got %d", configuration.HotkeyTriggerKeycode)
	}
	if configuration.HotkeyBoundsKeycode == 0 {
		t.Errorf("expected HotkeyBoundsKeycode to be non-zero, got %d", configuration.HotkeyBoundsKeycode)
	}
}

func TestLoad_durationOverride(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TIMEOUT_CAPTURE", "45s")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := 45 * time.Second
	if configuration.TimeoutCapture != expected {
		t.Errorf("expected TimeoutCapture to be %v, got %v", expected, configuration.TimeoutCapture)
	}
}

func TestLoad_invalidDurationFallsBack(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TIMEOUT_CAPTURE", "notaduration")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should fall back to default
	if configuration.TimeoutCapture != constants.TimeoutCapture {
		t.Errorf("expected TimeoutCapture to fall back to %v, got %v", constants.TimeoutCapture, configuration.TimeoutCapture)
	}
}

func TestLoad_intOverride(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_MAX_RETRIES", "5")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if configuration.TelegramMaxRetries != 5 {
		t.Errorf("expected TelegramMaxRetries to be 5, got %d", configuration.TelegramMaxRetries)
	}
}

func TestLoad_invalidIntFallsBack(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_MAX_RETRIES", "xyz")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should fall back to default
	if configuration.TelegramMaxRetries != constants.TelegramMaxRetries {
		t.Errorf("expected TelegramMaxRetries to fall back to %d, got %d", constants.TelegramMaxRetries, configuration.TelegramMaxRetries)
	}
}

func TestLoad_logLevelDebug(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LOG_LEVEL", "debug")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if configuration.LogLevel != slog.LevelDebug {
		t.Errorf("expected LogLevel to be Debug, got %v", configuration.LogLevel)
	}
}

func TestLoad_logLevelInvalidFallsBack(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LOG_LEVEL", "verbose")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should fall back to default (Info)
	if configuration.LogLevel != slog.LevelInfo {
		t.Errorf("expected LogLevel to fall back to Info, got %v", configuration.LogLevel)
	}
}

func TestLoad_allLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		value string
		level slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ANTHROPIC_API_KEY", "test-key")
			t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
			t.Setenv("LOG_LEVEL", tt.value)

			configuration, err := config.Load()
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if configuration.LogLevel != tt.level {
				t.Errorf("expected LogLevel %v, got %v", tt.level, configuration.LogLevel)
			}
		})
	}
}

func TestLoad_ocrAccuracyFast(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("VISION_ACCURACY", "fast")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if configuration.VisionAccuracy != "fast" {
		t.Errorf("expected VisionAccuracy 'fast', got %q", configuration.VisionAccuracy)
	}
}

func TestLoad_subscriberStorePath(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("SUBSCRIBER_STORE_PATH", "/tmp/test-subscribers")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if configuration.StorePath != "/tmp/test-subscribers" {
		t.Errorf("expected StorePath '/tmp/test-subscribers', got %q", configuration.StorePath)
	}
}

func TestLoad_claudeModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_MODEL", "claude-opus-4-6")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if configuration.AnthropicModel != "claude-opus-4-6" {
		t.Errorf("expected AnthropicModel 'claude-opus-4-6', got %q", configuration.AnthropicModel)
	}
}

func TestLoad_systemPrompt(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_SYSTEM_PROMPT", "custom prompt")

	configuration, err := config.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if configuration.AnthropicSystemPrompt != "custom prompt" {
		t.Errorf("expected AnthropicSystemPrompt 'custom prompt', got %q", configuration.AnthropicSystemPrompt)
	}
}
