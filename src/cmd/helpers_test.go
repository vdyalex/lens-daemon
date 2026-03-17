package cmd

import (
	"strings"
	"testing"
)

// withFlags temporarily sets flags for the duration of a test function.
func withFlags(newFlags globalFlags, fn func()) {
	orig := flags
	flags = newFlags
	defer func() { flags = orig }()
	fn()
}

// TestFlagEnvPairs_emptyFlagsReturnsEmpty tests that no flags set returns empty pairs.
func TestFlagEnvPairs_emptyFlagsReturnsEmpty(t *testing.T) {
	withFlags(globalFlags{}, func() {
		pairs := flagEnvPairs()
		if len(pairs) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(pairs))
		}
	})
}

// TestFlagEnvPairs_modelOnly tests that only model flag produces one pair.
func TestFlagEnvPairs_modelOnly(t *testing.T) {
	withFlags(globalFlags{model: "claude-3"}, func() {
		pairs := flagEnvPairs()
		if len(pairs) != 1 {
			t.Errorf("expected 1 pair, got %d", len(pairs))
		}
		if pairs[0].key != "ANTHROPIC_MODEL" || pairs[0].value != "claude-3" {
			t.Errorf("expected {ANTHROPIC_MODEL, claude-3}, got {%s, %s}", pairs[0].key, pairs[0].value)
		}
	})
}

// TestFlagEnvPairs_maxTokensZeroExcluded tests that maxTokens=0 produces no pair.
func TestFlagEnvPairs_maxTokensZeroExcluded(t *testing.T) {
	withFlags(globalFlags{maxTokens: 0}, func() {
		pairs := flagEnvPairs()
		if len(pairs) != 0 {
			t.Errorf("expected 0 pairs, got %d", len(pairs))
		}
	})
}

// TestFlagEnvPairs_maxTokensConverted tests that maxTokens is converted to string.
func TestFlagEnvPairs_maxTokensConverted(t *testing.T) {
	withFlags(globalFlags{maxTokens: 2048}, func() {
		pairs := flagEnvPairs()
		if len(pairs) != 1 {
			t.Errorf("expected 1 pair, got %d", len(pairs))
		}
		if pairs[0].key != "ANTHROPIC_MAX_RESPONSE_TOKENS" || pairs[0].value != "2048" {
			t.Errorf("expected {ANTHROPIC_MAX_RESPONSE_TOKENS, 2048}, got {%s, %s}", pairs[0].key, pairs[0].value)
		}
	})
}

// TestFlagEnvPairs_allFlagsSet tests that all six flags produce six pairs in order.
func TestFlagEnvPairs_allFlagsSet(t *testing.T) {
	flags := globalFlags{
		model:        "claude-3",
		systemPrompt: "You are helpful",
		maxTokens:    1024,
		logLevel:     "debug",
		apiKey:       "sk-test",
		botToken:     "123:ABC",
	}
	withFlags(flags, func() {
		pairs := flagEnvPairs()
		if len(pairs) != 6 {
			t.Errorf("expected 6 pairs, got %d", len(pairs))
		}

		expectedKeys := []string{"ANTHROPIC_MODEL", "ANTHROPIC_SYSTEM_PROMPT", "ANTHROPIC_MAX_RESPONSE_TOKENS", "LOG_LEVEL", "ANTHROPIC_API_KEY", "TELEGRAM_BOT_TOKEN"}
		for i, expected := range expectedKeys {
			if pairs[i].key != expected {
				t.Errorf("pair %d: expected key %s, got %s", i, expected, pairs[i].key)
			}
		}
	})
}

// TestFlagEnvPairs_partialFlags tests that only set flags are included.
func TestFlagEnvPairs_partialFlags(t *testing.T) {
	flags := globalFlags{logLevel: "info", botToken: "token"}
	withFlags(flags, func() {
		pairs := flagEnvPairs()
		if len(pairs) != 2 {
			t.Errorf("expected 2 pairs, got %d", len(pairs))
		}
		if pairs[0].key != "LOG_LEVEL" || pairs[1].key != "TELEGRAM_BOT_TOKEN" {
			t.Errorf("unexpected keys: %s, %s", pairs[0].key, pairs[1].key)
		}
	})
}

// TestAttrsToArgs_emptyMapReturnsEmptySlice tests empty input.
func TestAttrsToArgs_emptyMapReturnsEmptySlice(t *testing.T) {
	result := attrsToArgs(map[string]any{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}

// TestAttrsToArgs_singleStringValue tests string key-value pair.
func TestAttrsToArgs_singleStringValue(t *testing.T) {
	attrs := map[string]any{"key": "value"}
	result := attrsToArgs(attrs)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
	if result[0] != "key" || result[1] != "value" {
		t.Errorf("expected [key, value], got [%v, %v]", result[0], result[1])
	}
}

// TestAttrsToArgs_nonStringValuePassedThrough tests non-string values.
func TestAttrsToArgs_nonStringValuePassedThrough(t *testing.T) {
	attrs := map[string]any{"count": 42}
	result := attrsToArgs(attrs)
	if len(result) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result))
	}
	if result[0] != "count" || result[1] != 42 {
		t.Errorf("expected [count, 42], got [%v, %v]", result[0], result[1])
	}
}

