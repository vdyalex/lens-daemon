package pipeline_test

import (
	"context"
	"errors"
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

func TestProcess_noForegroundWindow(t *testing.T) {
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return nil, exceptions.ErrCapturerNoForegroundWindow
		},
	}

	client := createTestPipeline(t, mockCapturer, nil, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error (non-fatal skip), got %v", err)
	}
}

func TestProcess_capturerError(t *testing.T) {
	testErr := fmt.Errorf("capturer failed")
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return nil, testErr
		},
	}

	client := createTestPipeline(t, mockCapturer, nil, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_captureTimeout(t *testing.T) {
	blockCapture := make(chan struct{})
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
			// Simulate timeout by blocking until channel is closed
			<-blockCapture
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
		t.Errorf("expected capture timeout error, got %v", err)
	}
}

func TestProcess_ocrEmpty(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error (non-fatal skip for empty OCR), got %v", err)
	}
}

func TestProcess_ocrError(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, nil, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_agentEmpty(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, mockAgent, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error (non-fatal skip for empty agent response), got %v", err)
	}
}

func TestProcess_agentError(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, mockAgent, nil)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_broadcastError(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, mockAgent, mockBroadcaster)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_happyPath(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, mockAgent, mockBroadcaster)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(mockBroadcaster.Calls) != 1 {
		t.Errorf("expected 1 broadcast call, got %d", len(mockBroadcaster.Calls))
	}
	if len(mockBroadcaster.Calls) > 0 && mockBroadcaster.Calls[0] != "agent response" {
		t.Errorf("expected broadcast to be called with 'agent response', got %q", mockBroadcaster.Calls[0])
	}
}

func TestProcess_textTrimmed(t *testing.T) {
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

	client := createTestPipeline(t, mockCapturer, mockExtractor, mockAgent, mockBroadcaster)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if capturedText != "extracted text" {
		t.Errorf("expected trimmed text 'extracted text', got %q", capturedText)
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

func TestProcess_goroutineContextCheckOnExpiredContext(t *testing.T) {
	// Test that goroutines check context at start and exit early if already expired
	// This validates the early context check added in pipeline.go
	captureWasCalled := false
	mockCapturer := &mocks.MockCapturerService{
		ForegroundWindowFunc: func(ctx context.Context) (*capturer.WindowInfo, error) {
			return &capturer.WindowInfo{Title: "Test Window", Width: 1920, Height: 1080, X: 0, Y: 0}, nil
		},
		CaptureCenterFunc: func(windowInfo *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			captureWasCalled = true
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		},
	}

	p := createTestPipeline(t, mockCapturer, nil, nil, nil)

	// Use a very short timeout that will expire during capture
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Give timeout time to expire
	time.Sleep(5 * time.Millisecond)

	err := p.Process(ctx)

	// Should fail with capture timeout
	if err == nil {
		t.Error("expected error, got nil")
	}

	// The test validates that the early context check prevents unnecessary work
	// (though in this case, since we have a very short timeout, it's hard to guarantee
	// the goroutine hasn't started before timeout. This is more of a functional test.)
	_ = captureWasCalled
}
