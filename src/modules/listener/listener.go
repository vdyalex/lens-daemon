package listener

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// Forward declarations – implemented in Go via export.
extern void goHotkeyCallback(void);
extern void goRecordBounds(CGFloat minX, CGFloat minY, CGFloat maxX, CGFloat maxY);
extern void goTeleprompterToggle(void);
extern void goPositionChangeXY(int dx, int dy);
extern void goOpacityChange(int direction);
extern void goFontSizeChange(int direction);

// Global tap reference so the callback can re-enable it on timeout.
static CFMachPortRef gTap = NULL;

// Configurable hotkey keycodes.
static CGKeyCode gTriggerKeycode       = 0x3C;  // Right Shift by default
static CGKeyCode gBoundsKeycode        = 0x3D;  // Right Option by default
static CGKeyCode gToggleKeycode  = 0x36;  // Right Command by default

// Right Option bounds tracking state.
static bool gOptionHeld  = false;
static bool gMouseMoved  = false;
static bool gControlUsed = false; // true if arrows/opacity keys were pressed during this hold
static CGFloat gMinX, gMinY, gMaxX, gMaxY;

// Right Command teleprompter toggle edge detection state.
static bool gCommandHeld = false;

// setKeycodes sets the trigger, bounds, and toggle hotkey keycodes.
static inline void setKeycodes(int trigger, int bounds, int toggle) {
    gTriggerKeycode = (CGKeyCode)trigger;
    gBoundsKeycode  = (CGKeyCode)bounds;
    gToggleKeycode  = (CGKeyCode)toggle;
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
                gOptionHeld  = true;
                gMouseMoved  = false;
                gControlUsed = false;
                CGPoint loc = CGEventGetLocation(event);
                gMinX = gMaxX = loc.x;
                gMinY = gMaxY = loc.y;
            } else if (!optionNowHeld && gOptionHeld) {
                // Bounds key released: record bounds only if cursor moved and no
                // grid/opacity keys were pressed during this hold.
                gOptionHeld = false;
                if (gMouseMoved && !gControlUsed) {
                    goRecordBounds(gMinX, gMinY, gMaxX, gMaxY);
                }
            }
        }

        // Detect capture trigger on configured keycode.
        if (keycode == gTriggerKeycode && (flags & kCGEventFlagMaskShift)) {
            goHotkeyCallback();
        }

        // Detect teleprompter toggle on configured keycode (press edge only).
        if (keycode == gToggleKeycode) {
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
    // Handle arrow keys while bounds key is held for grid navigation.
    // 0x7B=Left, 0x7C=Right, 0x7E=Up, 0x7D=Down
    else if (type == kCGEventKeyDown && gOptionHeld) {
        gControlUsed = true;
        CGKeyCode key = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
        if      (key == 0x7B) goPositionChangeXY(-1,  0); // Left:  move column back
        else if (key == 0x7C) goPositionChangeXY( 1,  0); // Right: move column forward
        else if (key == 0x7E) goPositionChangeXY( 0, -1); // Up:    move row back
        else if (key == 0x7D) goPositionChangeXY( 0,  1); // Down:  move row forward
        else if (key == 0x1B) goOpacityChange(-1);         // Minus: decrease opacity
        else if (key == 0x18) goOpacityChange(1);          // Plus:  increase opacity
        else if (key == 0x1D) goOpacityChange(0);          // Zero:  reset opacity
        else if (key == 0x2B) goFontSizeChange(-1);        // Comma:         decrease font size
        else if (key == 0x2F) goFontSizeChange(1);         // Period:        increase font size
        else if (key == 0x2C) goFontSizeChange(0);         // Slash/Question: reset font size
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
		case current.togglesCh <- struct{}{}:
		default:
		}
	}
}

//export goPositionChangeXY
func goPositionChangeXY(dx, dy C.int) {
	if current != nil {
		direction := [2]int{int(dx), int(dy)}
		// Drain and replace: always keep the latest direction.
		select {
		case <-current.teleprompterGridPositionCh:
		default:
		}
		select {
		case current.teleprompterGridPositionCh <- direction:
		default:
		}
	}
}

//export goOpacityChange
func goOpacityChange(direction C.int) {
	if current != nil {
		select {
		case current.teleprompterOverlayOpacityCh <- int(direction):
		default:
		}
	}
}

//export goFontSizeChange
func goFontSizeChange(direction C.int) {
	if current != nil {
		select {
		case current.teleprompterTextFontSizeCh <- int(direction):
		default:
		}
	}
}

// New creates a new listener instance.
func New() *Listener {
	return &Listener{
		triggerCh:                    make(chan struct{}, constants.ListenerTriggerChannelBuffer),
		boundsCh:                     make(chan image.Rectangle, 1),
		togglesCh:                    make(chan struct{}, constants.ListenerTriggerChannelBuffer),
		teleprompterGridPositionCh:   make(chan [2]int, 1),
		teleprompterOverlayOpacityCh: make(chan int, 1),
		teleprompterTextFontSizeCh:   make(chan int, 1),
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
// triggerKeycode, boundsKeycode, and toggleKeycode are the MacOS virtual keycodes for the hotkeys.
// Listen can only be called once per Listener instance; subsequent calls return the same channels.
func (l *Listener) Listen(parentCtx context.Context, logger *slog.Logger, pollInterval time.Duration, triggerKeycode, boundsKeycode, toggleKeycode int) (*Channels, error) {
	var listenErr error

	l.startOnce.Do(func() {
		current = l // Register this listener as the active one for CGo callbacks

		C.setKeycodes(C.int(triggerKeycode), C.int(boundsKeycode), C.int(toggleKeycode))

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
				"toggle_keycode", toggleKeycode,
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
		return nil, listenErr
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
		for len(l.togglesCh) > 0 {
			<-l.togglesCh
		}
		for len(l.teleprompterGridPositionCh) > 0 {
			<-l.teleprompterGridPositionCh
		}
		for len(l.teleprompterOverlayOpacityCh) > 0 {
			<-l.teleprompterOverlayOpacityCh
		}
		for len(l.teleprompterTextFontSizeCh) > 0 {
			<-l.teleprompterTextFontSizeCh
		}
	}()

	return &Channels{
		Triggers:                     l.triggerCh,
		Bounds:                       l.boundsCh,
		Toggles:                      l.togglesCh,
		TeleprompterGridPositions:    l.teleprompterGridPositionCh,
		TeleprompterOverlayOpacities: l.teleprompterOverlayOpacityCh,
		TeleprompterTextFontSizes:    l.teleprompterTextFontSizeCh,
	}, nil
}
