package extractor

import "image"

// OCRClient abstracts the OCR engine for testability.
type OCRClient interface {
	RecognizeText(pngData []byte) (string, error)
}

// Service abstracts text extraction from images.
type Service interface {
	Extract(img *image.RGBA) (string, error)
	Close() error
}

// Extractor extracts text from an in-memory image using Apple's Vision framework.
type Extractor struct {
	client OCRClient
}
