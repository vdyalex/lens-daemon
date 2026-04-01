// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/modules/listener"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

// NewWithDependencies creates a pipeline with injectable dependencies.
// This is primarily used for testing with mock implementations.
func NewWithDependencies(
	settings *config.Config,
	logger *slog.Logger,
	capturer capturer.Service,
	extractor extractor.Service,
	agent ai.Processor,
	broadcaster im.Broadcaster,
	pollerService poller.Service,
	listenerService listener.Service,
) *Pipeline {
	return &Pipeline{
		settings:     settings,
		logger:       logger,
		capturer:     capturer,
		extractor:    extractor,
		agent:        agent,
		messenger:    broadcaster,
		poller:       pollerService,
		listener:     listenerService,
		startTime:    time.Now(),
		analyseQueue: make(chan CaptureResult, settings.AnalyseQueueCapacity),
	}
}

// New creates a fully wired pipeline from settings.
// Returns the pipeline, the subscriber store (for status reporting), and any error.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(settings *config.Config, logger *slog.Logger) (*Pipeline, im.Store, error) {
	ocr := extractor.New(settings.VisionLanguage, settings.VisionAccuracy)

	subscriberStore, err := store.New(settings.StorePath, logger)
	if err != nil {
		return nil, nil, err
	}

	pipeline := NewWithDependencies(
		settings,
		logger,
		capturer.New(),
		ocr,
		ai.New(
			settings.AnthropicAPIKey,
			settings.AnthropicModel,
			settings.AnthropicSystemPrompt,
			settings.AnthropicMaxResponseTokens,
			logger,
		),
		im.New(
			settings.TelegramBotToken,
			subscriberStore,
			logger,
			settings.TelegramMessageChunkSize,
			settings.TelegramMaxRetries,
			settings.TelegramHTTPClientTimeout,
		),
		poller.New(
			settings.TelegramBotToken,
			subscriberStore,
			logger,
			settings.TelegramLongPollTimeout,
			settings.TelegramPollerTimeout,
			settings.TelegramHTTPClientTimeout,
		),
		listener.New(),
	)
	return pipeline, subscriberStore, nil
}

// Process runs a single capture-to-broadcast cycle: detect foreground window, capture screenshot,
// extract text via OCR, send to Claude AI, and broadcast response to Telegram subscribers.
// Deprecated: not used by Run(), which uses the two-phase pipeline (capture, analyse).
// Retained for tests and tooling only.
// Returns nil on success; logs warnings for non-fatal conditions (empty OCR/response).
func (p *Pipeline) Process(ctx context.Context) error {
	window, err := p.fetchWindow(ctx)
	if window == nil || err != nil {
		return err
	}

	img, err := p.captureScreenshot(ctx, window)
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
