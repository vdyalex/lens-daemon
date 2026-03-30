package logger_test

import (
	"strings"
	"testing"

	"github.com/vdyalex/lens-daemon/src/helpers/logger"
)

// TestAttrsToArgs_emptyMapReturnsEmptySlice tests empty input.
func TestAttrsToArgs_emptyMapReturnsEmptySlice(t *testing.T) {
	result := logger.AttrsToArgs(map[string]any{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}

// TestAttrsToArgs_singleStringValue tests string key-value pair.
func TestAttrsToArgs_singleStringValue(t *testing.T) {
	attrs := map[string]any{"key": "value"}
	result := logger.AttrsToArgs(attrs)
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
	result := logger.AttrsToArgs(attrs)
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
	result := logger.AttrsToArgs(attrs)
	if len(result) != 6 {
		t.Errorf("expected 6 elements for 3-key map, got %d", len(result))
	}
}

// TestSanitizeAttrValue_nonStringPassedThrough tests non-string values unchanged.
func TestSanitizeAttrValue_nonStringPassedThrough(t *testing.T) {
	result := logger.SanitizeAttrValue(99)
	if result != 99 {
		t.Errorf("expected 99, got %v", result)
	}
}

// TestSanitizeAttrValue_stringWithoutNewlineUnchanged tests that strings without \n are unchanged.
func TestSanitizeAttrValue_stringWithoutNewlineUnchanged(t *testing.T) {
	result := logger.SanitizeAttrValue("hello world")
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %v", result)
	}
}

// TestSanitizeAttrValue_newlineEscaped tests single newline escaping.
func TestSanitizeAttrValue_newlineEscaped(t *testing.T) {
	result := logger.SanitizeAttrValue("line1\nline2")
	expected := `line1\nline2`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestSanitizeAttrValue_multipleNewlinesAllEscaped tests multiple newlines.
func TestSanitizeAttrValue_multipleNewlinesAllEscaped(t *testing.T) {
	result := logger.SanitizeAttrValue("a\nb\nc")
	expected := `a\nb\nc`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// BenchmarkAttrsToArgs benchmarks the AttrsToArgs function.
func BenchmarkAttrsToArgs(b *testing.B) {
	attrs := map[string]any{
		"a": "1", "b": "2", "c": "3", "d": 4, "e": "5",
	}
	for i := 0; i < b.N; i++ {
		logger.AttrsToArgs(attrs)
	}
}

// BenchmarkSanitizeAttrValue_withNewline benchmarks sanitization with newlines.
func BenchmarkSanitizeAttrValue_withNewline(b *testing.B) {
	val := strings.Repeat("line\n", 10)
	for i := 0; i < b.N; i++ {
		logger.SanitizeAttrValue(val)
	}
}
