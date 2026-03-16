package extractor

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/vdyalex/lens-daemon/src/adapters/vision"
)

// Extractor extracts text from an in-memory image.
// Callers must call Close to release resources.
type Extractor interface {
	// Extract converts the image to text.
	// No files are written to disk.
	Extract(image *image.RGBA) (string, error)

	// Close releases underlying resources.
	Close() error
}

// encodeImage encodes an RGBA image to PNG bytes.
func encodeImage(image *image.RGBA) ([]byte, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image); err != nil {
		return nil, fmt.Errorf("encode image to PNG: %w", err)
	}
	return buffer.Bytes(), nil
}

// VisionExtractor uses the Vision framework adapter for OCR.
type VisionExtractor struct {
	client *vision.Client
}

// New creates an extractor using the Vision framework adapter.
// language should be a BCP 47 code (e.g., "en-US", "zh-Hans", "ja", "ko").
func New(language string) (Extractor, error) {
	return &VisionExtractor{
		client: vision.New(language),
	}, nil
}

// Extract recognizes text in the image.
// No files are written to disk.
func (extractor *VisionExtractor) Extract(file *image.RGBA) (string, error) {
	pngData, err := encodeImage(file)
	if err != nil {
		return "", err
	}

	return extractor.client.RecognizeText(pngData)
}

// Close is a no-op (Vision framework has no persistent resources to release).
func (extractor *VisionExtractor) Close() error {
	return nil
}
