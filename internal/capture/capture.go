package capture

import "image"

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
	ForegroundWindow() (*WindowInfo, error)

	// ScreenSize returns the full screen dimensions.
	ScreenSize() (width, height int, err error)

	// CaptureCenter captures the center region of the given window.
	// The returned image is cropped to the center portion of the window.
	// If the window is full-screen, it captures the center of the full screen.
	CaptureCenter(win *WindowInfo) (*image.RGBA, error)
}

// centerRect computes a centered sub-rectangle within the given bounds.
// It captures roughly 60% of the width and 60% of the height, centered.
func centerRect(x, y, w, h int) image.Rectangle {
	marginX := w * 20 / 100 // 20% margin on each side = 60% center
	marginY := h * 20 / 100
	return image.Rect(
		x+marginX,
		y+marginY,
		x+w-marginX,
		y+h-marginY,
	)
}

// HideProcess is a no-op. The process is a pure CLI daemon with no GUI
// elements, so nothing shows in the Dock or Cmd-Tab.
func HideProcess() {}
