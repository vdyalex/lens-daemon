package listener

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// Forward declarations – implemented in Go via export.
extern void goHotkeyCallback(void);
extern void goRecordBounds(CGFloat minX, CGFloat minY, CGFloat maxX, CGFloat maxY);

// Global tap reference so the callback can re-enable it on timeout.
static CFMachPortRef gTap = NULL;

// Right Option bounds tracking state.
static bool gOptionHeld = false;
static CGFloat gMinX, gMinY, gMaxX, gMaxY;

// CGEventTap callback: fires on every flagsChanged event.
static CGEventRef eventCallback(CGEventTapProxy proxy, CGEventType type,
                                CGEventRef event, void *refcon) {
    (void)proxy;
    (void)refcon;

    // If the tap is disabled by the system, re-enable it.
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (gTap != NULL) {
            CGEventTapEnable(gTap, true);
        }
        return event;
    }

    // Handle Right Option (keycode 0x3D) bounds tracking.
    if (type == kCGEventFlagsChanged) {
        CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
        CGEventFlags flags = CGEventGetFlags(event);

        // Right Option: keycode 0x3D (61).
        if (keycode == 0x3D) {
            bool optionNowHeld = (flags & kCGEventFlagMaskAlternate) != 0;
            if (optionNowHeld && !gOptionHeld) {
                // Right Option pressed: start tracking.
                gOptionHeld = true;
                CGPoint loc = CGEventGetLocation(event);
                gMinX = gMaxX = loc.x;
                gMinY = gMaxY = loc.y;
            } else if (!optionNowHeld && gOptionHeld) {
                // Right Option released: record bounds.
                gOptionHeld = false;
                goRecordBounds(gMinX, gMinY, gMaxX, gMaxY);
            }
        }

        // kVK_RightShift == 0x3C (60)
        // Detect right-Shift key press (flag set + matching keycode).
        if (keycode == 0x3C && (flags & kCGEventFlagMaskShift)) {
            goHotkeyCallback();
        }
    }
    // Handle mouse movement while Right Option is held.
    else if ((type == kCGEventMouseMoved || type == kCGEventLeftMouseDragged || type == kCGEventRightMouseDragged) && gOptionHeld) {
        CGPoint loc = CGEventGetLocation(event);
        if (loc.x < gMinX) gMinX = loc.x;
        if (loc.y < gMinY) gMinY = loc.y;
        if (loc.x > gMaxX) gMaxX = loc.x;
        if (loc.y > gMaxY) gMaxY = loc.y;
    }

    return event;
}

static inline CFMachPortRef createTap(void) {
    CGEventMask mask = CGEventMaskBit(kCGEventFlagsChanged)
                     | CGEventMaskBit(kCGEventMouseMoved)
                     | CGEventMaskBit(kCGEventLeftMouseDragged)
                     | CGEventMaskBit(kCGEventRightMouseDragged);
    gTap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionListenOnly,
        mask,
        eventCallback,
        NULL
    );
    return gTap; // NULL if Accessibility permission not granted
}
*/
import "C"

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"runtime"
	"sync"
)

var (
	triggerCh = make(chan struct{}, 10)
	boundsCh  = make(chan image.Rectangle, 1)
	startOnce sync.Once
)

//export goHotkeyCallback
func goHotkeyCallback() {
	// Non-blocking send so the run-loop is never stalled.
	select {
	case triggerCh <- struct{}{}:
	default:
	}
}

//export goRecordBounds
func goRecordBounds(minX, minY, maxX, maxY float64) {
	// Drain and replace: always keep the latest bounds.
	select {
	case <-boundsCh:
	default:
	}
	rect := image.Rect(int(minX), int(minY), int(maxX), int(maxY))
	select {
	case boundsCh <- rect:
	default:
	}
}

// Listen starts the global event tap on a dedicated OS thread and returns channels:
// - triggers: receives a value each time the right Option key is pressed
// - bounds: receives updated screen-coordinate bounds when right Shift is released
// The caller must have Accessibility permission (System Settings → Privacy &
// Security → Accessibility). The event tap runs until parentCtx is cancelled.
func Listen(parentCtx context.Context, logger *slog.Logger) (<-chan struct{}, <-chan image.Rectangle, error) {
	var listenErr error

	startOnce.Do(func() {
		tap := C.createTap()
		if tap == 0 {
			listenErr = fmt.Errorf("CGEventTapCreate failed — grant Accessibility permission to this app")
			return
		}

		// Run the tap on a background thread with its own CFRunLoop.
		go func() {
			runtime.LockOSThread()
			source := C.CFMachPortCreateRunLoopSource(C.kCFAllocatorDefault, tap, 0)
			rl := C.CFRunLoopGetCurrent()
			C.CFRunLoopAddSource(rl, source, C.kCFRunLoopCommonModes)
			C.CGEventTapEnable(tap, C.bool(true))

			logger.Info("Hotkey listener started (right Shift key, right Option for bounds)")

			// Poll the run loop with a timeout so we can check context cancellation.
			for parentCtx.Err() == nil {
				C.CFRunLoopRunInMode(C.kCFRunLoopDefaultMode, 0.5, 0)
			}

			// Cleanup on context cancellation.
			C.CGEventTapEnable(tap, C.bool(false))
			C.CFRunLoopRemoveSource(rl, source, C.kCFRunLoopCommonModes)
			C.CFRelease(C.CFTypeRef(source))
			C.CFRelease(C.CFTypeRef(tap))
			logger.Info("Hotkey listener stopped")
		}()
	})

	if listenErr != nil {
		return nil, nil, listenErr
	}

	// Drain residual triggers and bounds on shutdown so nothing leaks.
	go func() {
		<-parentCtx.Done()
		for len(triggerCh) > 0 {
			<-triggerCh
		}
		for len(boundsCh) > 0 {
			<-boundsCh
		}
	}()

	return triggerCh, boundsCh, nil
}
