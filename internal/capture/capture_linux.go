//go:build linux

package capture

import (
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kbinani/screenshot"
)

type linuxCapturer struct{}

// New returns a Linux-specific capturer using xdotool and X11.
func New() Capturer {
	return &linuxCapturer{}
}

func (c *linuxCapturer) ForegroundWindow() (*WindowInfo, error) {
	// Get the active window ID
	idOut, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		return nil, fmt.Errorf("xdotool getactivewindow: %w", err)
	}
	windowID := strings.TrimSpace(string(idOut))

	// Get window name
	nameOut, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output()
	if err != nil {
		return nil, fmt.Errorf("xdotool getwindowname: %w", err)
	}
	title := strings.TrimSpace(string(nameOut))

	// Get window geometry
	geomOut, err := exec.Command("xdotool", "getwindowgeometry", "--shell", windowID).Output()
	if err != nil {
		return nil, fmt.Errorf("xdotool getwindowgeometry: %w", err)
	}

	info := &WindowInfo{Title: title}
	for _, line := range strings.Split(string(geomOut), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		val, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
		switch strings.TrimSpace(parts[0]) {
		case "X":
			info.X = val
		case "Y":
			info.Y = val
		case "WIDTH":
			info.Width = val
		case "HEIGHT":
			info.Height = val
		}
	}

	return info, nil
}

func (c *linuxCapturer) ScreenSize() (int, int, error) {
	bounds := screenshot.GetDisplayBounds(0)
	return bounds.Dx(), bounds.Dy(), nil
}

func (c *linuxCapturer) CaptureCenter(win *WindowInfo) (*image.RGBA, error) {
	screenW, screenH, err := c.ScreenSize()
	if err != nil {
		return nil, err
	}

	// If window covers full screen, use full screen dimensions
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
