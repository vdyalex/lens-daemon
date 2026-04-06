package pipeline

import (
	"image"
	"testing"
)

func TestBoundsUnchanged(t *testing.T) {
	tests := []struct {
		name     string
		a        image.Rectangle
		b        image.Rectangle
		expected bool
	}{
		{"identical", image.Rect(0, 0, 100, 100), image.Rect(0, 0, 100, 100), true},
		{"different minX", image.Rect(1, 0, 100, 100), image.Rect(0, 0, 100, 100), false},
		{"different minY", image.Rect(0, 1, 100, 100), image.Rect(0, 0, 100, 100), false},
		{"different maxX", image.Rect(0, 0, 101, 100), image.Rect(0, 0, 100, 100), false},
		{"different maxY", image.Rect(0, 0, 100, 101), image.Rect(0, 0, 100, 100), false},
		{"both zero", image.Rect(0, 0, 0, 0), image.Rect(0, 0, 0, 0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boundsUnchanged(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("boundsUnchanged(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
