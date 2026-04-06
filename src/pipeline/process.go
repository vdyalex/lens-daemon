// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/bridges/browser"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// fetchWindow detects the foreground window with timeout.
// Returns nil if no foreground window is active (non-fatal).
func (p *Pipeline) fetchWindow(ctx context.Context) (*capturer.WindowInfo, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, p.settings.TimeoutForegroundWindow)
	defer cancel()

	window, err := p.capturer.ForegroundWindow(ctxWithTimeout)
	if errors.Is(err, exceptions.ErrCapturerNoForegroundWindow) {
		p.logger.Debug("no foreground window, skipping")
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.logger.Debug("foreground window",
		slog.String("title", window.Title),
		slog.Int("width", window.Width),
		slog.Int("height", window.Height),
		slog.Int("x", window.X),
		slog.Int("y", window.Y),
	)

	canvas := browser.CanvasBounds(window.Title, window.X, window.Y, window.Width, window.Height)
	p.boundsMu.Lock()
	p.canvasBounds = canvas
	p.boundsMu.Unlock()
	if canvas != nil {
		p.logger.Debug("canvas bounds detected",
			slog.Int("minX", canvas.Min.X),
			slog.Int("minY", canvas.Min.Y),
			slog.Int("maxX", canvas.Max.X),
			slog.Int("maxY", canvas.Max.Y),
		)
		appkit.SetOverlayCanvasBounds(
			float64(canvas.Min.X), float64(canvas.Min.Y),
			float64(canvas.Dx()), float64(canvas.Dy()),
		)
	} else {
		appkit.SetOverlayCanvasBounds(0, 0, 0, 0)
	}

	return window, nil
}

// captureScreenshot captures the window center with timeout.
func (p *Pipeline) captureScreenshot(ctx context.Context, window *capturer.WindowInfo) (*image.RGBA, error) {
	p.boundsMu.RLock()
	bounds := p.captureBounds
	if bounds == nil {
		bounds = p.canvasBounds
	}
	p.boundsMu.RUnlock()

	if bounds != nil {
		p.logger.Debug("capturing with custom bounds",
			slog.Int("minX", bounds.Min.X),
			slog.Int("minY", bounds.Min.Y),
			slog.Int("maxX", bounds.Max.X),
			slog.Int("maxY", bounds.Max.Y),
		)
	} else {
		p.logger.Debug("capturing center of window (no custom bounds)")
	}

	ctxCapture, cancelCapture := context.WithTimeout(ctx, p.settings.TimeoutCapture)
	defer cancelCapture()

	imageCh := make(chan *image.RGBA, 1)
	errCh := make(chan error, 1)
	go func() {
		p.logger.Debug("screenshot goroutine started")
		img, err := p.capturer.CaptureCenter(window, bounds)
		if err != nil {
			p.logger.Error("screenshot capture failed", "error", err)
			errCh <- err
		} else {
			p.logger.Debug("screenshot captured successfully",
				"width", img.Bounds().Dx(),
				"height", img.Bounds().Dy(),
			)
			imageCh <- img
		}
	}()

	select {
	case img := <-imageCh:
		p.logger.Debug("screenshot received from goroutine")
		return img, nil
	case err := <-errCh:
		p.logger.Debug("screenshot error received from goroutine")
		return nil, err
	case <-ctxCapture.Done():
		p.logger.Error("screenshot capture timeout",
			slog.String("timeout", p.settings.TimeoutCapture.String()),
		)
		return nil, fmt.Errorf("%w (%s)", exceptions.ErrPipelineCaptureTimeout, p.settings.TimeoutCapture)
	}
}

// extractAndProcessText runs OCR and validates the result.
// Returns nil if OCR produces empty text (non-fatal).
func (p *Pipeline) extractAndProcessText(ctx context.Context, img *image.RGBA) (string, error) {
	p.logger.Debug("running ocr on captured image",
		slog.Int("width", img.Bounds().Dx()),
		slog.Int("height", img.Bounds().Dy()),
	)
	ocrCtx, ocrCancel := context.WithTimeout(ctx, p.settings.TimeoutOCRExtract)
	defer ocrCancel()

	textCh := make(chan string, 1)
	ocrErrCh := make(chan error, 1)
	go func() {
		text, err := p.extractor.Extract(img)
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
		return "", fmt.Errorf("%w (%s)", exceptions.ErrPipelineOCRTimeout, p.settings.TimeoutOCRExtract)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		p.logger.Warn("ocr returned empty text, skipping")
		return "", nil
	}
	p.logger.Info("ocr extracted text", slog.Int("character_count", len(text)))
	p.logger.Debug("ocr text content", slog.String("text", text))
	return text, nil
}

// processWithAIAndBroadcast sends text to Claude AI, then routes the response:
//   - short version to the teleprompter overlay
//   - detailed version to Telegram subscribers (or noop if disabled)
//
// Returns nil if AI produces empty response (non-fatal).
func (p *Pipeline) processWithAIAndBroadcast(ctx context.Context, text string) error {
	agentCtx, agentCancel := context.WithTimeout(ctx, p.settings.TimeoutAIProcess)
	defer agentCancel()
	response, err := p.agent.Process(agentCtx, text)
	if err != nil {
		return err
	}

	if response.Short == "" && response.Detailed.Answer == "" {
		p.logger.Warn("anthropic returned empty response, skipping")
		return nil
	}
	p.logger.Info("anthropic response received",
		slog.Int("short_character_count", len(response.Short)),
		slog.Int("detailed_character_count", len(response.Detailed.Reason)),
	)

	p.teleprompter.Display(response.Short)
	p.logger.Info("teleprompter updated", slog.String("text", response.Short))

	broadcast := fmt.Sprintf("Answer: **%s**\n\nReason:\n%s", response.Detailed.Answer, response.Detailed.Reason)
	broadcastCtx, broadcastCancel := context.WithTimeout(ctx, p.settings.TelegramBroadcastTimeout)
	defer broadcastCancel()
	if err := p.messenger.Broadcast(broadcastCtx, broadcast); err != nil {
		p.logger.Info("broadcast skipped", "error", err)
		return nil
	}
	p.logger.Info("broadcast to telegram subscribers successfully")
	return nil
}
