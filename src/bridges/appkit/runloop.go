//go:build darwin

// Package appkit provides CGo wrappers for macOS AppKit framework operations.
// It owns the NSApplication run loop and overlay window lifecycle.
// This package owns the CGo boundary and handles unsafe C pointer conversions and thread management.
package appkit

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

// initNSApp initialises NSApplication as an accessory (no Dock icon, no Cmd+Tab).
// Must be called on the process main thread before any window operations.
static void initNSApp(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

// pumpRunLoop runs the Cocoa run loop for a short interval.
// Returns immediately if there are no pending events.
static void pumpRunLoop(void) {
    @autoreleasepool {
        NSEvent* event;
        while ((event = [NSApp nextEventMatchingMask:NSEventMaskAny
                                           untilDate:nil
                                              inMode:NSDefaultRunLoopMode
                                             dequeue:YES])) {
            [NSApp sendEvent:event];
        }
    }
}
*/
import "C"

import (
	"runtime"
	"time"
	"unsafe"
)

// workItem carries a function to execute on the main thread and a channel to
// return the result.
type workItem struct {
	action func() unsafe.Pointer
	done   chan unsafe.Pointer
}

// mainQueue serialises work items that need the main OS thread.
var mainQueue = make(chan workItem, 64)

// StartRunLoop runs the AppKit event loop on the current goroutine.
// The calling goroutine MUST be the main goroutine (pinned to the main OS thread
// via runtime.LockOSThread in init()).
//
// StartRunLoop never returns. It drains mainQueue, executes each work item on the
// main thread, then pumps the Cocoa run loop for pending events.
func StartRunLoop() {
	runtime.LockOSThread()
	C.initNSApp()

	ticker := time.NewTicker(16 * time.Millisecond) // ~60 Hz
	defer ticker.Stop()

	for {
		select {
		case item := <-mainQueue:
			result := item.action()
			item.done <- result
		case <-ticker.C:
			// Pump Cocoa events so performSelectorOnMainThread and
			// dispatch_get_main_queue blocks execute.
		}
		C.pumpRunLoop()
	}
}

// RunOnMainThread dispatches a function onto the main OS thread and blocks
// until it completes. Safe to call from any goroutine.
//
// action: performs AppKit work and returns a pointer result (or nil).
// Returns the value returned by action.
func RunOnMainThread(action func() unsafe.Pointer) unsafe.Pointer {
	done := make(chan unsafe.Pointer, 1)
	mainQueue <- workItem{action: action, done: done}
	return <-done
}
