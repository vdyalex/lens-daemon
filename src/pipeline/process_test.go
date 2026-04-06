package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"image"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

func TestProcess_noForegroundWindow(t *testing.T) {
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(nil, exceptions.ErrCapturerNoForegroundWindow)

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error (non-fatal skip), got %v", err)
	}
}

func TestProcess_capturerError(t *testing.T) {
	testErr := fmt.Errorf("capturer failed")
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(nil, testErr)

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_captureTimeout(t *testing.T) {
	blockCapture := make(chan struct{})
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	testMocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	testMocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		DoAndReturn(func(window *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			<-blockCapture
			return nil, nil
		})

	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("", nil).
		Times(0)

	settings := createTestConfig()
	settings.TimeoutCapture = 10 * time.Millisecond
	logger := mocks.NopLogger()

	client := createTestPipelineWithSettings(t, settings, logger, testMocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err == nil || !containsError(err, "timeout") {
		t.Errorf("expected capture timeout error, got %v", err)
	}
}

func TestProcess_ocrEmpty(t *testing.T) {
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("", nil)

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error (non-fatal skip for empty OCR), got %v", err)
	}
}

func TestProcess_ocrError(t *testing.T) {
	testErr := fmt.Errorf("OCR failed")
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("", testErr)

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_agentEmpty(t *testing.T) {
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("extracted text", nil)

	mocks.agent.EXPECT().
		Process(gomock.Any(), "extracted text").
		Return(ai.Response{}, nil)

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error (non-fatal skip for empty agent response), got %v", err)
	}
}

func TestProcess_agentError(t *testing.T) {
	testErr := fmt.Errorf("AI failed")
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("extracted text", nil)

	mocks.agent.EXPECT().
		Process(gomock.Any(), "extracted text").
		Return(ai.Response{}, testErr)

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if !errors.Is(err, testErr) {
		t.Errorf("expected %v, got %v", testErr, err)
	}
}

func TestProcess_broadcastError(t *testing.T) {
	// Broadcast errors are now non-fatal; Process() should succeed even if broadcast fails.
	testErr := fmt.Errorf("Broadcast failed")
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("extracted text", nil)

	mocks.agent.EXPECT().
		Process(gomock.Any(), "extracted text").
		Return(ai.Response{
			Deterministic: true,
			Short:         "agent response",
			Detailed: ai.ResponseDetail{
				Answer: "agent response",
				Reason: "because",
			},
		}, nil)

	mocks.teleprompter.EXPECT().
		Display(gomock.Any())

	mocks.broadcaster.BroadcastFunc = func(ctx context.Context, text string) error {
		return testErr
	}

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	// Broadcast failure is non-fatal; expect nil error
	if err != nil {
		t.Errorf("expected no error (broadcast non-fatal), got %v", err)
	}
}

func TestProcess_happyPath(t *testing.T) {
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("extracted text", nil)

	mocks.agent.EXPECT().
		Process(gomock.Any(), "extracted text").
		Return(ai.Response{
			Deterministic: true,
			Short:         "agent response",
			Detailed: ai.ResponseDetail{
				Answer: "agent response",
				Reason: "because",
			},
		}, nil)

	mocks.teleprompter.EXPECT().
		Display(gomock.Any())

	mocks.broadcaster.BroadcastFunc = func(ctx context.Context, text string) error {
		return nil
	}

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestProcess_textTrimmed(t *testing.T) {
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	mocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("  extracted text  ", nil)

	// Capture the text passed to Process
	var capturedText string
	mocks.agent.EXPECT().
		Process(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, text string) (ai.Response, error) {
			capturedText = text
			return ai.Response{
				Deterministic: true,
				Short:         "response",
				Detailed: ai.ResponseDetail{
					Answer: "response",
					Reason: "because",
				},
			}, nil
		})

	mocks.teleprompter.EXPECT().
		Display(gomock.Any())

	mocks.broadcaster.BroadcastFunc = func(ctx context.Context, text string) error {
		return nil
	}

	client := createTestPipeline(t, mocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if capturedText != "extracted text" {
		t.Errorf("expected trimmed text 'extracted text', got %q", capturedText)
	}
}

func TestProcess_deterministicTrue_displaysOnTeleprompter(t *testing.T) {
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	testMocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	testMocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("question text", nil)

	testMocks.agent.EXPECT().
		Process(gomock.Any(), "question text").
		Return(ai.Response{
			Deterministic: true,
			Short:         "B",
			Detailed: ai.ResponseDetail{
				Answer: "B",
				Reason: "reason",
			},
		}, nil)

	testMocks.teleprompter.EXPECT().
		Display("B").
		Times(1)

	client := createTestPipeline(t, testMocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestProcess_deterministicFalse_suppressesTeleprompter(t *testing.T) {
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	windowInfo := &capturer.WindowInfo{
		Title:  "Test Window",
		X:      0,
		Y:      0,
		Width:  100,
		Height: 100,
	}

	testMocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(windowInfo, nil)

	testMocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		Return(image.NewRGBA(image.Rect(0, 0, 100, 100)), nil)

	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("ambiguous question", nil)

	testMocks.agent.EXPECT().
		Process(gomock.Any(), "ambiguous question").
		Return(ai.Response{
			Deterministic: false,
			Short:         "maybe C",
			Detailed: ai.ResponseDetail{
				Answer: "maybe C",
				Reason: "uncertain",
			},
		}, nil)

	// Display must NOT be called when Deterministic is false.
	testMocks.teleprompter.EXPECT().
		Display(gomock.Any()).
		Times(0)

	client := createTestPipeline(t, testMocks)

	ctx := context.Background()
	err := client.Process(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestProcess_goroutineContextCheckOnExpiredContext(t *testing.T) {
	mocks := createTestMocks(t)
	defer mocks.ctrl.Finish()

	mocks.capturer.EXPECT().
		ForegroundWindow(gomock.Any()).
		Return(&capturer.WindowInfo{Title: "Test Window", Width: 1920, Height: 1080, X: 0, Y: 0}, nil)

	// CaptureCenter might or might not be called depending on timing, so allow any calls
	mocks.capturer.EXPECT().
		CaptureCenter(gomock.Any(), gomock.Any()).
		DoAndReturn(func(windowInfo *capturer.WindowInfo, bounds *image.Rectangle) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
		}).
		AnyTimes()

	p := createTestPipeline(t, mocks)

	// Create an already-cancelled context to test expiration handling
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.Process(ctx)

	// Should fail with context deadline exceeded
	if err == nil {
		t.Error("expected error, got nil")
	}
}
