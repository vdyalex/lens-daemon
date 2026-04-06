//go:build darwin

// Package appkit grid wrappers: Go entry points for grid-positioning C functions
// defined in window.go. The C implementations live in window.go because they
// require access to static globals (gDelegate, gMargin, gCanvas*, gWindow*, etc.)
// that are file-local to that translation unit.
package appkit

/*
#cgo CFLAGS: -x objective-c -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework Cocoa -framework CoreGraphics

// Forward declarations for C functions defined in window.go.
// These are non-static so they are visible at link time across CGo translation units.
extern void setOverlayWindowBounds(double x, double y, double width, double height);
extern void fadeOutForMove(void);
extern void commitMoveToGridSpot(double col, double row);
extern void fadeInAfterMove(void);
*/
import "C"

// SetOverlayWindowBounds stores the raw window bounds (Y-down screen coordinates,
// logical pixels). Always set alongside canvas bounds so the outer reference is
// available for per-side margin calculation in gridSpotFrame.
// Safe to call from any goroutine; C scalar writes only, no Obj-C dispatch required.
func SetOverlayWindowBounds(x, y, width, height float64) {
	C.setOverlayWindowBounds(C.double(x), C.double(y), C.double(width), C.double(height))
}

// FadeOutForMove starts a fade-out animation for grid repositioning without ordering
// the window out. Idempotent if a move animation is already in progress.
// Safe to call from any goroutine; dispatched to the main thread internally.
func FadeOutForMove() {
	C.fadeOutForMove()
}

// CommitMoveToGridSpot repositions the overlay window at the given (col, row)
// position. col and row are percentages in [0.0, 1.0]. Does not capture the
// screen or trigger renderAdaptiveColor — safe from the screen-recording indicator.
// Safe to call from any goroutine; dispatched to the main thread internally.
func CommitMoveToGridSpot(col, row float64) {
	C.commitMoveToGridSpot(C.double(col), C.double(row))
}

// FadeInAfterMove clears the move-in-progress flag and fades the overlay in if it
// was intended to be visible. If the overlay should stay hidden, it is ordered out
// to keep the window hierarchy clean.
// Safe to call from any goroutine; dispatched to the main thread internally.
func FadeInAfterMove() {
	C.fadeInAfterMove()
}
