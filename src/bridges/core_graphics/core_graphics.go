//go:build darwin

// Package core_graphics provides CGo wrappers for CoreGraphics screen capture and display queries.
// It contains Objective-C bridges (core_graphics.m) for captureScreenRect, getMainDisplayWidth, and getMainDisplayHeight.
// This package owns the CGo boundary and handles unsafe C pointer conversions, memory management, and RGBA buffer conversion.
package core_graphics

import (
	"unsafe"
)

/*
#cgo CFLAGS: -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework CoreGraphics -framework AppKit -framework Foundation
#include <stdlib.h>
unsigned char* captureScreenRect(int x, int y, int width, int height, int* outLength, int* outWidth, int* outHeight);
int getMainDisplayWidth(void);
int getMainDisplayHeight(void);
*/
import "C"

// GetMainDisplayWidth returns the pixel width of the primary display.
func GetMainDisplayWidth() int {
	return int(C.getMainDisplayWidth())
}

// GetMainDisplayHeight returns the pixel height of the primary display.
func GetMainDisplayHeight() int {
	return int(C.getMainDisplayHeight())
}

// CaptureScreenRect captures a rectangular region of the screen and returns the pixel data as raw RGBA bytes.
// On HiDPI/Retina displays, the returned actualWidth and actualHeight reflect physical pixel dimensions
// which may differ from the requested width/height (logical coordinates) by a scaling factor.
// Returns (pixelData, actualWidth, actualHeight, true) on success, or (nil, 0, 0, false) if capture fails.
// The caller takes ownership of the returned byte slice and must not free it.
func CaptureScreenRect(x, y, width, height int) ([]byte, int, int, bool) {
	if width <= 0 || height <= 0 {
		return nil, 0, 0, false
	}

	var outLength C.int
	var outWidth C.int
	var outHeight C.int
	pixelDataPtr := C.captureScreenRect(C.int(x), C.int(y), C.int(width), C.int(height), &outLength, &outWidth, &outHeight)
	if pixelDataPtr == nil {
		return nil, 0, 0, false
	}
	defer C.free(unsafe.Pointer(pixelDataPtr))

	// Convert raw RGBA bytes to Go slice
	pixelData := unsafe.Slice((*byte)(pixelDataPtr), outLength)

	// Create a copy that outlives the C pointer
	result := make([]byte, len(pixelData))
	copy(result, pixelData)

	return result, int(outWidth), int(outHeight), true
}
