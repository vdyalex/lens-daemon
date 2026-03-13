package capture

import (
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kbinani/screenshot"
)

type darwinCapturer struct{}

func New() Capturer {
	return &darwinCapturer{}
}

func (c *darwinCapturer) ForegroundWindow() (*WindowInfo, error) {
	// Use AppleScript to get the frontmost application window info
	script := `
	tell application "System Events"
		set frontApp to first application process whose frontmost is true
		set appName to name of frontApp
		tell frontApp
			set win to front window
			set winPos to position of win
			set winSize to size of win
		end tell
		return appName & "," & (item 1 of winPos) & "," & (item 2 of winPos) & "," & (item 1 of winSize) & "," & (item 2 of winSize)
	end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, fmt.Errorf("osascript: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 5 {
		return nil, fmt.Errorf("unexpected osascript output: %s", out)
	}

	x, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	y, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	w, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	h, _ := strconv.Atoi(strings.TrimSpace(parts[4]))

	return &WindowInfo{
		Title:  strings.TrimSpace(parts[0]),
		X:      x,
		Y:      y,
		Width:  w,
		Height: h,
	}, nil
}

func (c *darwinCapturer) ScreenSize() (int, int, error) {
	bounds := screenshot.GetDisplayBounds(0)
	return bounds.Dx(), bounds.Dy(), nil
}

func (c *darwinCapturer) CaptureCenter(win *WindowInfo) (*image.RGBA, error) {
	screenW, screenH, err := c.ScreenSize()
	if err != nil {
		return nil, err
	}

	captureX, captureY, captureW, captureH := win.X, win.Y, win.Width, win.Height
	if win.Width >= screenW && win.Height >= screenH {
		captureX, captureY = 0, 0
		captureW, captureH = screenW, screenH
	}

	rect := centerRect(captureX, captureY, captureW, captureH)
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("screenshot capture: %w", err)
	}

	return img, nil
}
