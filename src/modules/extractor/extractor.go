package extractor

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/vdyalex/lens-daemon/src/adapters/vision"
)

// encodeImage encodes an RGBA image to PNG bytes.
func encodeImage(image *image.RGBA) ([]byte, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image); err != nil {
		return nil, fmt.Errorf("encode image to PNG: %w", err)
	}
	return buffer.Bytes(), nil
}

// Extractor extracts text from an in-memory image using Apple's Vision framework.
type Extractor struct {
	client *vision.Client
}

// New creates an extractor using the Vision framework adapter.
// language should be a BCP 47 code (e.g., "en-US", "zh-Hans", "ja", "ko").
// accuracy should be "accurate" or "fast"; any other value defaults to "accurate".
func New(language, accuracy string) *Extractor {
	return &Extractor{
		client: vision.New(language, accuracy),
	}
}

// Extract recognizes text in the image.
// No files are written to disk.
func (extractor *Extractor) Extract(file *image.RGBA) (string, error) {
	pngData, err := encodeImage(file)
	if err != nil {
		return "", err
	}

	return extractor.client.RecognizeText(pngData)
}

// Close is a no-op (Vision framework has no persistent resources to release).
func (extractor *Extractor) Close() error {
	return nil
}
