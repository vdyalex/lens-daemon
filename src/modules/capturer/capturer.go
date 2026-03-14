package capturer

import (
	"context"
	"image"
)

// WindowInfo describes the currently focused window.
type WindowInfo struct {
	Title  string
	X      int
	Y      int
	Width  int
	Height int
}

// Capturer detects the foreground window and captures screenshots.
type Capturer interface {
	// ForegroundWindow returns information about the currently focused window.
	// The context parameter is used for cancellation and timeout.
	ForegroundWindow(ctx context.Context) (*WindowInfo, error)

	// ScreenSize returns the full screen dimensions.
	ScreenSize() (width, height int, err error)

	// CaptureCenter captures the center region of the given window.
	// The returned image is cropped to the center portion of the window.
	// If the window is full-screen, it captures the center of the full screen.
	CaptureCenter(window *WindowInfo) (*image.RGBA, error)
}

// centerRect computes a sub-rectangle that skips the top 200px of the window.
// It captures the full width and the height below the 200px top margin.
func centerRect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y+200, x+w, y+h)
}
