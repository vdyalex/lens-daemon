package pipeline

import (
	"context"
	"errors"
	"log/slog"

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
) *Pipeline {
	return &Pipeline{
		settings:  settings,
		logger:    logger,
		capturer:  capturer,
		extractor: extractor,
		agent:     agent,
		messenger: broadcaster,
		poller:    pollerService,
	}
}

// New creates a fully wired pipeline from settings.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(settings *config.Config, logger *slog.Logger) (*Pipeline, error) {
	ocr := extractor.New(settings.VisionLanguage, settings.VisionAccuracy)

	store, err := store.NewStore(settings.SubscriberStorePath, logger)
	if err != nil {
		return nil, err
	}

	pipeline := NewWithDependencies(
		settings,
		logger,
		capturer.New(),
		ocr,
		ai.New(settings.AnthropicAPIKey, settings.AnthropicModel, settings.AnthropicSystemPrompt, settings.AnthropicMaxResponseTokens),
		im.New(settings.TelegramBotToken, store, logger, settings.TelegramMessageChunkSize, settings.TelegramMaxRetries, settings.TelegramHTTPClientTimeout),
		poller.New(settings.TelegramBotToken, store, logger, settings.TelegramLongPollTimeout, settings.TelegramPollerTimeout, settings.TelegramHTTPClientTimeout),
	)
	return pipeline, nil
}

// Process runs a single capture-to-broadcast cycle: detect foreground window, capture screenshot,
// extract text via OCR, send to Claude AI, and broadcast response to Telegram subscribers.
// Returns nil on success; logs warnings for non-fatal conditions (empty OCR/response).
func (pipeline *Pipeline) Process(ctx context.Context) error {
	window, err := pipeline.fetchWindow(ctx)
	if window == nil || err != nil {
		return err
	}

	img, err := pipeline.captureScreenshot(ctx, window)
	if err != nil {
		return err
	}

	text, err := pipeline.extractAndProcessText(ctx, img)
	if text == "" || err != nil {
		return err
	}

	return pipeline.processWithAIAndBroadcast(ctx, text)
}

// Run starts listening for the hotkey and processes on each trigger.
// It blocks until the context is cancelled. Each trigger spawns an async
// goroutine (limited by semaphore to 1 concurrent run for compatibility with
// OCR engines that serialize operations). The function waits for all in-flight
// goroutines before closing the OCR client on exit.
func (pipeline *Pipeline) Run(ctx context.Context) error {
	defer pipeline.extractor.Close()

	hotkeyListener := listener.New()
	triggers, bounds, err := hotkeyListener.Listen(ctx, pipeline.logger, pipeline.settings.EventTapPollInterval, pipeline.settings.HotkeyTriggerKeycode, pipeline.settings.HotkeyBoundsKeycode)
	if err != nil {
		return err
	}

	// Start the Telegram subscriber poller in background
	go pipeline.poller.Run(ctx)

	// Start the bounds tracker goroutine
	go func() {
		for rect := range bounds {
			pipeline.boundsMutex.Lock()
			pipeline.captureBounds = &rect
			pipeline.boundsMutex.Unlock()
			pipeline.logger.Info("Capture bounds updated", slog.Int("minX", rect.Min.X), slog.Int("minY", rect.Min.Y), slog.Int("maxX", rect.Max.X), slog.Int("maxY", rect.Max.Y))
		}
	}()

	pipeline.logger.Info("Pipeline ready — press right Shift key to capture (right Option to set bounds)")

	// Worker goroutine processes captures sequentially while main loop stays responsive.
	queue := make(chan struct{}, pipeline.settings.WorkerQueueCapacity)
	go func() {
		for range queue {
			// Create a context for this run (allows long OCR+API calls)
			runCtx, cancel := context.WithTimeout(ctx, pipeline.settings.TimeoutPipelineOverall)
			if err := pipeline.Process(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				pipeline.logger.Error("Pipeline error", "error", err)
			}
			cancel()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			pipeline.logger.Info("Pipeline shutting down")
			close(queue)
			return ctx.Err()
		case <-triggers:
			select {
			case queue <- struct{}{}:
				// Queued
			default:
				pipeline.logger.Warn("Capture trigger dropped; a capture is already in progress")
			}
		}
	}
}
