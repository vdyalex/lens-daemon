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
	"github.com/vdyalex/lens-daemon/src/bridges/core_graphics"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// fetchWindow detects the foreground window with timeout.
// Returns (nil, nil) if no foreground window is active (non-fatal).
// The second return value is the browser canvas rectangle (nil for non-browsers).
func (p *Pipeline) fetchWindow(ctx context.Context) (*capturer.WindowInfo, *image.Rectangle, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, p.settings.TimeoutForegroundWindow)
	defer cancel()

	window, err := p.capturer.ForegroundWindow(ctxWithTimeout)
	if errors.Is(err, exceptions.ErrCapturerNoForegroundWindow) {
		p.logger.Debug("no foreground window, skipping")
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	p.logger.Debug("foreground window",
		slog.String("title", window.Title),
		slog.Int("width", window.Width),
		slog.Int("height", window.Height),
		slog.Int("x", window.X),
		slog.Int("y", window.Y),
	)

	canvas := browser.CanvasBounds(window.Title, window.X, window.Y, window.Width, window.Height)

	windowRect := image.Rect(window.X, window.Y, window.X+window.Width, window.Y+window.Height)

	p.boundsMu.Lock()
	p.canvasBounds = canvas
	p.lastWindowBounds = windowRect
	p.capturedWindowPID = core_graphics.CapturedWindowPID()
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
		appkit.SetOverlayWindowBounds(0, 0, 0, 0)
	} else {
		appkit.SetOverlayCanvasBounds(0, 0, 0, 0)
		appkit.SetOverlayWindowBounds(
			float64(window.X), float64(window.Y),
			float64(window.Width), float64(window.Height),
		)
	}

	return window, canvas, nil
}

// captureScreenshot captures the full window, then crops to the relevant area.
// Manual bounds (from the bounds hotkey) take priority; canvas bounds (browser
// content area excluding toolbar) are used next; full window is the fallback.
//
// Capturing the full window first and cropping in Go avoids CGDisplayCreateImageForRect
// coordinate-offset issues and gives Vision OCR a focused, smaller image.
func (p *Pipeline) captureScreenshot(ctx context.Context, window *capturer.WindowInfo, canvas *image.Rectangle) (*image.RGBA, error) {
	p.boundsMu.RLock()
	bounds := p.captureBounds
	p.boundsMu.RUnlock()

	ctxCapture, cancelCapture := context.WithTimeout(ctx, p.settings.TimeoutCapture)
	defer cancelCapture()

	imageCh := make(chan *image.RGBA, 1)
	errCh := make(chan error, 1)
	go func() {
		p.logger.Debug("screenshot goroutine started")
		// Hide the overlay so it doesn't appear in the display capture.
		appkit.HideOverlayForCapture()
		img, err := p.capturer.CaptureCenter(window, bounds)
		appkit.RestoreOverlayAfterCapture()
		if err != nil {
			p.logger.Error("screenshot capture failed", "error", err)
			errCh <- err
			return
		}
		// When no manual bounds are set, crop to canvas bounds so Vision OCR
		// receives a focused image without toolbar chrome.
		switch {
		case bounds != nil:
			// Manual bounds from hotkey — image already captured to those bounds.
			p.logger.Debug("captured with manual bounds",
				"width", img.Bounds().Dx(),
				"height", img.Bounds().Dy(),
			)

		case canvas != nil:
			// Browser detected — crop full-window capture to content area.
			p.logger.Debug("cropping to canvas",
				"before", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy()),
				"canvas", fmt.Sprintf("(%d,%d)-(%d,%d)", canvas.Min.X, canvas.Min.Y, canvas.Max.X, canvas.Max.Y),
			)
			img = cropToCanvas(img, window, canvas)
			p.logger.Debug("cropped to canvas",
				"after", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy()),
			)

		default:
			// Non-browser window, no manual bounds — full window capture.
			p.logger.Debug("captured full window",
				"width", img.Bounds().Dx(),
				"height", img.Bounds().Dy(),
			)
		}

		imageCh <- img
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

	switch p.OutputMethod() {
	case constants.OutputMethodTeleprompter:
		p.teleprompter.Display(response.Short)
		p.logger.Info("teleprompter updated", slog.String("text", response.Short))

	case constants.OutputMethodTelegram:
		broadcast := fmt.Sprintf("Answer: **%s**\n\nReason:\n%s", response.Detailed.Answer, response.Detailed.Reason)
		broadcastCtx, broadcastCancel := context.WithTimeout(ctx, p.settings.TelegramBroadcastTimeout)
		defer broadcastCancel()
		if err := p.messenger.Broadcast(broadcastCtx, broadcast); err != nil {
			p.logger.Info("broadcast skipped", "error", err)
			return nil
		}
		p.logger.Info("broadcast to telegram subscribers successfully")
	}

	return nil
}

// cropToCanvas extracts the canvas region from a full-window screenshot.
// Canvas bounds are in screen coordinates (Y-down); the image origin is the
// window's top-left corner. The crop rectangle is translated from screen space
// to image space and intersected with the image bounds.
// Returns the original image unchanged if the crop would be empty.
func cropToCanvas(img *image.RGBA, window *capturer.WindowInfo, canvas *image.Rectangle) *image.RGBA {
	// Retina displays may produce images larger than logical window dimensions.
	// Derive scale from the captured image vs the logical window size.
	scaleX := float64(img.Bounds().Dx()) / float64(window.Width)
	scaleY := float64(img.Bounds().Dy()) / float64(window.Height)

	// Translate canvas from screen coordinates to window-relative, then scale.
	relMinX := int(float64(canvas.Min.X-window.X) * scaleX)
	relMinY := int(float64(canvas.Min.Y-window.Y) * scaleY)
	relMaxX := int(float64(canvas.Max.X-window.X) * scaleX)
	relMaxY := int(float64(canvas.Max.Y-window.Y) * scaleY)

	cropRect := image.Rect(relMinX, relMinY, relMaxX, relMaxY).Intersect(img.Bounds())
	if cropRect.Empty() {
		return img
	}

	cropped := img.SubImage(cropRect).(*image.RGBA)
	return cropped
}
