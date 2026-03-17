package pipeline

import (
	"context"
	"log/slog"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
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

	store, err := store.New(settings.StorePath, logger)
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
