//go:build darwin

package capturer

import (
	"errors"
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kbinani/screenshot"
)

// ErrNoForegroundWindow is returned when the frontmost process has no
// visible window (e.g. Desktop, menu-bar-only app, minimized window),
// or when System Events lacks permission to access the window.
var ErrNoForegroundWindow = errors.New("no foreground window")

type Capture struct{}

func New() Capturer {
	return &Capture{}
}

func (capture *Capture) ForegroundWindow() (*WindowInfo, error) {
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

	out, err := exec.Command("osascript", "-e", script).Output()
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

func (capture *Capture) CaptureCenter(window *WindowInfo) (*image.RGBA, error) {
	screenW, screenH, err := capture.ScreenSize()
	if err != nil {
		return nil, fmt.Errorf("Get screen size: %w", err)
	}

	captureX, captureY, captureW, captureH := window.X, window.Y, window.Width, window.Height
	if window.Width >= screenW && window.Height >= screenH {
		captureX, captureY = 0, 0
		captureW, captureH = screenW, screenH
	}

	rect := centerRect(captureX, captureY, captureW, captureH)

	// Validate the rectangle before capture
	if rect.Min.X >= rect.Max.X || rect.Min.Y >= rect.Max.Y {
		return nil, fmt.Errorf("Invalid capture rectangle: min=(%d,%d) max=(%d,%d) from window=(x:%d, y:%d, w:%d, h:%d)",
			rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, captureX, captureY, captureW, captureH)
	}

	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("Screenshot capture: rect=(%d,%d)-(%d,%d): %w", rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y, err)
	}

	return img, nil
}
