package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

func TestLoad_defaults(test *testing.T) {
	// Clear all optional env vars to test defaults
	test.Setenv("LOG_LEVEL", "")
	test.Setenv("VISION_LANG", "")
	test.Setenv("VISION_ACCURACY", "")
	// Set required env vars
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if configuration == nil {
		test.Fatal("expected config, got nil")
	}

	// Check some defaults
	if configuration.LogLevel != slog.LevelInfo {
		test.Errorf("expected default LogLevel to be Info, got %v", configuration.LogLevel)
	}
	if configuration.VisionLanguage != "en-US" {
		test.Errorf("expected default VisionLanguage to be 'en-US', got %q", configuration.VisionLanguage)
	}
	if configuration.VisionAccuracy != "accurate" {
		test.Errorf("expected default VisionAccuracy to be 'accurate', got %q", configuration.VisionAccuracy)
	}
	if configuration.TelegramMessageChunkSize != constants.TelegramMessageChunkSize {
		test.Errorf("expected default TelegramMessageChunkSize to be %d, got %d", constants.TelegramMessageChunkSize, configuration.TelegramMessageChunkSize)
	}
}

func TestLoad_missingAPIKey(test *testing.T) {
	// Do NOT set ANTHROPIC_API_KEY
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	_, err := config.Load()

	if err == nil {
		test.Errorf("expected ConfigMissingAPIKeyException, got no error")
	}
	if err != exceptions.ConfigMissingAPIKeyException {
		test.Errorf("expected ConfigMissingAPIKeyException, got %v", err)
	}
}

func TestLoad_missingBotToken(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	// Do NOT set TELEGRAM_BOT_TOKEN

	_, err := config.Load()

	if err == nil {
		test.Errorf("expected ConfigMissingBotTokenException, got no error")
	}
	if err != exceptions.ConfigMissingBotTokenException {
		test.Errorf("expected ConfigMissingBotTokenException, got %v", err)
	}
}

func TestLoad_invalidHotkeyTrigger(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("HOTKEY_TRIGGER_KEYNAME", "BadKey")

	_, err := config.Load()

	if err == nil {
		test.Errorf("expected ConfigInvalidHotkeyException, got no error")
	}
	if err != exceptions.ConfigInvalidHotkeyException {
		test.Errorf("expected ConfigInvalidHotkeyException, got %v", err)
	}
}

func TestLoad_invalidHotkeyBounds(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("HOTKEY_BOUNDS_KEYNAME", "InvalidKey")

	_, err := config.Load()

	if err == nil {
		test.Errorf("expected ConfigInvalidHotkeyException, got no error")
	}
	if err != exceptions.ConfigInvalidHotkeyException {
		test.Errorf("expected ConfigInvalidHotkeyException, got %v", err)
	}
}

func TestLoad_validHotkeyNames(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("HOTKEY_TRIGGER_KEYNAME", "RightShift")
	test.Setenv("HOTKEY_BOUNDS_KEYNAME", "RightOption")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	// Verify keycodes are set
	if configuration.HotkeyTriggerKeycode == 0 {
		test.Errorf("expected HotkeyTriggerKeycode to be non-zero, got %d", configuration.HotkeyTriggerKeycode)
	}
	if configuration.HotkeyBoundsKeycode == 0 {
		test.Errorf("expected HotkeyBoundsKeycode to be non-zero, got %d", configuration.HotkeyBoundsKeycode)
	}
}

func TestLoad_durationOverride(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("TIMEOUT_CAPTURE", "45s")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	expected := 45 * time.Second
	if configuration.TimeoutCapture != expected {
		test.Errorf("expected TimeoutCapture to be %v, got %v", expected, configuration.TimeoutCapture)
	}
}

func TestLoad_invalidDurationFallsBack(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("TIMEOUT_CAPTURE", "notaduration")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	// Should fall back to default
	if configuration.TimeoutCapture != constants.TimeoutCapture {
		test.Errorf("expected TimeoutCapture to fall back to %v, got %v", constants.TimeoutCapture, configuration.TimeoutCapture)
	}
}

func TestLoad_intOverride(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("TELEGRAM_MAX_RETRIES", "5")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	if configuration.TelegramMaxRetries != 5 {
		test.Errorf("expected TelegramMaxRetries to be 5, got %d", configuration.TelegramMaxRetries)
	}
}

func TestLoad_invalidIntFallsBack(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("TELEGRAM_MAX_RETRIES", "xyz")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	// Should fall back to default
	if configuration.TelegramMaxRetries != constants.TelegramMaxRetries {
		test.Errorf("expected TelegramMaxRetries to fall back to %d, got %d", constants.TelegramMaxRetries, configuration.TelegramMaxRetries)
	}
}

func TestLoad_logLevelDebug(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("LOG_LEVEL", "debug")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	if configuration.LogLevel != slog.LevelDebug {
		test.Errorf("expected LogLevel to be Debug, got %v", configuration.LogLevel)
	}
}

func TestLoad_logLevelInvalidFallsBack(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("LOG_LEVEL", "verbose")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}

	// Should fall back to default (Info)
	if configuration.LogLevel != slog.LevelInfo {
		test.Errorf("expected LogLevel to fall back to Info, got %v", configuration.LogLevel)
	}
}

func TestLoad_allLogLevels(test *testing.T) {
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

	for _, condition := range tests {
		test.Run(condition.name, func(test *testing.T) {
			test.Setenv("ANTHROPIC_API_KEY", "test-key")
			test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
			test.Setenv("LOG_LEVEL", condition.value)

			configuration, err := config.Load()
			if err != nil {
				test.Fatalf("expected no error, got %v", err)
			}
			if configuration.LogLevel != condition.level {
				test.Errorf("expected LogLevel %v, got %v", condition.level, configuration.LogLevel)
			}
		})
	}
}

func TestLoad_ocrAccuracyFast(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("VISION_ACCURACY", "fast")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if configuration.VisionAccuracy != "fast" {
		test.Errorf("expected VisionAccuracy 'fast', got %q", configuration.VisionAccuracy)
	}
}

func TestLoad_subscriberStorePath(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("SUBSCRIBER_STORE_PATH", "/tmp/test-subscribers")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if configuration.SubscriberStorePath != "/tmp/test-subscribers" {
		test.Errorf("expected SubscriberStorePath '/tmp/test-subscribers', got %q", configuration.SubscriberStorePath)
	}
}

func TestLoad_claudeModel(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("ANTHROPIC_MODEL", "claude-opus-4-6")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if configuration.AnthropicModel != "claude-opus-4-6" {
		test.Errorf("expected AnthropicModel 'claude-opus-4-6', got %q", configuration.AnthropicModel)
	}
}

func TestLoad_systemPrompt(test *testing.T) {
	test.Setenv("ANTHROPIC_API_KEY", "test-key")
	test.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	test.Setenv("ANTHROPIC_SYSTEM_PROMPT", "custom prompt")

	configuration, err := config.Load()

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if configuration.AnthropicSystemPrompt != "custom prompt" {
		test.Errorf("expected AnthropicSystemPrompt 'custom prompt', got %q", configuration.AnthropicSystemPrompt)
	}
}
