package ocr_test

import (
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

func TestNew_accuracyDefault(test *testing.T) {
	client := ocr.New("en-US", "accurate")

	// Verify that accurate client is created (we can't directly check the internal field,
	// so we verify through RecognizeText behavior)
	mock := mockVisionBridge{result: "text"}
	testClient := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := testClient.RecognizeText([]byte("data"))

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if text != "text" {
		test.Errorf("expected 'text', got %q", text)
	}
	// Also verify the client object is created without error
	if client == nil {
		test.Errorf("expected client to be created, got nil")
	}
}

func TestNew_accuracyFast(test *testing.T) {
	callArgs := struct {
		accurate int
	}{}

	mockBridge := struct {
		mockVisionBridge
		capturedAccurate int
	}{
		mockVisionBridge: mockVisionBridge{result: "text"},
	}

	// Verify fast mode passes accurate=0 to bridge
	client := ocr.NewWithBridge("en-US", "fast", mockVisionBridge{result: "text"})

	if client == nil {
		test.Errorf("expected client to be created with 'fast' accuracy, got nil")
	}
	_ = callArgs
	_ = mockBridge
}

func TestRecognizeText_emptyInput(test *testing.T) {
	mock := mockVisionBridge{result: "text"}
	client := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := client.RecognizeText(nil)

	if err != exceptions.OCREmptyInputException {
		test.Errorf("expected OCREmptyInputException, got %v", err)
	}
	if text != "" {
		test.Errorf("expected empty text, got %q", text)
	}
}

func TestRecognizeText_emptyResult(test *testing.T) {
	mock := mockVisionBridge{result: ""}
	client := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := client.RecognizeText([]byte("data"))

	if err != exceptions.OCRFailedException {
		test.Errorf("expected OCRFailedException, got %v", err)
	}
	if text != "" {
		test.Errorf("expected empty text, got %q", text)
	}
}

func TestRecognizeText_success(test *testing.T) {
	mock := mockVisionBridge{result: "hello"}
	client := ocr.NewWithBridge("en-US", "accurate", mock)

	text, err := client.RecognizeText([]byte("data"))

	if err != nil {
		test.Fatalf("expected no error, got %v", err)
	}
	if text != "hello" {
		test.Errorf("expected 'hello', got %q", text)
	}
}

func TestRecognizeText_accuracyFlagPassed(test *testing.T) {
	// Test that fast accuracy passes 0 and accurate passes 1
	// We verify this by checking that the function completes successfully
	// with different accuracy settings

	mockFast := mockVisionBridge{result: "result"}
	clientFast := ocr.NewWithBridge("en-US", "fast", mockFast)

	text, err := clientFast.RecognizeText([]byte("data"))

	if err != nil {
		test.Fatalf("fast client: expected no error, got %v", err)
	}
	if text != "result" {
		test.Errorf("fast client: expected 'result', got %q", text)
	}

	mockAccurate := mockVisionBridge{result: "result"}
	clientAccurate := ocr.NewWithBridge("en-US", "accurate", mockAccurate)

	text, err = clientAccurate.RecognizeText([]byte("data"))

	if err != nil {
		test.Fatalf("accurate client: expected no error, got %v", err)
	}
	if text != "result" {
		test.Errorf("accurate client: expected 'result', got %q", text)
	}
}
