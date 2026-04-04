// Package teleprompter provides a stealth macOS overlay window for displaying short answers.
// The overlay is excluded from screen sharing, Mission Control, Dock, and Cmd+Tab.
// All native AppKit operations are delegated to the appkit bridge.
package teleprompter

import (
	"unsafe"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
)

// New creates a teleprompter overlay window with the given appearance config.
// The window is positioned according to config.Position and starts with visibility
// set to config.Visible. Call Toggle() to change visibility at runtime.
//
// Must be called after appkit.StartRunLoop() has been invoked on the main thread.
func New(config appkit.OverlayConfig, visible bool) *Teleprompter {
	appkit.ConfigureOverlay(config)
	appkit.RunOnMainThread(func() unsafe.Pointer {
		return appkit.CreateOverlayWindow()
	})

	teleprompter := &Teleprompter{visible: visible}
	if visible {
		appkit.ShowOverlay()
	}

	return teleprompter
}

// Display updates the text shown on the teleprompter overlay.
// Safe to call from any goroutine; the update is dispatched to the AppKit main thread.
//
// text: the short answer to display. Empty string clears the display.
func (t *Teleprompter) Display(text string) {
	appkit.UpdateOverlayText(text)
}

// Toggle switches the overlay visibility between hidden and visible.
// Safe to call from any goroutine; the visibility change is dispatched to the AppKit main thread.
// Returns true if the overlay is now visible, false if hidden.
func (t *Teleprompter) Toggle() bool {
	t.visible = !t.visible
	if t.visible {
		appkit.ShowOverlay()
	} else {
		appkit.HideOverlay()
	}
	return t.visible
}
