//go:build windows

package capture

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"

	"github.com/kbinani/screenshot"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type windowsCapturer struct{}

func New() Capturer {
	return &windowsCapturer{}
}

func (c *windowsCapturer) ForegroundWindow() (*WindowInfo, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return nil, fmt.Errorf("no foreground window found")
	}

	// Get window title
	buf := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	title := syscall.UTF16ToString(buf)

	// Get window rect
	var r rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))

	return &WindowInfo{
		Title:  title,
		X:      int(r.Left),
		Y:      int(r.Top),
		Width:  int(r.Right - r.Left),
		Height: int(r.Bottom - r.Top),
	}, nil
}

func (c *windowsCapturer) ScreenSize() (int, int, error) {
	bounds := screenshot.GetDisplayBounds(0)
	return bounds.Dx(), bounds.Dy(), nil
}

func (c *windowsCapturer) CaptureCenter(win *WindowInfo) (*image.RGBA, error) {
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
