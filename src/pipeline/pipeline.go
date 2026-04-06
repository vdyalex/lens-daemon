// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/factory"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/modules/listener"
	"github.com/vdyalex/lens-daemon/src/modules/teleprompter"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

// New creates a fully wired pipeline from settings.
// Returns the pipeline, the subscriber store (nil when Telegram is disabled), and any error.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(settings *config.Config, logger *slog.Logger) (*Pipeline, im.Store, error) {
	subscriberStore, err := factory.BuildStore(settings, logger)
	if err != nil {
		return nil, nil, err
	}

	pipeline := NewBuilder(settings, logger).
		WithCapturer(capturer.New()).
		WithExtractor(extractor.New(settings.VisionLanguage, settings.VisionAccuracy)).
		WithAgent(ai.New(
			settings.AnthropicAPIKey,
			settings.AnthropicModel,
			settings.AnthropicSystemPrompt,
			settings.AnthropicMaxResponseTokens,
			anthropic.CacheControlEphemeralTTL(settings.AnthropicCacheTTL),
			logger,
		)).
		WithBroadcaster(factory.BroadcasterFactory{Settings: settings, Store: subscriberStore, Logger: logger}.Build()).
		WithPoller(factory.PollerFactory{Settings: settings, Store: subscriberStore, Logger: logger}.Build()).
		WithListener(listener.New()).
		WithTeleprompter(teleprompter.New(appkit.OverlayConfig{
			FontFamily:    settings.TeleprompterFontFamily,
			FontWeight:    settings.TeleprompterFontWeight,
			FontSize:      settings.TeleprompterFontSize,
			Opacity:       settings.TeleprompterOpacity,
			Alignment:     settings.TeleprompterAlignment,
			AdaptiveColor: settings.TeleprompterAdaptiveColor,
			FadeDuration:  settings.TeleprompterFadeDuration,
		}, settings.TeleprompterVisible)).
		Build()

	return pipeline, subscriberStore, nil
}

// Process runs a single capture-to-broadcast cycle: detect foreground window, capture screenshot,
// extract text via OCR, send to Claude AI, and broadcast response to Telegram subscribers.
// Deprecated: not used by Run(), which uses the two-phase pipeline (capture, analyse).
// Retained for tests and tooling only.
// Returns nil on success; logs warnings for non-fatal conditions (empty OCR/response).
func (p *Pipeline) Process(ctx context.Context) error {
	window, canvas, err := p.fetchWindow(ctx)
	if window == nil || err != nil {
		return err
	}

	img, err := p.captureScreenshot(ctx, window, canvas)
	if err != nil {
		return err
	}

	text, err := p.extractAndProcessText(ctx, img)
	if text == "" || err != nil {
		return err
	}

	return p.processWithAIAndBroadcast(ctx, text)
}

// Status returns a snapshot of the pipeline's current runtime state.
// lastCaptureTime reflects the Phase 1 capture time, not the broadcast completion time.
// Safe to call concurrently.
func (p *Pipeline) Status() (int, float64, time.Time, string) {
	p.lastCaptureMu.RLock()
	defer p.lastCaptureMu.RUnlock()

	uptime := time.Since(p.startTime).Seconds()
	return os.Getpid(), uptime, p.lastCaptureTime, p.lastWindowTitle
}
