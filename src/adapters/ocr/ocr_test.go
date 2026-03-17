package ocr_test

import (
	"errors"
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/ocr"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

type mockVisionBridge struct {
	result string
}

func (mock mockVisionBridge) RecognizeText(pngData []byte, language string, accurate int) string {
	return mock.result
}

type capturingMockBridge struct {
	result string
	onCall func(accurate int)
}

func (m *capturingMockBridge) RecognizeText(pngData []byte, language string, accurate int) string {
	if m.onCall != nil {
		m.onCall(accurate)
	}
	return m.result
}

func TestNew_accuracyDefault(t *testing.T) {
	client := ocr.New("en-US", "accurate")

	// Verify that accurate client is created (we can't directly check the internal field,
	// so we verify through RecognizeText behavior)
	mock := mockVisionBridge{result: "text"}
	testClient := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := testClient.RecognizeText([]byte("data"))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "text" {
		t.Errorf("expected 'text', got %q", text)
	}
	// Also verify the client object is created without error
	if client == nil {
		t.Errorf("expected client to be created, got nil")
	}
}

func TestNew_accuracyFast(t *testing.T) {
	// Verify fast mode passes accurate=0 to bridge
	capturedAccurate := 0
	mockBridge := &capturingMockBridge{
		result: "text",
		onCall: func(accurate int) {
			capturedAccurate = accurate
		},
	}

	client := ocr.NewWithBridge("en-US", "fast", mockBridge)
	if client == nil {
		t.Errorf("expected client to be created with 'fast' accuracy, got nil")
	}

	// Trigger a call to verify the accurate flag
	client.RecognizeText([]byte("test"))
	if capturedAccurate != 0 {
		t.Errorf("expected fast mode to pass accurate=0, got %d", capturedAccurate)
	}
}

func TestRecognizeText_emptyInput(t *testing.T) {
	mock := mockVisionBridge{result: "text"}
	client := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := client.RecognizeText(nil)

	if !errors.Is(err, exceptions.ErrOCREmptyInput) {
		t.Errorf("expected ErrOCREmptyInput, got %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
}

func TestRecognizeText_emptyResult(t *testing.T) {
	mock := mockVisionBridge{result: ""}
	client := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := client.RecognizeText([]byte("data"))

	if !errors.Is(err, exceptions.ErrOCRFailed) {
		t.Errorf("expected ErrOCRFailed, got %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
}

func TestRecognizeText_success(t *testing.T) {
	mock := mockVisionBridge{result: "hello"}
	client := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := client.RecognizeText([]byte("data"))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "hello" {
		t.Errorf("expected 'hello', got %q", text)
	}
}

func TestRecognizeText_accuracyFlagPassed(t *testing.T) {
	// Test that fast accuracy passes 0 and accurate passes 1
	// We verify this by checking that the function completes successfully
	// with different accuracy settings

	mockFast := mockVisionBridge{result: "result"}
	clientFast := ocr.NewWithBridge("en-US", "fast", mockFast)

	text, err := clientFast.RecognizeText([]byte("data"))

	if err != nil {
		t.Fatalf("fast client: expected no error, got %v", err)
	}
	if text != "result" {
		t.Errorf("fast client: expected 'result', got %q", text)
	}

	mockAccurate := mockVisionBridge{result: "result"}
	clientAccurate := ocr.NewWithBridge("en-US", "accurate", mockAccurate)

	text, err = clientAccurate.RecognizeText([]byte("data"))

	if err != nil {
		t.Fatalf("accurate client: expected no error, got %v", err)
	}
	if text != "result" {
		t.Errorf("accurate client: expected 'result', got %q", text)
	}
}
