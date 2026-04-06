// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"errors"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// capture executes Phase 1: detects the foreground window, takes a screenshot,
// updates lastCaptureTime and lastWindowTitle, then enqueues the CaptureResult
// for Phase 2 via a non-blocking send on analyseQueue.
//
// capture is safe to call concurrently from multiple goroutines.
// A zero context (already cancelled) returns context.Canceled immediately via the
// inner timeout context. An empty-window result (no foreground window) is non-fatal
// and returns nil. If analyseQueue is full, the result is dropped with a warning log.
//
// ctx: parent context; capture wraps it with TimeoutCapturePhase internally.
// Returns a non-nil error only for fatal capture failures.
func (p *Pipeline) capture(ctx context.Context) error {
	captureCtx, cancel := context.WithTimeout(ctx, p.settings.TimeoutCapturePhase)
	defer cancel()

	window, canvas, err := p.fetchWindow(captureCtx)
	if window == nil || err != nil {
		return err
	}

	img, err := p.captureScreenshot(captureCtx, window, canvas)
	if err != nil {
		return err
	}

	now := time.Now()

	p.lastCaptureMu.Lock()
	p.lastCaptureTime = now
	p.lastWindowTitle = window.Title
	p.lastCaptureMu.Unlock()

	result := CaptureResult{
		Image:       img,
		WindowTitle: window.Title,
		Timestamp:   now,
	}

	select {
	case p.analyseQueue <- result:
		// enqueued for Phase 2
	default:
		p.logger.Warn("analyse queue full; capture result dropped",
			"window", window.Title,
		)
	}

	return nil
}

// analyse executes Phase 2: runs OCR on the captured image, sends text to the AI agent,
// and broadcasts the response to Telegram subscribers.
//
// analyse is safe to call concurrently from multiple goroutines. Each call wraps ctx
// with its own TimeoutAnalysePhase sub-context. An empty OCR result or an empty AI
// response is non-fatal and returns nil.
//
// ctx: parent context; analyse wraps it with TimeoutAnalysePhase internally.
// result: the CaptureResult produced by Phase 1.
// Returns a non-nil error for fatal OCR, AI, or broadcast failures.
func (p *Pipeline) analyse(ctx context.Context, result CaptureResult) error {
	analyseCtx, cancel := context.WithTimeout(ctx, p.settings.TimeoutAnalysePhase)
	defer cancel()

	text, err := p.extractAndProcessText(analyseCtx, result.Image)
	if text == "" || err != nil {
		return err
	}

	return p.processWithAIAndBroadcast(analyseCtx, text)
}

// isFatalError reports whether err should be logged as a pipeline error.
// context.Canceled and ErrCapturerNoForegroundWindow are non-fatal (expected on shutdown
// or empty foreground); all other errors are fatal.
func isFatalError(err error) bool {
	if err == nil {
		return false
	}
	return !errors.Is(err, context.Canceled) &&
		!errors.Is(err, exceptions.ErrCapturerNoForegroundWindow)
}
