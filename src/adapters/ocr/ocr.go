package ocr

import (
	"github.com/vdyalex/lens-daemon/src/bridges/vision"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// RecognizeText calls the real Vision framework bridge.
func (RealVisionBridge) RecognizeText(pngData []byte, language string, accurate int) string {
	return vision.RecognizeText(pngData, language, accurate)
}

// New creates a Vision framework OCR client with the specified language and accuracy level.
// language should be a BCP 47 code (e.g., "en-US", "zh-Hans", "ja", "ko").
// accuracy should be "accurate" or "fast"; any other value defaults to "accurate".
func New(language, accuracy string) *Client {
	return NewWithBridge(language, accuracy, RealVisionBridge{})
}

// NewWithBridge creates a Vision framework OCR client with an injectable bridge.
// This is primarily used for testing.
func NewWithBridge(language, accuracy string, bridge VisionBridge) *Client {
	return &Client{
		bridge:   bridge,
		language: language,
		accuracy: accuracy != "fast", // true unless explicitly "fast"
	}
}

// RecognizeText recognizes text in PNG data using the Vision framework.
// Input: PNG-encoded image bytes.
// Output: Recognized text as a string.
func (c *Client) RecognizeText(pngData []byte) (string, error) {
	if len(pngData) == 0 {
		return "", exceptions.ErrOCREmptyInput
	}

	accurateC := 0
	if c.accuracy {
		accurateC = 1
	}

	result := c.bridge.RecognizeText(pngData, c.language, accurateC)
	if result == "" {
		return "", exceptions.ErrOCRFailed
	}

	return result, nil
}
