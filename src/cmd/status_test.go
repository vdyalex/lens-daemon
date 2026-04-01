package cmd_test

import (
	"github.com/vdyalex/lens-daemon/src/cmd"
	"testing"
)

// TestFormatSeconds_zero tests zero uptime.
func TestFormatSeconds_zero(t *testing.T) {
	result := cmd.FormatSeconds(0.0)
	if result != "0s" {
		t.Errorf("expected '0s', got '%s'", result)
	}
}

// TestFormatSeconds_belowOneMinute tests seconds only.
func TestFormatSeconds_belowOneMinute(t *testing.T) {
	result := cmd.FormatSeconds(45.0)
	if result != "45s" {
		t.Errorf("expected '45s', got '%s'", result)
	}
}

// TestFormatSeconds_exactlyOneMinute tests 60 seconds.
func TestFormatSeconds_exactlyOneMinute(t *testing.T) {
	result := cmd.FormatSeconds(60.0)
	if result != "1m" {
		t.Errorf("expected '1m', got '%s'", result)
	}
}

// TestFormatSeconds_belowOneHour tests minute boundaries.
func TestFormatSeconds_belowOneHour(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{90.0, "1m"},
		{119.9, "1m"},
		{120.0, "2m"},
	}
	for _, tt := range tests {
		result := cmd.FormatSeconds(tt.seconds)
		if result != tt.expected {
			t.Errorf("cmd.FormatSeconds(%.1f): expected '%s', got '%s'", tt.seconds, tt.expected, result)
		}
	}
}

// TestFormatSeconds_oneHourExact tests 1 hour.
func TestFormatSeconds_oneHourExact(t *testing.T) {
	result := cmd.FormatSeconds(3600.0)
	if result != "1h0m" {
		t.Errorf("expected '1h0m', got '%s'", result)
	}
}

// TestFormatSeconds_hoursAndMinutes tests mixed hours and minutes.
func TestFormatSeconds_hoursAndMinutes(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{3661.0, "1h1m"},
		{7322.0, "2h2m"},
	}
	for _, tt := range tests {
		result := cmd.FormatSeconds(tt.seconds)
		if result != tt.expected {
			t.Errorf("cmd.FormatSeconds(%.1f): expected '%s', got '%s'", tt.seconds, tt.expected, result)
		}
	}
}

// BenchmarkFormatSeconds benchmarks the cmd.FormatSeconds function.
func BenchmarkFormatSeconds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd.FormatSeconds(7322.5)
	}
}
