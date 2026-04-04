package listener

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// Forward declarations – implemented in Go via export.
extern void goHotkeyCallback(void);
extern void goRecordBounds(CGFloat minX, CGFloat minY, CGFloat maxX, CGFloat maxY);
extern void goTeleprompterToggle(void);
extern void goPositionChange(int alignment);

// Global tap reference so the callback can re-enable it on timeout.
static CFMachPortRef gTap = NULL;

// Configurable hotkey keycodes.
static CGKeyCode gTriggerKeycode       = 0x3C;  // Right Shift by default
static CGKeyCode gBoundsKeycode        = 0x3D;  // Right Option by default
static CGKeyCode gTeleprompterKeycode  = 0x36;  // Right Command by default

// Right Option bounds tracking state.
static bool gOptionHeld = false;
static bool gMouseMoved = false;
static CGFloat gMinX, gMinY, gMaxX, gMaxY;

// Right Command teleprompter toggle edge detection state.
static bool gCommandHeld = false;

// setKeycodes sets the trigger, bounds, and teleprompter hotkey keycodes.
static inline void setKeycodes(int trigger, int bounds, int teleprompter) {
    gTriggerKeycode      = (CGKeyCode)trigger;
    gBoundsKeycode       = (CGKeyCode)bounds;
    gTeleprompterKeycode = (CGKeyCode)teleprompter;
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
                gMouseMoved = false;
                CGPoint loc = CGEventGetLocation(event);
                gMinX = gMaxX = loc.x;
                gMinY = gMaxY = loc.y;
            } else if (!optionNowHeld && gOptionHeld) {
                // Bounds key released: record bounds only if cursor moved.
                gOptionHeld = false;
                if (gMouseMoved) {
                    goRecordBounds(gMinX, gMinY, gMaxX, gMaxY);
                }
            }
        }

        // Detect capture trigger on configured keycode.
        if (keycode == gTriggerKeycode && (flags & kCGEventFlagMaskShift)) {
            goHotkeyCallback();
        }

        // Detect teleprompter toggle on configured keycode (press edge only).
        if (keycode == gTeleprompterKeycode) {
            bool commandNowHeld = (flags & kCGEventFlagMaskCommand) != 0;
            if (commandNowHeld && !gCommandHeld) {
                gCommandHeld = true;
                goTeleprompterToggle();
            } else if (!commandNowHeld && gCommandHeld) {
                gCommandHeld = false;
            }
        }
    }
    // Handle mouse movement while Right Option is held.
    else if ((type == kCGEventMouseMoved || type == kCGEventLeftMouseDragged || type == kCGEventRightMouseDragged) && gOptionHeld) {
        gMouseMoved = true;
        CGPoint loc = CGEventGetLocation(event);
        if (loc.x < gMinX) gMinX = loc.x;
        if (loc.y < gMinY) gMinY = loc.y;
        if (loc.x > gMaxX) gMaxX = loc.x;
        if (loc.y > gMaxY) gMaxY = loc.y;
    }
    // Handle arrow keys while bounds key is held for position rotation.
    // 0x7B=Left (backward), 0x7C=Right (forward)
    else if (type == kCGEventKeyDown && gOptionHeld) {
        CGKeyCode key = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
        if (key == 0x7B) goPositionChange(-1);      // rotate backward
        else if (key == 0x7C) goPositionChange(1);   // rotate forward
    }

    return event;
}

