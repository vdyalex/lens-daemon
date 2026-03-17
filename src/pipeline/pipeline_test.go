package pipeline_test

import (
	"context"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/pipeline"
	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

func TestProcess_noForegroundWindow(test *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return nil, exceptions.CapturerNoForegroundWindowException
		},
	}

	client := createTestPipeline(test, mockCapturer, nil, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		test.Errorf("expected no error (non-fatal skip), got %v", err)
	}
}

func TestProcess_capturerError(test *testing.T) {
	testErr := fmt.Errorf("capturer failed")
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return nil, testErr
		},
	}

	client := createTestPipeline(test, mockCapturer, nil, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != testErr {
		test.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_captureTimeout(test *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			// Simulate timeout by blocking
			time.Sleep(1 * time.Second)
			return nil, nil
		},
	}

	settings := createTestConfig()
	settings.TimeoutCapture = 10 * time.Millisecond // Short timeout
	logger := mocks.NopLogger()

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "", nil
		},
	}

	mockAgent := &mocks.MockAIProcessor{
		ProcessFunc: func(ctx context.Context, text string) (string, error) {
			return "", nil
		},
	}

	mockBroadcaster := &mocks.MockIMBroadcaster{
		BroadcastFunc: func(ctx context.Context, text string) error {
			return nil
		},
	}

	mockPoller := &mocks.MockPollerService{}

	client := pipeline.NewWithDependencies(settings, logger, mockCapturer, mockExtractor, mockAgent, mockBroadcaster, mockPoller)

	ctx := context.Background()
	err := client.Process(ctx)

	if err == nil || !containsError(err, "timeout") {
		test.Errorf("expected capture timeout error, got %v", err)
	}
}

func TestProcess_ocrEmpty(test *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "", nil
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		test.Errorf("expected no error (non-fatal skip for empty OCR), got %v", err)
	}
}

func TestProcess_ocrError(test *testing.T) {
	testErr := fmt.Errorf("OCR failed")
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "", testErr
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != testErr {
		test.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_agentEmpty(test *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "extracted text", nil
		},
	}

	mockAgent := &mocks.MockAIProcessor{
		ProcessFunc: func(ctx context.Context, text string) (string, error) {
			return "", nil
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, mockAgent, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		test.Errorf("expected no error (non-fatal skip for empty agent response), got %v", err)
	}
}

func TestProcess_agentError(test *testing.T) {
	testErr := fmt.Errorf("AI failed")
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "extracted text", nil
		},
	}

	mockAgent := &mocks.MockAIProcessor{
		ProcessFunc: func(ctx context.Context, text string) (string, error) {
			return "", testErr
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, mockAgent, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != testErr {
		test.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_broadcastError(test *testing.T) {
	testErr := fmt.Errorf("Broadcast failed")
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "extracted text", nil
		},
	}

	mockAgent := &mocks.MockAIProcessor{
		ProcessFunc: func(ctx context.Context, text string) (string, error) {
			return "agent response", nil
		},
	}

	mockBroadcaster := &mocks.MockIMBroadcaster{
		BroadcastFunc: func(ctx context.Context, text string) error {
			return testErr
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, mockAgent, mockBroadcaster)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != testErr {
		test.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_happyPath(test *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "extracted text", nil
		},
	}

	mockAgent := &mocks.MockAIProcessor{
		ProcessFunc: func(ctx context.Context, text string) (string, error) {
			return "agent response", nil
		},
	}

	mockBroadcaster := &mocks.MockIMBroadcaster{
		BroadcastFunc: func(ctx context.Context, text string) error {
			return nil
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, mockAgent, mockBroadcaster)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		test.Errorf("expected no error, got %v", err)
	}

	if len(mockBroadcaster.Calls) != 1 {
		test.Errorf("expected 1 broadcast call, got %d", len(mockBroadcaster.Calls))
	}
	if len(mockBroadcaster.Calls) > 0 && mockBroadcaster.Calls[0] != "agent response" {
		test.Errorf("expected broadcast to be called with 'agent response', got %q", mockBroadcaster.Calls[0])
	}
}

func TestProcess_textTrimmed(test *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{
				Title:  "Test Window",
				X:      0,
				Y:      0,
				Width:  100,
				Height: 100,
			}, nil
		},
		CaptureCenterFunc: func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	var capturedText string
	mockExtractor := &mocks.MockExtractorService{
		ExtractFunc: func(img *image.RGBA) (string, error) {
			return "  extracted text  ", nil
		},
	}

	mockAgent := &mocks.MockAIProcessor{
		ProcessFunc: func(ctx context.Context, text string) (string, error) {
			capturedText = text
			return "response", nil
		},
	}

	mockBroadcaster := &mocks.MockIMBroadcaster{
		BroadcastFunc: func(ctx context.Context, text string) error {
			return nil
		},
	}

	client := createTestPipeline(test, mockCapturer, mockExtractor, mockAgent, mockBroadcaster)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		test.Errorf("expected no error, got %v", err)
	}

	if capturedText != "extracted text" {
		test.Errorf("expected trimmed text 'extracted text', got %q", capturedText)
	}
}

// Helper functions

func createTestConfig() *config.Config {
	return &config.Config{
		TimeoutForegroundWindow:  5 * time.Second,
		TimeoutCapture:           30 * time.Second,
		TimeoutOCRExtract:        30 * time.Second,
		TimeoutAIProcess:         60 * time.Second,
		TelegramBroadcastTimeout: 30 * time.Second,
		TimeoutPipelineOverall:   5 * time.Minute,
	}
}

func createTestPipeline(test *testing.T, capturer capturer.Service, extractor extractor.Service, agent ai.Processor, broadcaster im.Broadcaster) *pipeline.Pipeline {
	settings := createTestConfig()
	logger := mocks.NopLogger()

	if extractor == nil {
		extractor = &mocks.MockExtractorService{
			ExtractFunc: func(img *image.RGBA) (string, error) {
				return "", nil
			},
		}
	}

	if agent == nil {
		agent = &mocks.MockAIProcessor{
			ProcessFunc: func(ctx context.Context, text string) (string, error) {
				return "", nil
			},
		}
	}

	if broadcaster == nil {
		broadcaster = &mocks.MockIMBroadcaster{
			BroadcastFunc: func(ctx context.Context, text string) error {
				return nil
			},
		}
	}

	return pipeline.NewWithDependencies(settings, logger, capturer, extractor, agent, broadcaster, &mocks.MockPollerService{})
}

func containsError(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
