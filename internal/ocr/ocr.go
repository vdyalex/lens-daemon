package ocr

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
func New(lang string) (*Extractor, error) {
	client := gosseract.NewClient()
	if err := client.SetLanguage(lang); err != nil {
		client.Close()
		return nil, fmt.Errorf("set tesseract language %q: %w", lang, err)
	}
	return &Extractor{client: client}, nil
}

// Extract takes an in-memory image and returns the recognized text.
// No files are written to disk.
func (e *Extractor) Extract(img *image.RGBA) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("encode image to png: %w", err)
	}

	if err := e.client.SetImageFromBytes(buf.Bytes()); err != nil {
		return "", fmt.Errorf("set image from bytes: %w", err)
	}

	text, err := e.client.Text()
	if err != nil {
		return "", fmt.Errorf("tesseract OCR: %w", err)
	}

	return text, nil
}

// Close releases the Tesseract client resources.
func (e *Extractor) Close() error {
	return e.client.Close()
}
