package extractor

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/otiai10/gosseract/v2"
)

// Extractor extracts text from images using Tesseract OCR.
type Extractor struct {
	client *gosseract.Client
}

// New creates a new OCR extractor with the specified language.
func New(language string) (*Extractor, error) {
	client := gosseract.NewClient()
	if err := client.SetLanguage(language); err != nil {
		client.Close()
		return nil, fmt.Errorf("Set tesseract language %q: %w", language, err)
	}
	return &Extractor{client: client}, nil
}

// Extract takes an in-memory image and returns the recognized text.
// No files are written to disk.
func (extractor *Extractor) Extract(file *image.RGBA) (string, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, file); err != nil {
		return "", fmt.Errorf("Encode image to png: %w", err)
	}

	if err := extractor.client.SetImageFromBytes(buffer.Bytes()); err != nil {
		return "", fmt.Errorf("Set image from bytes: %w", err)
	}

	text, err := extractor.client.Text()
	if err != nil {
		return "", fmt.Errorf("Tesseract OCR: %w", err)
	}

	return text, nil
}

// Close releases the Tesseract client resources.
func (extractor *Extractor) Close() error {
	return extractor.client.Close()
}
