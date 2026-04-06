//go:build darwin

package browser_test

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/bridges/browser"
)

func TestCanvasBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		app      string
		x        int
		y        int
		width    int
		height   int
		wantNil  bool
		wantMinY int
		wantMaxY int
		wantMinX int
		wantMaxX int
	}{
		{
			// Max.Y = y + height = 200 + 600 = 800 (chrome offset cancels out in rectangle max)
			name: "safari",
			app:  "Safari",
			x:    100, y: 200, width: 800, height: 600,
			wantNil:  false,
			wantMinX: 100, wantMinY: 274,
			wantMaxX: 900, wantMaxY: 800,
		},
		{
			// Max.Y = 0 + 900 = 900
			name: "google chrome",
			app:  "Google Chrome",
			x:    0, y: 0, width: 1440, height: 900,
			wantNil:  false,
			wantMinX: 0, wantMinY: 88,
			wantMaxX: 1440, wantMaxY: 900,
		},
		{
			// Max.Y = 50 + 800 = 850
			name: "chrome alias",
			app:  "Chrome",
			x:    50, y: 50, width: 1200, height: 800,
			wantNil:  false,
			wantMinX: 50, wantMinY: 138,
			wantMaxX: 1250, wantMaxY: 850,
		},
		{
			// Max.Y = 100 + 768 = 868
			name: "firefox",
			app:  "Firefox",
			x:    0, y: 100, width: 1024, height: 768,
			wantNil:  false,
			wantMinX: 0, wantMinY: 190,
			wantMaxX: 1024, wantMaxY: 868,
		},
		{
			name: "non-browser app returns nil",
			app:  "Xcode",
			x:    0, y: 0, width: 1440, height: 900,
			wantNil: true,
		},
		{
			name: "window too short for safari chrome returns nil",
			app:  "Safari",
			x:    0, y: 0, width: 800, height: 74,
			wantNil: true,
		},
		{
			name: "window exactly chrome height returns nil",
			app:  "Safari",
			x:    0, y: 0, width: 800, height: 73,
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := browser.CanvasBounds(tc.app, tc.x, tc.y, tc.width, tc.height)
			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if result.Min.X != tc.wantMinX {
				t.Errorf("Min.X: got %d, want %d", result.Min.X, tc.wantMinX)
			}
			if result.Min.Y != tc.wantMinY {
				t.Errorf("Min.Y: got %d, want %d", result.Min.Y, tc.wantMinY)
			}
			if result.Max.X != tc.wantMaxX {
				t.Errorf("Max.X: got %d, want %d", result.Max.X, tc.wantMaxX)
			}
			if result.Max.Y != tc.wantMaxY {
				t.Errorf("Max.Y: got %d, want %d", result.Max.Y, tc.wantMaxY)
			}
		})
	}
}

func TestToolbarHeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		app        string
		wantHeight int
		wantOK     bool
	}{
		{name: "safari", app: "Safari", wantHeight: 74, wantOK: true},
		{name: "google chrome", app: "Google Chrome", wantHeight: 88, wantOK: true},
		{name: "chrome alias", app: "Chrome", wantHeight: 88, wantOK: true},
		{name: "firefox", app: "Firefox", wantHeight: 90, wantOK: true},
		{name: "unknown app", app: "Terminal", wantHeight: 0, wantOK: false},
		{name: "empty string", app: "", wantHeight: 0, wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			height, ok := browser.ToolbarHeight(tc.app)
			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if height != tc.wantHeight {
				t.Errorf("height: got %d, want %d", height, tc.wantHeight)
			}
		})
	}
}
