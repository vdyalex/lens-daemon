package pipeline

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"sync"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/modules/listener"
	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
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
	// Step 1: Detect the foreground window
	ctxWithTimeout, cancel := context.WithTimeout(ctx, p.settings.TimeoutForegroundWindow)
	defer cancel()

	window, err := p.capturer.ForegroundWindow(ctxWithTimeout)
	if errors.Is(err, exceptions.ErrCapturerNoForegroundWindow) {
		p.logger.Debug("no foreground window, skipping")
		return nil
	}
	if err != nil {
		return err
	}
	p.logger.Debug("foreground window", slog.String("title", window.Title), slog.Int("width", window.Width), slog.Int("height", window.Height), slog.Int("x", window.X), slog.Int("y", window.Y))

	// Step 2: Capture the center of the window
	p.boundsMu.RLock()
	bounds := p.captureBounds
	p.boundsMu.RUnlock()

	if bounds != nil {
		p.logger.Debug("capturing with custom bounds", slog.Int("minX", bounds.Min.X), slog.Int("minY", bounds.Min.Y), slog.Int("maxX", bounds.Max.X), slog.Int("maxY", bounds.Max.Y))
	} else {
		p.logger.Debug("capturing center of window (no custom bounds)")
	}
	ctxCapture, cancelCapture := context.WithTimeout(ctx, p.settings.TimeoutCapture)
	defer cancelCapture()

	imageCh := make(chan *image.RGBA, 1)
	errCh := make(chan error, 1)
	go func() {
		p.logger.Debug("screenshot goroutine started")
		// Check context before starting work
		if ctxCapture.Err() != nil {
			errCh <- ctxCapture.Err()
			return
		}
		img, err := p.capturer.CaptureCenter(window, bounds)
		if err != nil {
			p.logger.Error("screenshot capture failed", "error", err)
			errCh <- err
		} else {
			p.logger.Debug("screenshot captured successfully", "width", img.Bounds().Dx(), "height", img.Bounds().Dy())
			imageCh <- img
		}
	}()

	var img *image.RGBA
	select {
	case img = <-imageCh:
		p.logger.Debug("screenshot received from goroutine")
		// Capture succeeded
	case err := <-errCh:
		p.logger.Debug("screenshot error received from goroutine")
		return err
	case <-ctxCapture.Done():
		p.logger.Error("screenshot capture timeout", slog.String("timeout", p.settings.TimeoutCapture.String()))
		return fmt.Errorf("%w (%s)", exceptions.ErrPipelineCaptureTimeout, p.settings.TimeoutCapture)
	}

	// Step 3: Extract text via OCR
	p.logger.Debug("running ocr on captured image", slog.Int("width", img.Bounds().Dx()), slog.Int("height", img.Bounds().Dy()))
	ocrCtx, ocrCancel := context.WithTimeout(ctx, p.settings.TimeoutOCRExtract)
	defer ocrCancel()

	textCh := make(chan string, 1)
	ocrErrCh := make(chan error, 1)
	go func() {
		// Check context before starting work
		if ocrCtx.Err() != nil {
			ocrErrCh <- ocrCtx.Err()
			return
		}
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
		return err
	case <-ocrCtx.Done():
		return fmt.Errorf("%w (%s)", exceptions.ErrPipelineOCRTimeout, p.settings.TimeoutOCRExtract)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		p.logger.Warn("ocr returned empty text, skipping")
		return nil
	}
	p.logger.Info("ocr extracted text", slog.Int("character_count", len(text)))
	p.logger.Debug("ocr text content", slog.String("text", text))

	// Step 4: Process with Anthropic
	agentCtx, agentCancel := context.WithTimeout(ctx, p.settings.TimeoutAIProcess)
	defer agentCancel()
	response, err := p.agent.Process(agentCtx, text)
	if err != nil {
		return err
	}

	response = strings.TrimSpace(response)
	if response == "" {
		p.logger.Warn("anthropic returned empty response, skipping")
		return nil
	}
	p.logger.Info("anthropic response received", slog.Int("character_count", len(response)))
	p.logger.Debug("anthropic response content", slog.String("response", response))

	// Step 5: Broadcast to Telegram subscribers
	broadcastCtx, broadcastCancel := context.WithTimeout(ctx, p.settings.TelegramBroadcastTimeout)
	defer broadcastCancel()
	if err := p.messenger.Broadcast(broadcastCtx, response); err != nil {
		return err
	}
	p.logger.Info("broadcast to telegram subscribers successfully")

	return nil
}

// Run starts listening for the hotkey and processes on each trigger.
// It blocks until the context is cancelled. Each trigger spawns an async
// goroutine (limited by semaphore to 1 concurrent run for compatibility with
// OCR engines that serialize operations). The function waits for all in-flight
// goroutines before closing the OCR client on exit.
func (p *Pipeline) Run(ctx context.Context) error {
	defer p.extractor.Close()

	hotkeyListener := listener.New()
	triggers, bounds, err := hotkeyListener.Listen(ctx, p.logger, p.settings.EventTapPollInterval, p.settings.HotkeyTriggerKeycode, p.settings.HotkeyBoundsKeycode)
	if err != nil {
		return err
	}

	// Start the Telegram subscriber poller in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.poller.Run(ctx)
	}()

	// Start the bounds tracker goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for rect := range bounds {
			p.boundsMu.Lock()
			p.captureBounds = &rect
			p.boundsMu.Unlock()
			p.logger.Info("capture bounds updated", slog.Int("minX", rect.Min.X), slog.Int("minY", rect.Min.Y), slog.Int("maxX", rect.Max.X), slog.Int("maxY", rect.Max.Y))
		}
	}()

	p.logger.Info("pipeline ready — set bounds and press capture key")

	// Worker goroutine processes captures sequentially while main loop stays responsive.
	queue := make(chan struct{}, p.settings.WorkerQueueCapacity)
	wg.Add(1)
	go func() {
		defer wg.Done()
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
			wg.Wait()
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
