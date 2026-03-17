//go:generate mockgen -destination ../../tests/mocks/mock_capturer_service.go -package mocks . Service

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

// Service abstracts foreground-window detection and screen capture.
type Service interface {
	ForegroundWindow(ctx context.Context) (*WindowInfo, error)
	CaptureCenter(window *WindowInfo, bounds *image.Rectangle) (*image.RGBA, error)
}

// Capturer detects the foreground window and captures screenshots on MacOS.
// It uses AppleScript to detect the foreground window and delegates to src/bridges/core_graphics
// for CoreGraphics calls via CGo.
type Capturer struct{}
