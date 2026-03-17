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

	"github.com/vdyalex/lens-daemon/src/bridges/core_graphics"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// AppleScript error codes returned by System Events.
const (
	errNoForegroundWindowCode  = "(-1728)"
	errAccessibilityDeniedCode = "(-10003)"
	osascriptOutputParts       = 5
)

// New creates a new Capturer instance for MacOS.
func New() *Capturer {
	return &Capturer{}
}

// parseWindowInfo parses AppleScript output into a WindowInfo.
// Output format: "AppName,x,y,width,height"
func parseWindowInfo(rawOutput string) (*WindowInfo, error) {
	parts := strings.Split(strings.TrimSpace(rawOutput), ",")
	if len(parts) != osascriptOutputParts {
		return nil, fmt.Errorf("unexpected applescript output %s: %w", rawOutput, exceptions.ErrCapturerAppleScriptFailed)
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("parse window x from applescript: %w", err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("parse window y from applescript: %w", err)
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return nil, fmt.Errorf("parse window width from applescript: %w", err)
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return nil, fmt.Errorf("parse window height from applescript: %w", err)
	}

	return &WindowInfo{
		Title:  strings.TrimSpace(parts[0]),
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}, nil
}

// computeCaptureRect determines the screen-coordinate rectangle to capture.
// Applies custom bounds if non-nil, otherwise uses window bounds.
// Clamps to screen dimensions and validates the result.
func computeCaptureRect(window *WindowInfo, bounds *image.Rectangle, screenW, screenH int) (image.Rectangle, error) {
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
		return image.Rectangle{}, fmt.Errorf("invalid capture rectangle: min=(%d,%d) max=(%d,%d): %w",
			rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, exceptions.ErrCapturerInvalidCaptureRect)
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
		return image.Rectangle{}, fmt.Errorf("invalid capture rectangle after clamping: width=%d height=%d: %w", width, height, exceptions.ErrCapturerInvalidCaptureRect)
	}

	return rect, nil
}

// ForegroundWindow returns information about the currently focused window using AppleScript.
// It queries System Events for the frontmost application window's position and size.
// Returns ErrCapturerNoForegroundWindow if no foreground window exists (e.g., on the desktop).
// Returns ErrCapturerAccessibilityDenied if Accessibility permission is not granted.
// Returns ErrCapturerAppleScriptFailed for other AppleScript errors.
func (c *Capturer) ForegroundWindow(ctx context.Context) (*WindowInfo, error) {
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
				return nil, exceptions.ErrCapturerNoForegroundWindow
			}
			// Permission denied (-10003) indicates Accessibility access not granted
			if strings.Contains(stderrStr, errAccessibilityDeniedCode) {
				return nil, exceptions.ErrCapturerAccessibilityDenied
			}
			return nil, fmt.Errorf("applescript: %s: %w", strings.TrimSpace(stderrStr), exceptions.ErrCapturerAppleScriptFailed)
		}
		return nil, fmt.Errorf("applescript: %w", err)
	}

	return parseWindowInfo(strings.TrimSpace(string(out)))
}

// ScreenSize returns the full screen dimensions of the main display using CoreGraphics.
// Returns ErrCapturerInvalidDisplayDimensions if the dimensions are invalid (zero or negative).
func (c *Capturer) ScreenSize() (int, int, error) {
	width := core_graphics.GetMainDisplayWidth()
	height := core_graphics.GetMainDisplayHeight()
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("invalid display dimensions: %dx%d: %w", width, height, exceptions.ErrCapturerInvalidDisplayDimensions)
	}
	return width, height, nil
}

// CaptureCenter captures a rectangular region of the screen and returns the pixel data as image.RGBA.
// If bounds is non-nil, it is used as the capture rectangle (in screen coordinates).
// If bounds is nil, the full window bounds are used.
// The capture rectangle is clamped to screen bounds to avoid capturing offscreen areas.
// On HiDPI/Retina displays, the returned image may have higher physical pixel dimensions
// than the logical coordinate dimensions used for the capture rectangle.
// Returns ErrCapturerInvalidCaptureRect if the rectangle is invalid or becomes empty after clamping.
// Returns ErrCapturerCaptureFailed if the CoreGraphics capture call fails.
func (c *Capturer) CaptureCenter(window *WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
	screenW, screenH, err := c.ScreenSize()
	if err != nil {
		return nil, fmt.Errorf("get screen size: %w", err)
	}

	rect, err := computeCaptureRect(window, bounds, screenW, screenH)
	if err != nil {
		return nil, err
	}

	width := rect.Max.X - rect.Min.X
	height := rect.Max.Y - rect.Min.Y

	// Call CoreGraphics to capture the screen region
	pixelData, actualWidth, actualHeight, ok := core_graphics.CaptureScreenRect(rect.Min.X, rect.Min.Y, width, height)
	if !ok {
		return nil, fmt.Errorf("screenshot capture failed: rect=(%d,%d)-(%d,%d): %w", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, exceptions.ErrCapturerCaptureFailed)
	}

	// Validate returned buffer size using actual physical pixel dimensions
	expectedSize := actualWidth * actualHeight * 4
	if len(pixelData) != expectedSize {
		return nil, fmt.Errorf("screenshot capture returned unexpected size: got %d bytes, expected %d: %w", len(pixelData), expectedSize, exceptions.ErrCapturerCaptureFailed)
	}

	// Create image.RGBA with actual physical pixel dimensions and copy pixel data
	img := image.NewRGBA(image.Rect(0, 0, actualWidth, actualHeight))
	copy(img.Pix, pixelData)

	return img, nil
}
