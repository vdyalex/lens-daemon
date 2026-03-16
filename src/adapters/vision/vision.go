package vision

import (
	"fmt"
	"unsafe"
)

/*
#cgo LDFLAGS: -framework Vision -framework AppKit -framework Foundation
#include <stdlib.h>
char* visionRecognizeText(const unsigned char* pngData, size_t length, const char* language);
*/
import "C"

// Client wraps the Vision framework OCR bridge.
// Safe to call concurrently (each call is independent).
type Client struct {
	language string
}

// New creates a Vision framework OCR client with the specified language.
// language should be a BCP 47 code (e.g., "en-US", "zh-Hans", "ja", "ko").
func New(language string) *Client {
	return &Client{language: language}
}

// RecognizeText recognizes text in PNG data using the Vision framework.
// Input: PNG-encoded image bytes.
// Output: Recognized text as a string.
func (client *Client) RecognizeText(pngData []byte) (string, error) {
	if len(pngData) == 0 {
		return "", fmt.Errorf("empty PNG data")
	}

	langC := C.CString(client.language)
	defer C.free(unsafe.Pointer(langC))

	result := C.visionRecognizeText(
		(*C.uchar)(unsafe.Pointer(&pngData[0])),
		C.size_t(len(pngData)),
		langC,
	)
	defer C.free(unsafe.Pointer(result))

	if result == nil {
		return "", fmt.Errorf("Vision OCR returned null")
	}

	return C.GoString(result), nil
}
