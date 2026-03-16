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

	"github.com/kbinani/screenshot"
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

	// CaptureCenter captures a region of the given window.
	// If bounds is non-nil, it is used as the capture rectangle (in screen coordinates)
	// instead of the default centerRect heuristic. Coordinates are clamped to screen bounds.
	// If bounds is nil, the entire active window is captured.
	CaptureCenter(window *WindowInfo, bounds *image.Rectangle) (*image.RGBA, error)
}

// centerRect computes a sub-rectangle.
// It captures the full width and the height.
func centerRect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

// ErrNoForegroundWindow is returned when the frontmost process has no
// visible window (e.g. Desktop, menu-bar-only app, minimized window),
// or when System Events lacks permission to access the window.
var ErrNoForegroundWindow = errors.New("no foreground window")

type Capture struct{}

func New() Capturer {
	return &Capture{}
}

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
			if strings.Contains(stderrStr, "(-1728)") {
				return nil, ErrNoForegroundWindow
			}
			// Permission denied (-10003) indicates Accessibility access not granted
			if strings.Contains(stderrStr, "(-10003)") {
				return nil, fmt.Errorf("Accessibility permission denied: grant access to this app in System Settings > Privacy & Security > Accessibility")
			}
			return nil, fmt.Errorf("osascript: %s", strings.TrimSpace(stderrStr))
		}
		return nil, fmt.Errorf("osascript: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 5 {
		return nil, fmt.Errorf("Unexpected osascript output: %s", out)
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("Parse window x from osascript: %w", err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("Parse window y from osascript: %w", err)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return nil, fmt.Errorf("Parse window width from osascript: %w", err)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return nil, fmt.Errorf("Parse window height from osascript: %w", err)
	}

	return &WindowInfo{
		Title:  strings.TrimSpace(parts[0]),
		X:      x,
		Y:      y,
		Width:  w,
		Height: h,
	}, nil
}

func (capture *Capture) ScreenSize() (int, int, error) {
	bounds := screenshot.GetDisplayBounds(0)
	return bounds.Dx(), bounds.Dy(), nil
}

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
		rect = centerRect(captureX, captureY, captureW, captureH)
	}

	// Validate the rectangle before capture
	if rect.Min.X >= rect.Max.X || rect.Min.Y >= rect.Max.Y {
		return nil, fmt.Errorf("Invalid capture rectangle: min=(%d,%d) max=(%d,%d)",
			rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
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

	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("Screenshot capture: rect=(%d,%d)-(%d,%d): %w", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, err)
	}

	return img, nil
}