// TestAttrsToArgs_lengthIsDoubleMapSize tests that result length is 2× map size.
func TestAttrsToArgs_lengthIsDoubleMapSize(t *testing.T) {
	attrs := map[string]any{"a": "1", "b": "2", "c": "3"}
	result := attrsToArgs(attrs)
	if len(result) != 6 {
		t.Errorf("expected 6 elements for 3-key map, got %d", len(result))
	}
}

// TestSanitizeAttrValue_nonStringPassedThrough tests non-string values unchanged.
func TestSanitizeAttrValue_nonStringPassedThrough(t *testing.T) {
	result := sanitizeAttrValue(99)
	if result != 99 {
		t.Errorf("expected 99, got %v", result)
	}
}

// TestSanitizeAttrValue_stringWithoutNewlineUnchanged tests that strings without \n are unchanged.
func TestSanitizeAttrValue_stringWithoutNewlineUnchanged(t *testing.T) {
	result := sanitizeAttrValue("hello world")
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %v", result)
	}
}

// TestSanitizeAttrValue_newlineEscaped tests single newline escaping.
func TestSanitizeAttrValue_newlineEscaped(t *testing.T) {
	result := sanitizeAttrValue("line1\nline2")
	expected := `line1\nline2`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestSanitizeAttrValue_multipleNewlinesAllEscaped tests multiple newlines.
func TestSanitizeAttrValue_multipleNewlinesAllEscaped(t *testing.T) {
	result := sanitizeAttrValue("a\nb\nc")
	expected := `a\nb\nc`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestFormatUptimeSeconds_zero tests zero uptime.
func TestFormatUptimeSeconds_zero(t *testing.T) {
	result := formatUptimeSeconds(0.0)
	if result != "0s" {
		t.Errorf("expected '0s', got '%s'", result)
	}
}

// TestFormatUptimeSeconds_belowOneMinute tests seconds only.
func TestFormatUptimeSeconds_belowOneMinute(t *testing.T) {
	result := formatUptimeSeconds(45.0)
	if result != "45s" {
		t.Errorf("expected '45s', got '%s'", result)
	}
}

// TestFormatUptimeSeconds_exactlyOneMinute tests 60 seconds.
func TestFormatUptimeSeconds_exactlyOneMinute(t *testing.T) {
	result := formatUptimeSeconds(60.0)
	if result != "1m" {
		t.Errorf("expected '1m', got '%s'", result)
	}
}

// TestFormatUptimeSeconds_belowOneHour tests minute boundaries.
func TestFormatUptimeSeconds_belowOneHour(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{90.0, "1m"},
		{119.9, "1m"},
		{120.0, "2m"},
	}
	for _, tt := range tests {
		result := formatUptimeSeconds(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatUptimeSeconds(%.1f): expected '%s', got '%s'", tt.seconds, tt.expected, result)
		}
	}
}

// TestFormatUptimeSeconds_oneHourExact tests 1 hour.
func TestFormatUptimeSeconds_oneHourExact(t *testing.T) {
	result := formatUptimeSeconds(3600.0)
	if result != "1h0m" {
		t.Errorf("expected '1h0m', got '%s'", result)
	}
}

// TestFormatUptimeSeconds_hoursAndMinutes tests mixed hours and minutes.
func TestFormatUptimeSeconds_hoursAndMinutes(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{3661.0, "1h1m"},
		{7322.0, "2h2m"},
	}
	for _, tt := range tests {
		result := formatUptimeSeconds(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatUptimeSeconds(%.1f): expected '%s', got '%s'", tt.seconds, tt.expected, result)
		}
	}
}

// BenchmarkFlagEnvPairs benchmarks the flagEnvPairs function.
func BenchmarkFlagEnvPairs(b *testing.B) {
	flags := globalFlags{
		model:        "claude-3",
		systemPrompt: "prompt",
		maxTokens:    1024,
		logLevel:     "info",
		apiKey:       "key",
		botToken:     "token",
	}
	withFlags(flags, func() {
		for i := 0; i < b.N; i++ {
			flagEnvPairs()
		}
	})
}

// BenchmarkAttrsToArgs benchmarks the attrsToArgs function.
func BenchmarkAttrsToArgs(b *testing.B) {
	attrs := map[string]any{
		"a": "1", "b": "2", "c": "3", "d": 4, "e": "5",
	}
	for i := 0; i < b.N; i++ {
		attrsToArgs(attrs)
	}
}

// BenchmarkSanitizeAttrValue_withNewline benchmarks sanitization with newlines.
func BenchmarkSanitizeAttrValue_withNewline(b *testing.B) {
	val := strings.Repeat("line\n", 10)
	for i := 0; i < b.N; i++ {
		sanitizeAttrValue(val)
	}
}

// BenchmarkFormatUptimeSeconds benchmarks the formatUptimeSeconds function.
func BenchmarkFormatUptimeSeconds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		formatUptimeSeconds(7322.5)
	}
}
