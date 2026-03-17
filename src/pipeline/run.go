// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"errors"
	"log/slog"

	"github.com/vdyalex/lens-daemon/src/modules/listener"
)

// Run starts listening for the hotkey and processes on each trigger.
// It blocks until the context is cancelled. Each trigger spawns an async
// goroutine (limited by queue capacity to control concurrency).
// The function returns when the context is cancelled.
func (p *Pipeline) Run(ctx context.Context) error {
	defer p.extractor.Close()

	hotkeyListener := listener.New()
	triggers, bounds, err := hotkeyListener.Listen(
		ctx,
		p.logger,
		p.settings.EventTapPollInterval,
		p.settings.HotkeyTriggerKeycode,
		p.settings.HotkeyBoundsKeycode,
	)
	if err != nil {
		return err
	}

	// Start the Telegram subscriber poller in background
	go p.poller.Run(ctx)

	// Start the bounds tracker goroutine
	go func() {
		for rect := range bounds {
			p.boundsMu.Lock()
			p.captureBounds = &rect
			p.boundsMu.Unlock()
			p.logger.Info("capture bounds updated",
				slog.Int("minX", rect.Min.X),
				slog.Int("minY", rect.Min.Y),
				slog.Int("maxX", rect.Max.X),
				slog.Int("maxY", rect.Max.Y),
			)
		}
	}()

	p.logger.Info("pipeline ready — set bounds and press capture key")

	// Worker goroutine processes captures sequentially while main loop stays responsive.
	queue := make(chan struct{}, p.settings.WorkerQueueCapacity)
	go func() {
		for range queue {
			// Create a context for this run (allows long OCR+API calls)
			runCtx, cancel := context.WithTimeout(ctx, p.settings.TimeoutPipelineOverall)
			if err := p.Process(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				p.logger.Error("pipeline error", "error", err)
			}
			cancel()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pipeline shutting down")
			close(queue)
			return ctx.Err()
		case <-triggers:
			select {
			case queue <- struct{}{}:
				// Queued
			default:
				p.logger.Warn("capture trigger dropped; a capture is already in progress")
			}
		}
	}
}
