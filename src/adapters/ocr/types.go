package ocr

// VisionBridge abstracts the Vision framework for testability.
type VisionBridge interface {
	RecognizeText(pngData []byte, language string, accurate int) string
}

// realVisionBridge wraps the actual Vision framework calls.
type realVisionBridge struct{}

// Client wraps the Vision framework OCR adapter.
// It delegates to src/bridges/vision for CGo boundary and Vision framework calls.
// Safe to call concurrently (each call is independent).
type Client struct {
	bridge   VisionBridge
	language string
	accuracy bool // true for accurate, false for fast
}
