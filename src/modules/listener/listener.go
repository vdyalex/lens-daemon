package listener

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

// Forward declaration – implemented in Go via export.
extern void goHotkeyCallback(void);

// Global tap reference so the callback can re-enable it on timeout.
static CFMachPortRef gTap = NULL;

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

    // kVK_RightOption == 0x3D (61)
    CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
    CGEventFlags flags = CGEventGetFlags(event);

    // Detect right-Option key press (flag set + matching keycode).
    if (keycode == 0x3D && (flags & kCGEventFlagMaskAlternate)) {
        goHotkeyCallback();
    }

    return event;
}

static inline CFMachPortRef createTap(void) {
    CGEventMask mask = CGEventMaskBit(kCGEventFlagsChanged);
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
	"log/slog"
	"runtime"
	"sync"
)

var (
	triggerCh = make(chan struct{}, 1)
	startOnce sync.Once
	ctx       context.Context
	done      context.CancelFunc
)

//export goHotkeyCallback
func goHotkeyCallback() {
	// Non-blocking send so the run-loop is never stalled.
	select {
	case triggerCh <- struct{}{}:
	default:
	}
}

// Listen starts the global event tap on a dedicated OS thread and returns a
// channel that receives a value each time the right Option key is pressed.
// The caller must have Accessibility permission (System Settings → Privacy &
// Security → Accessibility).  The event tap runs until ctx is cancelled.
func Listen(ctx context.Context, logger *slog.Logger) (<-chan struct{}, error) {
	var listenErr error

	startOnce.Do(func() {
		tap := C.createTap()
		if tap == 0 {
			listenErr = fmt.Errorf("CGEventTapCreate failed — grant Accessibility permission to this app")
			return
		}

		ctx, done = context.WithCancel(context.Background())

		// Run the tap on a background thread with its own CFRunLoop.
		go func() {
			runtime.LockOSThread()
			source := C.CFMachPortCreateRunLoopSource(C.kCFAllocatorDefault, tap, 0)
			rl := C.CFRunLoopGetCurrent()
			C.CFRunLoopAddSource(rl, source, C.kCFRunLoopCommonModes)
			C.CGEventTapEnable(tap, C.bool(true))

			logger.Info("hotkey listener started (right Option key)")

			// Poll the run loop with a timeout so we can check context cancellation.
			for ctx.Err() == nil {
				C.CFRunLoopRunInMode(C.kCFRunLoopDefaultMode, 0.5, 0)
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

	// Wrap in a context-aware channel.
	out := make(chan struct{})
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				if done != nil {
					done()
				}
				return
			case <-triggerCh:
				select {
				case out <- struct{}{}:
				case <-ctx.Done():
					if done != nil {
						done()
					}
					return
				}
			}
		}
	}()

	return out, nil
}
