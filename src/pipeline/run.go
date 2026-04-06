// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"sync"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
)

// Run starts the two-phase pipeline event loop and blocks until ctx is cancelled.
//
// On each hotkey trigger, Run spawns a Phase 1 goroutine (capture) tracked by
// captureGroup. A Phase 2 worker goroutine (analyse) reads from analyseQueue and
// spawns one goroutine per result, processing concurrently.
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
	channels, err := hotkeyListener.Listen(
		ctx,
		p.logger,
		p.settings.EventTapPollInterval,
		p.settings.HotkeyTriggerKeycode,
		p.settings.HotkeyBoundsKeycode,
		p.settings.HotkeyToggleKeycode,
	)
	if err != nil {
		return err
	}

	go p.poller.Run(ctx)
	go p.trackBounds(channels.Bounds)
	go p.trackVisibility(channels.Toggles)
	go p.trackPosition(channels.Positions)
	go p.trackOpacity(channels.Opacities)
	go p.trackWindowChanges(ctx)

	// Sync the C-side grid position with the configured initial values.
	appkit.CommitMoveToGridSpot(p.settings.GridInitialCol, p.settings.GridInitialRow)

	p.logger.Info("pipeline ready — set bounds and press capture key")

	workerDone := make(chan struct{})
	go p.runAnalyseWorker(ctx, workerDone)

	var captureGroup sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pipeline shutting down")
			captureGroup.Wait()
			close(p.analyseQueue)
			<-workerDone
			if closeErr := p.extractor.Close(); closeErr != nil {
				p.logger.Error("extractor close error", "error", closeErr)
			}
			return ctx.Err()
		case <-channels.Triggers:
			p.teleprompter.Display("")
			captureGroup.Add(1)
			go func() {
				defer captureGroup.Done()
				if err := p.capture(ctx); isFatalError(err) {
					p.logger.Error("capture error", "error", err)
				}
			}()
		}
	}
}

// runAnalyseWorker is the Phase 2 concurrent worker. It drains analyseQueue until
// the channel is closed, spawning one goroutine per result. All goroutines share ctx;
// cancellation propagates to every in-flight analyse call.
//
// wg.Wait() blocks until every goroutine completes, so workerDone is not closed until
// all in-flight work is done. This preserves the shutdown contract with Run().
//
// ctx: parent context; passed through to each analyse goroutine.
// workerDone: closed when all in-flight goroutines have completed.
func (p *Pipeline) runAnalyseWorker(ctx context.Context, workerDone chan<- struct{}) {
	defer close(workerDone)
	var wg sync.WaitGroup
	for result := range p.analyseQueue {
		wg.Add(1)
		go func(result CaptureResult) {
			defer wg.Done()
			if err := p.analyse(ctx, result); isFatalError(err) {
				p.logger.Error("analyse error", "error", err,
					"window", result.WindowTitle,
				)
			}
		}(result)
	}
	wg.Wait()
}
