// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"image"
	"log/slog"
)

// Run starts the two-phase pipeline event loop and blocks until ctx is cancelled.
//
// On each hotkey trigger, Run spawns a Phase 1 goroutine (capture) tracked by
// captureGroup. A single Phase 2 worker goroutine (analyse) drains the analyseQueue
// channel serially.
//
// Shutdown sequence:
//  1. ctx.Done fires; the main select loop exits.
//  2. Run waits on captureGroup for all in-flight Phase 1 goroutines.
//  3. analyseQueue is closed, signalling the analyse worker to stop.
//  4. Run blocks on workerDone, waiting for the analyse worker to drain the queue.
//  5. extractor.Close() is called after the worker exits.
//  6. Run returns ctx.Err().
//
// ctx: root context; cancellation triggers the shutdown sequence above.
// Returns the listener setup error, or ctx.Err() on normal shutdown.
func (p *Pipeline) Run(ctx context.Context) error {
	hotkeyListener := p.listener
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

	go p.poller.Run(ctx)
	go p.trackBounds(bounds)

	p.logger.Info("pipeline ready — set bounds and press capture key")

	workerDone := make(chan struct{})
	go p.runAnalyseWorker(ctx, workerDone)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pipeline shutting down")
			p.captureGroup.Wait()
			close(p.analyseQueue)
			<-workerDone
			if closeErr := p.extractor.Close(); closeErr != nil {
				p.logger.Error("extractor close error", "error", closeErr)
			}
			return ctx.Err()
		case <-triggers:
			p.captureGroup.Add(1)
			go func() {
				defer p.captureGroup.Done()
				if err := p.capture(ctx); isFatalError(err) {
					p.logger.Error("capture error", "error", err)
				}
			}()
		}
	}
}

// trackBounds receives updated capture rectangles from the hotkey listener
// and stores them under boundsMu. Exits when the bounds channel is closed.
func (p *Pipeline) trackBounds(bounds <-chan image.Rectangle) {
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
}

// runAnalyseWorker is the Phase 2 serial worker. It drains analyseQueue until
// the channel is closed, then closes workerDone to signal Run().
func (p *Pipeline) runAnalyseWorker(ctx context.Context, workerDone chan<- struct{}) {
	defer close(workerDone)
	for result := range p.analyseQueue {
		if err := p.analyse(ctx, result); isFatalError(err) {
			p.logger.Error("analyse error", "error", err,
				"window", result.WindowTitle,
			)
		}
	}
}
