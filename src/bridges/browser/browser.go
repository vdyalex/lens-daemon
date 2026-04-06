//go:build darwin

// Package browser derives the browser content-area rectangle from window geometry,
// excluding browser toolbar (address bar, tab bar) for overlay grid positioning.
package browser

import "image"

// Toolbar heights in logical (HiDPI-independent) pixels.
// Measured empirically on macOS with default browser configurations.
// TODO: validate across macOS versions and non-default toolbar layouts.
const (
	safariToolbarHeight  = 74
	chromeToolbarHeight  = 88
	firefoxToolbarHeight = 90
)

// toolbarHeight returns the pixel height of the browser toolbar for known browsers.
// Returns (0, false) for unrecognised app names.
func toolbarHeight(appName string) (int, bool) {
	switch appName {
	case "Safari":
		return safariToolbarHeight, true
	case "Google Chrome", "Chrome":
		return chromeToolbarHeight, true
	case "Firefox":
		return firefoxToolbarHeight, true
	default:
		return 0, false
	}
}

// CanvasBounds returns the browser content rectangle for the given window,
// excluding the browser toolbar (address bar, tab bar).
//
// appName is the frontmost application name as returned by the capturer.
// x, y, width, height are the window's screen-coordinate geometry in logical pixels.
//
// Returns nil if appName is not a recognised browser or if the computed
// canvas height would be zero or negative. Callers should fall back to
// the full window rectangle when nil is returned.
func CanvasBounds(appName string, x, y, width, height int) *image.Rectangle {
	offset, ok := toolbarHeight(appName)
	if !ok {
		return nil
	}
	canvasY := y + offset
	canvasHeight := height - offset
	if canvasHeight <= 0 {
		return nil
	}
	rect := image.Rect(x, canvasY, x+width, canvasY+canvasHeight)
	return &rect
}
