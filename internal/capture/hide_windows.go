//go:build windows

package capture

import "syscall"

// HideProcess on Windows detaches the console window so the process
// does not appear in the taskbar or Alt-Tab.
// Build with: go build -ldflags="-H windowsgui" to suppress the console entirely.
func HideProcess() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	user32 := syscall.NewLazyDLL("user32.dll")
	showWindow := user32.NewProc("ShowWindow")

	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		const swHide = 0
		showWindow.Call(hwnd, swHide)
	}
}
