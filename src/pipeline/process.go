package pipeline

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"

	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// fetchWindow detects the foreground window with timeout.
// Returns nil if no foreground window is active (non-fatal).
func (pipeline *Pipeline) fetchWindow(ctx context.Context) (*capturer.WindowInfo, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, pipeline.settings.TimeoutForegroundWindow)
	defer cancel()

	window, err := pipeline.capturer.ForegroundWindow(ctxWithTimeout)
	if errors.Is(err, exceptions.CapturerNoForegroundWindowException) {
		pipeline.logger.Debug("No foreground window, skipping")
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	pipeline.logger.Debug("Foreground window", slog.String("title", window.Title), slog.Int("width", window.Width), slog.Int("height", window.Height), slog.Int("x", window.X), slog.Int("y", window.Y))
	return window, nil
}

// captureScreenshot captures the window center with timeout.
func (pipeline *Pipeline) captureScreenshot(ctx context.Context, window *capturer.WindowInfo) (*image.RGBA, error) {
	pipeline.boundsMutex.RLock()
	bounds := pipeline.captureBounds
	pipeline.boundsMutex.RUnlock()

	if bounds != nil {
		pipeline.logger.Debug("Capturing with custom bounds", slog.Int("minX", bounds.Min.X), slog.Int("minY", bounds.Min.Y), slog.Int("maxX", bounds.Max.X), slog.Int("maxY", bounds.Max.Y))
	} else {
		pipeline.logger.Debug("Capturing center of window (no custom bounds)")
	}

	ctxCapture, cancelCapture := context.WithTimeout(ctx, pipeline.settings.TimeoutCapture)
	defer cancelCapture()

	imageCh := make(chan *image.RGBA, 1)
	errCh := make(chan error, 1)
	go func() {
		pipeline.logger.Debug("Screenshot goroutine started")
		img, err := pipeline.capturer.CaptureCenter(window, bounds)
		if err != nil {
			pipeline.logger.Error("Screenshot capture failed", "error", err)
			errCh <- err
		} else {
			pipeline.logger.Debug("Screenshot captured successfully", "width", img.Bounds().Dx(), "height", img.Bounds().Dy())
			imageCh <- img
		}
	}()

	select {
	case img := <-imageCh:
		pipeline.logger.Debug("Screenshot received from goroutine")
		return img, nil
	case err := <-errCh:
		pipeline.logger.Debug("Screenshot error received from goroutine")
		return nil, err
	case <-ctxCapture.Done():
		pipeline.logger.Error("Screenshot capture timeout", slog.String("timeout", pipeline.settings.TimeoutCapture.String()))
		return nil, fmt.Errorf("%w (%s)", exceptions.PipelineCaptureTimeoutException, pipeline.settings.TimeoutCapture)
	}
}

// extractAndProcessText runs OCR and validates the result.
// Returns nil if OCR produces empty text (non-fatal).
func (pipeline *Pipeline) extractAndProcessText(ctx context.Context, img *image.RGBA) (string, error) {
	pipeline.logger.Debug("Running OCR on captured image", slog.Int("width", img.Bounds().Dx()), slog.Int("height", img.Bounds().Dy()))
	ocrCtx, ocrCancel := context.WithTimeout(ctx, pipeline.settings.TimeoutOCRExtract)
	defer ocrCancel()

	textCh := make(chan string, 1)
	ocrErrCh := make(chan error, 1)
	go func() {
		text, err := pipeline.extractor.Extract(img)
		if err != nil {
			ocrErrCh <- err
		} else {
			textCh <- text
		}
	}()

	var text string
	select {
	case text = <-textCh:
		// OCR succeeded
	case err := <-ocrErrCh:
		return "", err
	case <-ocrCtx.Done():
		return "", fmt.Errorf("%w (%s)", exceptions.PipelineOCRTimeoutException, pipeline.settings.TimeoutOCRExtract)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		pipeline.logger.Warn("OCR returned empty text, skipping")
		return "", nil
	}
	pipeline.logger.Info("OCR extracted text", slog.Int("character_count", len(text)))
	pipeline.logger.Debug("OCR text content", slog.String("text", text))
	return text, nil
}

// processWithAIAndBroadcast sends text to Claude AI and broadcasts response.
// Returns nil if AI produces empty response (non-fatal).
func (pipeline *Pipeline) processWithAIAndBroadcast(ctx context.Context, text string) error {
	agentCtx, agentCancel := context.WithTimeout(ctx, pipeline.settings.TimeoutAIProcess)
	defer agentCancel()
	response, err := pipeline.agent.Process(agentCtx, text)
	if err != nil {
		return err
	}

	response = strings.TrimSpace(response)
	if response == "" {
		pipeline.logger.Warn("Anthropic returned empty response, skipping")
		return nil
	}
	pipeline.logger.Info("Anthropic response received", slog.Int("character_count", len(response)))
	pipeline.logger.Debug("Anthropic response content", slog.String("response", response))

	broadcastCtx, broadcastCancel := context.WithTimeout(ctx, pipeline.settings.TelegramBroadcastTimeout)
	defer broadcastCancel()
	if err := pipeline.messenger.Broadcast(broadcastCtx, response); err != nil {
		return err
	}
	pipeline.logger.Info("Broadcast to Telegram subscribers successfully")
	return nil
}