static inline CFMachPortRef createTap(void) {
    CGEventMask mask = CGEventMaskBit(kCGEventFlagsChanged)
                     | CGEventMaskBit(kCGEventMouseMoved)
                     | CGEventMaskBit(kCGEventLeftMouseDragged)
                     | CGEventMaskBit(kCGEventRightMouseDragged)
                     | CGEventMaskBit(kCGEventKeyDown);
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
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

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

//export goTeleprompterToggle
func goTeleprompterToggle() {
	if current != nil {
		// Non-blocking send so the run-loop is never stalled.
		select {
		case current.teleprompterCh <- struct{}{}:
		default:
		}
	}
}

//export goPositionChange
func goPositionChange(alignment C.int) {
	if current != nil {
		// Drain and replace: always keep the latest position.
		select {
		case <-current.positionCh:
		default:
		}
		select {
		case current.positionCh <- int(alignment):
		default:
		}
	}
}

// New creates a new listener instance.
func New() *Listener {
	return &Listener{
		triggerCh:      make(chan struct{}, constants.ListenerTriggerChannelBuffer),
		boundsCh:       make(chan image.Rectangle, 1),
		teleprompterCh: make(chan struct{}, constants.ListenerTriggerChannelBuffer),
		positionCh:     make(chan int, 1),
	}
}

// Listen starts the event tap on a dedicated OS thread and returns channels:
//   - triggers: receives a value each time the trigger hotkey is pressed
//   - bounds: receives updated screen-coordinate bounds when the bounds hotkey is released
//   - teleprompter: receives a value each time the teleprompter toggle hotkey is pressed
//
// The caller must have Accessibility permission (System Settings > Privacy &
// Security > Accessibility). The event tap runs until parentCtx is cancelled.
// pollInterval is the CFRunLoop polling timeout; smaller values increase responsiveness but use more CPU.
// triggerKeycode, boundsKeycode, and teleprompterKeycode are the MacOS virtual keycodes for the hotkeys.
// Listen can only be called once per Listener instance; subsequent calls return the same channels.
func (l *Listener) Listen(parentCtx context.Context, logger *slog.Logger, pollInterval time.Duration, triggerKeycode, boundsKeycode, teleprompterKeycode int) (<-chan struct{}, <-chan image.Rectangle, <-chan struct{}, <-chan int, error) {
	var listenErr error

	l.startOnce.Do(func() {
		current = l // Register this listener as the active one for CGo callbacks

		C.setKeycodes(C.int(triggerKeycode), C.int(boundsKeycode), C.int(teleprompterKeycode))

		tap := C.createTap()
		if tap == 0 {
			listenErr = exceptions.ErrListenerEventTapCreateFailed
			return
		}

		// Run the tap on a background thread with its own CFRunLoop.
		go func() {
			runtime.LockOSThread()
			source := C.CFMachPortCreateRunLoopSource(C.kCFAllocatorDefault, tap, 0)
			rl := C.CFRunLoopGetCurrent()
			C.CFRunLoopAddSource(rl, source, C.kCFRunLoopCommonModes)
			C.CGEventTapEnable(tap, C.bool(true))

			logger.Info("hotkey listener started",
				"trigger_keycode", triggerKeycode,
				"bounds_keycode", boundsKeycode,
				"teleprompter_keycode", teleprompterKeycode,
			)

			// Poll the run loop with a short timeout to check context cancellation responsively.
			// EventTapRunLoopTimeout ensures graceful shutdown within ~50ms of context cancellation.
			for parentCtx.Err() == nil {
				C.CFRunLoopRunInMode(C.kCFRunLoopDefaultMode, C.double(constants.EventTapRunLoopTimeout.Seconds()), 0)
			}

			// Cleanup on context cancellation.
			C.CGEventTapEnable(tap, C.bool(false))
			C.CFRunLoopRemoveSource(rl, source, C.kCFRunLoopCommonModes)
			C.CFRelease(C.CFTypeRef(source))
			C.CFRelease(C.CFTypeRef(tap))
			logger.Info("hotkey listener stopped")
		}()
	})

	if listenErr != nil {
		return nil, nil, nil, nil, listenErr
	}

	// Drain residual events on shutdown so nothing leaks.
	go func() {
		<-parentCtx.Done()
		for len(l.triggerCh) > 0 {
			<-l.triggerCh
		}
		for len(l.boundsCh) > 0 {
			<-l.boundsCh
		}
		for len(l.teleprompterCh) > 0 {
			<-l.teleprompterCh
		}
		for len(l.positionCh) > 0 {
			<-l.positionCh
		}
	}()

	return l.triggerCh, l.boundsCh, l.teleprompterCh, l.positionCh, nil
}
