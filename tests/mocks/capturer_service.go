package mocks

import (
	"context"
	"image"

	"github.com/vdyalex/lens-daemon/src/modules/capturer"
)

// MockCapturerService satisfies capturer.CapturerService interface.
type MockCapturerService struct {
	ForegroundWindowFunc func(ctx context.Context) (*capturer.WindowInfo, error)
	CaptureCenterFunc    func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error)
}

// ForegroundWindow implements capturer.CapturerService.
func (mock *MockCapturerService) ForegroundWindow(ctx context.Context) (*capturer.WindowInfo, error) {
	return mock.ForegroundWindowFunc(ctx)
}

// CaptureCenter implements capturer.CapturerService.
func (mock *MockCapturerService) CaptureCenter(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
	return mock.CaptureCenterFunc(window, bounds)
}
