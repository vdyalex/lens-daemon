//go:build darwin

package core_graphics_test

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/bridges/core_graphics"
)

// TestCapturedWindowRectByID_zeroIDReturnsNil verifies that a zero windowID produces
// a nil rectangle without invoking the CGWindowList system call.
func TestCapturedWindowRectByID_zeroIDReturnsNil(t *testing.T) {
	t.Parallel()
	result := core_graphics.CapturedWindowRectByID(0)
	if result != nil {
		t.Errorf("CapturedWindowRectByID(0): expected nil, got %v", result)
	}
}
