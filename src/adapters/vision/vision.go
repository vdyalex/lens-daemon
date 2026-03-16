package vision

import (
	"unsafe"

	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

/*
#cgo LDFLAGS: -framework Vision -framework AppKit -framework Foundation
#include <stdlib.h>
char* visionRecognizeText(const unsigned char* pngData, size_t length, const char* language, int accurate);
*/
import "C"

// Client wraps the Vision framework OCR bridge.
// Safe to call concurrently (each call is independent).
type Client struct {
	language string
	accuracy bool // true for accurate, false for fast
}

// New creates a Vision framework OCR client with the specified language and accuracy level.
// language should be a BCP 47 code (e.g., "en-US", "zh-Hans", "ja", "ko").
// accuracy should be "accurate" or "fast"; any other value defaults to "accurate".
func New(language, accuracy string) *Client {
	return &Client{
		language: language,
		accuracy: accuracy != "fast", // true unless explicitly "fast"
	}
}

// RecognizeText recognizes text in PNG data using the Vision framework.
// Input: PNG-encoded image bytes.
// Output: Recognized text as a string.
func (client *Client) RecognizeText(pngData []byte) (string, error) {
	if len(pngData) == 0 {
		return "", exceptions.VisionEmptyInputException
	}

	langC := C.CString(client.language)
	defer C.free(unsafe.Pointer(langC))

	accurateC := 0
	if client.accuracy {
		accurateC = 1
	}

	result := C.visionRecognizeText(
		(*C.uchar)(unsafe.Pointer(&pngData[0])),
		C.size_t(len(pngData)),
		langC,
		(C.int)(accurateC),
	)
	defer C.free(unsafe.Pointer(result))

	if result == nil {
		return "", exceptions.VisionOCRFailedException
	}

	return C.GoString(result), nil
}
