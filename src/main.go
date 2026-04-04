//go:build darwin

package main

import (
	"runtime"

	"github.com/vdyalex/lens-daemon/src/cmd"
)

func init() {
	// Pin the main goroutine to the main OS thread.
	// AppKit requires the process main thread (pthread_main_np() == 1)
	// for NSApplication and NSWindow operations.
	runtime.LockOSThread()
}

func main() {
	cmd.Execute()
}
