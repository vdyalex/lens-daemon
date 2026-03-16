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

// Configurable hotkey keycodes.
static CGKeyCode gTriggerKeycode = 0x3C;  // Right Shift by default
static CGKeyCode gBoundsKeycode = 0x3D;   // Right Option by default

// Right Option bounds tracking state.
static bool gOptionHeld = false;
static CGFloat gMinX, gMinY, gMaxX, gMaxY;

// setKeycodes sets the trigger and bounds hotkey keycodes.
static inline void setKeycodes(int trigger, int bounds) {
    gTriggerKeycode = (CGKeyCode)trigger;
    gBoundsKeycode = (CGKeyCode)bounds;
}

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

    // Handle bounds tracking hotkey.
    if (type == kCGEventFlagsChanged) {
        CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
        CGEventFlags flags = CGEventGetFlags(event);

        // Bounds tracking on configured keycode.
        if (keycode == gBoundsKeycode) {
            bool optionNowHeld = (flags & kCGEventFlagMaskAlternate) != 0;
            if (optionNowHeld && !gOptionHeld) {
                // Bounds key pressed: start tracking.
                gOptionHeld = true;
                CGPoint loc = CGEventGetLocation(event);
                gMinX = gMaxX = loc.x;
                gMinY = gMaxY = loc.y;
            } else if (!optionNowHeld && gOptionHeld) {
                // Bounds key released: record bounds.
                gOptionHeld = false;
                goRecordBounds(gMinX, gMinY, gMaxX, gMaxY);
            }
        }

        // Detect capture trigger on configured keycode.
        if (keycode == gTriggerKeycode && (flags & kCGEventFlagMaskShift)) {
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
	"image"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// Listener manages global hotkey detection and bounds tracking.
type Listener struct {
	triggerCh chan struct{}
	boundsCh  chan image.Rectangle
	startOnce sync.Once
}

// current holds the active listener instance for CGo callbacks.
var current *Listener

//export goHotkeyCallback
func goHotkeyCallback() {
	if current != nil {
		// Non-blocking send so the run-loop is never stalled.
		select {
		case current.triggerCh <- struct{}{}:
		default:
		}
	}
}

//export goRecordBounds
func goRecordBounds(minX, minY, maxX, maxY float64) {
	if current != nil {
		// Drain and replace: always keep the latest bounds.
		select {
		case <-current.boundsCh:
		default:
		}
		rect := image.Rect(int(minX), int(minY), int(maxX), int(maxY))
		select {
		case current.boundsCh <- rect:
		default:
		}
	}
}

// New creates a new listener instance.
func New() *Listener {
	return &Listener{
		triggerCh: make(chan struct{}, 10),
		boundsCh:  make(chan image.Rectangle, 1),
	}
}

// Listen starts the event tap on a dedicated OS thread and returns channels:
// - triggers: receives a value each time the trigger hotkey is pressed
// - bounds: receives updated screen-coordinate bounds when the bounds hotkey is released
// The caller must have Accessibility permission (System Settings → Privacy &
// Security → Accessibility). The event tap runs until parentCtx is cancelled.
// pollInterval is the CFRunLoop polling timeout; smaller values increase responsiveness but use more CPU.
// triggerKeycode and boundsKeycode are the MacOS virtual keycodes for the hotkeys.
// Listen can only be called once per Listener instance; subsequent calls return the same channels.
func (listener *Listener) Listen(parentCtx context.Context, logger *slog.Logger, pollInterval time.Duration, triggerKeycode, boundsKeycode int) (<-chan struct{}, <-chan image.Rectangle, error) {
	var listenErr error

	listener.startOnce.Do(func() {
		current = listener // Register this listener as the active one for CGo callbacks

		C.setKeycodes(C.int(triggerKeycode), C.int(boundsKeycode))

		tap := C.createTap()
		if tap == 0 {
			listenErr = exceptions.ListenerEventTapCreateFailedException
			return
		}

		// Run the tap on a background thread with its own CFRunLoop.
		go func() {
			runtime.LockOSThread()
			source := C.CFMachPortCreateRunLoopSource(C.kCFAllocatorDefault, tap, 0)
			rl := C.CFRunLoopGetCurrent()
			C.CFRunLoopAddSource(rl, source, C.kCFRunLoopCommonModes)
			C.CGEventTapEnable(tap, C.bool(true))

			logger.Info("Hotkey listener started", "trigger_keycode", triggerKeycode, "bounds_keycode", boundsKeycode)

			// Poll the run loop with a timeout so we can check context cancellation.
			for parentCtx.Err() == nil {
				C.CFRunLoopRunInMode(C.kCFRunLoopDefaultMode, C.double(pollInterval.Seconds()), 0)
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
		for len(listener.triggerCh) > 0 {
			<-listener.triggerCh
		}
		for len(listener.boundsCh) > 0 {
			<-listener.boundsCh
		}
	}()

	return listener.triggerCh, listener.boundsCh, nil
}
