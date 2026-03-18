package cmd_test

import (
	"github.com/vdyalex/lens-daemon/src/cmd"
	"testing"
)

// TestFormatUptimeSeconds_zero tests zero uptime.
func TestFormatUptimeSeconds_zero(t *testing.T) {
	result := cmd.FormatUptime(0.0)
	if result != "0s" {
		t.Errorf("expected '0s', got '%s'", result)
	}
}

// TestFormatUptimeSeconds_belowOneMinute tests seconds only.
func TestFormatUptimeSeconds_belowOneMinute(t *testing.T) {
	result := cmd.FormatUptime(45.0)
	if result != "45s" {
		t.Errorf("expected '45s', got '%s'", result)
	}
}

// TestFormatUptimeSeconds_exactlyOneMinute tests 60 seconds.
func TestFormatUptimeSeconds_exactlyOneMinute(t *testing.T) {
	result := cmd.FormatUptime(60.0)
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
		result := cmd.FormatUptime(tt.seconds)
		if result != tt.expected {
			t.Errorf("cmd.FormatUptime(%.1f): expected '%s', got '%s'", tt.seconds, tt.expected, result)
		}
	}
}

// TestFormatUptimeSeconds_oneHourExact tests 1 hour.
func TestFormatUptimeSeconds_oneHourExact(t *testing.T) {
	result := cmd.FormatUptime(3600.0)
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
		result := cmd.FormatUptime(tt.seconds)
		if result != tt.expected {
			t.Errorf("cmd.FormatUptime(%.1f): expected '%s', got '%s'", tt.seconds, tt.expected, result)
		}
	}
}

// BenchmarkFormatUptimeSeconds benchmarks the cmd.FormatUptime function.
func BenchmarkFormatUptimeSeconds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd.FormatUptime(7322.5)
	}
}
