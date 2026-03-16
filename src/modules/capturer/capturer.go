//go:build darwin

package capturer

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"

	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

/*
#cgo CFLAGS: -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework CoreGraphics -framework AppKit -framework Foundation
#include <stdlib.h>
unsigned char* captureScreenRect(int x, int y, int width, int height, int* outLength);
int getMainDisplayWidth(void);
int getMainDisplayHeight(void);
*/
import "C"

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

	// CaptureCenter captures a region of the given window.
	// If bounds is non-nil, it is used as the capture rectangle (in screen coordinates)
	// instead of the default centerRect heuristic. Coordinates are clamped to screen bounds.
	// If bounds is nil, the entire active window is captured.
	CaptureCenter(window *WindowInfo, bounds *image.Rectangle) (*image.RGBA, error)
}

// AppleScript error codes returned by System Events.
const (
	errNoForegroundWindowCode  = "(-1728)"
	errAccessibilityDeniedCode = "(-10003)"
	osascriptOutputParts       = 5
)

// Capture is a macOS-specific implementation of the Capturer interface.
// It uses AppleScript to detect the foreground window and CoreGraphics (via CGo) to capture screenshots.
type Capture struct{}

// New creates a new Capturer instance for macOS.
func New() Capturer {
	return &Capture{}
}

// ForegroundWindow returns information about the currently focused window using AppleScript.
// It queries System Events for the frontmost application window's position and size.
// Returns CapturerNoForegroundWindowException if no foreground window exists (e.g., on the desktop).
// Returns CapturerAccessibilityDeniedException if Accessibility permission is not granted.
// Returns CapturerAppleScriptFailedException for other AppleScript errors.
func (capture *Capture) ForegroundWindow(ctx context.Context) (*WindowInfo, error) {
	// Use AppleScript to get the frontmost application window info
	script := `
	tell application "System Events"
		set frontApp to first application process whose frontmost is true
		set appName to name of frontApp
		tell frontApp
			set frontWin to front window
			set winPos to position of frontWin
			set winSize to size of frontWin
		end tell
		return appName & "," & (item 1 of winPos) & "," & (item 2 of winPos) & "," & (item 1 of winSize) & "," & (item 2 of winSize)
	end tell`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			stderrStr := string(exitErr.Stderr)
			// Check for "no foreground window" error (-1728)
			if strings.Contains(stderrStr, errNoForegroundWindowCode) {
				return nil, exceptions.CapturerNoForegroundWindowException
			}
			// Permission denied (-10003) indicates Accessibility access not granted
			if strings.Contains(stderrStr, errAccessibilityDeniedCode) {
				return nil, exceptions.CapturerAccessibilityDeniedException
			}
			return nil, fmt.Errorf("AppleScript: %s: %w", strings.TrimSpace(stderrStr), exceptions.CapturerAppleScriptFailedException)
		}
		return nil, fmt.Errorf("AppleScript: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != osascriptOutputParts {
		return nil, fmt.Errorf("Unexpected AppleScript output %s: %w", out, exceptions.CapturerAppleScriptFailedException)
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("Parse window x from AppleScript: %w", err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("Parse window y from AppleScript: %w", err)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return nil, fmt.Errorf("Parse window width from AppleScript: %w", err)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return nil, fmt.Errorf("Parse window height from AppleScript: %w", err)
	}

	return &WindowInfo{
		Title:  strings.TrimSpace(parts[0]),
		X:      x,
		Y:      y,
		Width:  w,
		Height: h,
	}, nil
}

// ScreenSize returns the full screen dimensions of the main display using CoreGraphics.
// Returns CapturerInvalidDisplayDimensionsException if the dimensions are invalid (zero or negative).
func (capture *Capture) ScreenSize() (int, int, error) {
	width := int(C.getMainDisplayWidth())
	height := int(C.getMainDisplayHeight())
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("invalid display dimensions: %dx%d: %w", width, height, exceptions.CapturerInvalidDisplayDimensionsException)
	}
	return width, height, nil
}

// CaptureCenter captures a rectangular region of the screen and returns the pixel data as image.RGBA.
// If bounds is non-nil, it is used as the capture rectangle (in screen coordinates).
// If bounds is nil, the full window bounds are used.
// The capture rectangle is clamped to screen bounds to avoid capturing offscreen areas.
// Returns CapturerInvalidCaptureRectException if the rectangle is invalid or becomes empty after clamping.
// Returns CapturerCaptureFailedException if the CoreGraphics capture call fails.
func (capture *Capture) CaptureCenter(window *WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
	screenW, screenH, err := capture.ScreenSize()
	if err != nil {
		return nil, fmt.Errorf("Get screen size: %w", err)
	}

	var rect image.Rectangle
	if bounds != nil {
		rect = *bounds
	} else {
		captureX, captureY, captureW, captureH := window.X, window.Y, window.Width, window.Height
		if window.Width >= screenW && window.Height >= screenH {
			captureX, captureY = 0, 0
			captureW, captureH = screenW, screenH
		}
		rect = image.Rect(captureX, captureY, captureX+captureW, captureY+captureH)
	}

	// Validate the rectangle before capture
	if rect.Min.X >= rect.Max.X || rect.Min.Y >= rect.Max.Y {
		return nil, fmt.Errorf("Invalid capture rectangle: min=(%d,%d) max=(%d,%d): %w",
			rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, exceptions.CapturerInvalidCaptureRectException)
	}

	// Clamp rectangle to screen bounds to avoid offscreen captures
	if rect.Min.X < 0 {
		rect.Min.X = 0
	}
	if rect.Min.Y < 0 {
		rect.Min.Y = 0
	}
	if rect.Max.X > screenW {
		rect.Max.X = screenW
	}
	if rect.Max.Y > screenH {
		rect.Max.Y = screenH
	}

	width := rect.Max.X - rect.Min.X
	height := rect.Max.Y - rect.Min.Y
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("Invalid capture rectangle after clamping: width=%d height=%d: %w", width, height, exceptions.CapturerInvalidCaptureRectException)
	}

	// Call CoreGraphics to capture the screen region
	var outLength C.int
	pixelDataPtr := C.captureScreenRect(C.int(rect.Min.X), C.int(rect.Min.Y), C.int(width), C.int(height), &outLength)
	if pixelDataPtr == nil {
		return nil, fmt.Errorf("Screenshot capture failed: rect=(%d,%d)-(%d,%d): %w", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, exceptions.CapturerCaptureFailedException)
	}
	defer C.free(unsafe.Pointer(pixelDataPtr))

	// Convert raw RGBA bytes to image.RGBA
	expectedSize := width * height * 4
	if outLength != C.int(expectedSize) {
		return nil, fmt.Errorf("Screenshot capture returned unexpected size: got %d bytes, expected %d: %w", outLength, expectedSize, exceptions.CapturerCaptureFailedException)
	}

	// Create a Go slice from the C buffer
	pixelSlice := unsafe.Slice((*byte)(pixelDataPtr), outLength)

	// Create image.RGBA and copy pixel data
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, pixelSlice)

	return img, nil
}
