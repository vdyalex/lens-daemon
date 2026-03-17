package extractor_test

import (
	"errors"
	"image"
	"testing"

	"github.com/vdyalex/lens-daemon/src/modules/extractor"
)

type mockOCR struct {
	result string
	err    error
	calls  [][]byte
}

func (m *mockOCR) RecognizeText(data []byte) (string, error) {
	m.calls = append(m.calls, data)
	return m.result, m.err
}

func TestExtract_success(t *testing.T) {
	mock := &mockOCR{result: "hello world"}
	client := extractor.NewWithClient(mock)

	text, err := client.Extract(image.NewRGBA(image.Rect(0, 0, 10, 10)))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "hello world" {
		t.Errorf("expected 'hello world', got %q", text)
	}
	if len(mock.calls) != 1 {
		t.Errorf("expected 1 OCR call, got %d", len(mock.calls))
	}
	if len(mock.calls[0]) == 0 {
		t.Errorf("expected non-empty PNG bytes passed to OCR")
	}
}

func TestExtract_ocrError(t *testing.T) {
	ocrErr := errors.New("ocr failed")
	mock := &mockOCR{err: ocrErr}
	client := extractor.NewWithClient(mock)

	_, err := client.Extract(image.NewRGBA(image.Rect(0, 0, 10, 10)))

	if !errors.Is(err, ocrErr) {
		t.Errorf("expected ocrErr, got %v", err)
	}
}

func TestExtract_pngEncoding(t *testing.T) {
	mock := &mockOCR{result: "text"}
	client := extractor.NewWithClient(mock)

	client.Extract(image.NewRGBA(image.Rect(0, 0, 5, 5)))

	if len(mock.calls) == 0 {
		t.Fatal("expected OCR call, got none")
	}

	receivedBytes := mock.calls[0]
	if len(receivedBytes) == 0 {
		t.Errorf("expected PNG bytes to be non-empty")
	}
	// PNG magic bytes: \x89PNG
	if len(receivedBytes) < 4 || receivedBytes[1] != 'P' || receivedBytes[2] != 'N' || receivedBytes[3] != 'G' {
		t.Errorf("expected PNG magic bytes, got first 4 bytes: %v", receivedBytes[:4])
	}
}

func TestClose_noError(t *testing.T) {
	client := extractor.NewWithClient(&mockOCR{result: "text"})

	if err := client.Close(); err != nil {
		t.Errorf("expected nil from Close, got %v", err)
	}
}
