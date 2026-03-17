//go:build darwin

// Package vision provides a CGo wrapper for Apple's Vision framework OCR.
// It contains the Objective-C bridge (vision_bridge.m) and Go wrappers for the visionRecognizeText C function.
// This package owns the CGo boundary and handles unsafe C pointer conversions and memory management.
package vision

import (
	"unsafe"
)

/*
#cgo LDFLAGS: -framework Vision -framework AppKit -framework Foundation
#include <stdlib.h>
char* visionRecognizeText(const unsigned char* pngData, size_t length, const char* language, int accurate);
*/
import "C"

// RecognizeText calls the Vision framework OCR to recognize text in PNG data.
// pngData must be non-empty PNG-encoded image bytes.
// language should be a BCP 47 language code (e.g., "en-US", "zh-Hans").
// accurate: 1 for accurate mode, 0 for fast mode.
// Returns the recognized text as a Go string.
func RecognizeText(pngData []byte, language string, accurate int) string {
	if len(pngData) == 0 {
		return ""
	}

	langC := C.CString(language)
	defer C.free(unsafe.Pointer(langC))

	result := C.visionRecognizeText(
		(*C.uchar)(unsafe.Pointer(&pngData[0])),
		C.size_t(len(pngData)),
		langC,
		(C.int)(accurate),
	)
	defer C.free(unsafe.Pointer(result))

	if result == nil {
		return ""
	}

	return C.GoString(result)
}
