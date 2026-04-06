package pipeline

import (
	"testing"
)

func TestWrapPercent(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"zero stays zero", 0.0, 0.0},
		{"mid stays mid", 0.5, 0.5},
		{"just below one", 0.95, 0.95},
		{"wraps at one", 1.0, 0.0},
		{"wraps above one", 1.05, 0.05},
		{"wraps negative", -0.05, 0.95},
		{"wraps large negative", -1.05, 0.95},
		{"wraps large positive", 2.1, 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapPercent(tt.input)
			// Float comparison with tolerance.
			if diff := result - tt.expected; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("wrapPercent(%f) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}
