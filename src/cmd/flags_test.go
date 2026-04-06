package cmd_test

import (
	"github.com/vdyalex/lens-daemon/src/cmd"
	"testing"
)

// TestFlagEnvPairs_emptyFlagsReturnsEmpty tests that no flags set returns empty pairs.
func TestFlagEnvPairs_emptyFlagsReturnsEmpty(t *testing.T) {
	cmd.WithFlags(cmd.GlobalFlags{}, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(pairs))
		}
	})
}

// TestFlagEnvPairs_modelOnly tests that only model flag produces one pair.
func TestFlagEnvPairs_modelOnly(t *testing.T) {
	cmd.WithFlags(cmd.GlobalFlags{Model: "claude-3"}, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 1 {
			t.Errorf("expected 1 pair, got %d", len(pairs))
		}
		if pairs[0].Key != "ANTHROPIC_MODEL" || pairs[0].Value != "claude-3" {
			t.Errorf("expected {ANTHROPIC_MODEL, claude-3}, got {%s, %s}", pairs[0].Key, pairs[0].Value)
		}
	})
}

// TestFlagEnvPairs_maxTokensZeroExcluded tests that maxTokens=0 produces no pair.
func TestFlagEnvPairs_maxTokensZeroExcluded(t *testing.T) {
	cmd.WithFlags(cmd.GlobalFlags{MaxTokens: 0}, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(pairs))
		}
	})
}

// TestFlagEnvPairs_maxTokensConverted tests that maxTokens is converted to string.
func TestFlagEnvPairs_maxTokensConverted(t *testing.T) {
	cmd.WithFlags(cmd.GlobalFlags{MaxTokens: 2048}, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 1 {
			t.Errorf("expected 1 pair, got %d", len(pairs))
		}
		if pairs[0].Key != "ANTHROPIC_MAX_RESPONSE_TOKENS" || pairs[0].Value != "2048" {
			t.Errorf("expected {ANTHROPIC_MAX_RESPONSE_TOKENS, 2048}, got {%s, %s}", pairs[0].Key, pairs[0].Value)
		}
	})
}

// TestFlagEnvPairs_allFlagsSet tests that all eight flags produce eight pairs in order.
func TestFlagEnvPairs_allFlagsSet(t *testing.T) {
	flags := cmd.GlobalFlags{
		Model:        "claude-3",
		SystemPrompt: "You are helpful",
		MaxTokens:    1024,
		LogLevel:     "debug",
		APIKey:       "sk-test",
		BotToken:     "123:ABC",
		StorePath:    "/tmp/subscribers",
		OutputMethod: "telegram",
	}
	cmd.WithFlags(flags, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 8 {
			t.Errorf("expected 8 pairs, got %d", len(pairs))
		}

		expectedKeys := []string{"ANTHROPIC_MODEL", "ANTHROPIC_SYSTEM_PROMPT", "ANTHROPIC_MAX_RESPONSE_TOKENS", "LOG_LEVEL", "ANTHROPIC_API_KEY", "TELEGRAM_BOT_TOKEN", "TELEGRAM_SUBSCRIBER_STORE_PATH", "OUTPUT_METHOD"}
		for i, expected := range expectedKeys {
			if pairs[i].Key != expected {
				t.Errorf("pair %d: expected key %s, got %s", i, expected, pairs[i].Key)
			}
		}
	})
}

// TestFlagEnvPairs_partialFlags tests that only set flags are included.
func TestFlagEnvPairs_partialFlags(t *testing.T) {
	flags := cmd.GlobalFlags{LogLevel: "info", BotToken: "token"}
	cmd.WithFlags(flags, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 2 {
			t.Errorf("expected 2 pairs, got %d", len(pairs))
		}
		if pairs[0].Key != "LOG_LEVEL" || pairs[1].Key != "TELEGRAM_BOT_TOKEN" {
			t.Errorf("unexpected keys: %s, %s", pairs[0].Key, pairs[1].Key)
		}
	})
}

// TestFlagEnvPairs_outputMethod tests that output-method flag produces a pair.
func TestFlagEnvPairs_outputMethod(t *testing.T) {
	flags := cmd.GlobalFlags{OutputMethod: "telegram"}
	cmd.WithFlags(flags, func() {
		pairs := cmd.FlagEnvPairs()
		if len(pairs) != 1 {
			t.Errorf("expected 1 pair, got %d", len(pairs))
		}
		if pairs[0].Key != "OUTPUT_METHOD" || pairs[0].Value != "telegram" {
			t.Errorf("expected {OUTPUT_METHOD, telegram}, got {%s, %s}", pairs[0].Key, pairs[0].Value)
		}
	})
}

// BenchmarkFlagEnvPairs benchmarks the flagEnvPairs function.
func BenchmarkFlagEnvPairs(b *testing.B) {
	flags := cmd.GlobalFlags{
		Model:        "claude-3",
		SystemPrompt: "prompt",
		MaxTokens:    1024,
		LogLevel:     "info",
		APIKey:       "key",
		BotToken:     "token",
	}
	cmd.WithFlags(flags, func() {
		for i := 0; i < b.N; i++ {
			cmd.FlagEnvPairs()
		}
	})
}
