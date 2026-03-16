package pipeline

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"sync"

	"github.com/vdyalex/lens-daemon/src/adapters/agent"
	"github.com/vdyalex/lens-daemon/src/adapters/messenger"
	"github.com/vdyalex/lens-daemon/src/adapters/messenger/poller"
	"github.com/vdyalex/lens-daemon/src/adapters/messenger/subscriber"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/modules/listener"
	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	settings      *config.Config
	logger        *slog.Logger
	capturer      capturer.Capturer
	extractor     extractor.Extractor
	agent         *agent.Agent
	messenger     *messenger.Sender
	poller        *poller.Poller
	boundsMu      sync.RWMutex
	captureBounds *image.Rectangle
}

// New creates a fully wired pipeline from settings.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(settings *config.Config, logger *slog.Logger) (*Pipeline, error) {
	ocr, err := extractor.New(settings.VisionLanguage)
	if err != nil {
		return nil, err
	}

	store, err := subscriber.NewStore(settings.SubscriberStorePath, settings.TelegramChatID)
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		settings:  settings,
		logger:    logger,
		capturer:  capturer.New(),
		extractor: ocr,
		agent:     agent.New(settings.AnthropicAPIKey, settings.ClaudeModel, settings.SystemPrompt),
		messenger: messenger.New(settings.TelegramBotToken, store, logger),
		poller:    poller.New(settings.TelegramBotToken, store, logger),
	}, nil
}

// Run starts listening for the hotkey and processes on each trigger.
// It blocks until the context is cancelled. Each trigger spawns an async
// goroutine (limited by semaphore to 1 concurrent run for compatibility with
// OCR engines that serialize operations). The function waits for all in-flight
// goroutines before closing the OCR client on exit.
func (pipeline *Pipeline) Run(ctx context.Context) error {
	defer pipeline.extractor.Close()

	triggers, bounds, err := listener.Listen(ctx, pipeline.logger)
	if err != nil {
		return err
	}

	// Start the Telegram subscriber poller in background
	go pipeline.poller.Run(ctx)

	// Start the bounds tracker goroutine
	go func() {
		for rect := range bounds {
			pipeline.boundsMu.Lock()
			pipeline.captureBounds = &rect
			pipeline.boundsMu.Unlock()
			pipeline.logger.Info("Capture bounds updated", slog.Int("minX", rect.Min.X), slog.Int("minY", rect.Min.Y), slog.Int("maxX", rect.Max.X), slog.Int("maxY", rect.Max.Y))
		}
	}()

	pipeline.logger.Info("Pipeline ready — press right Shift key to capture (right Option to set bounds)")

	// Worker goroutine processes captures sequentially while main loop stays responsive.
	queue := make(chan struct{}, constants.WorkerQueueCapacity)
	go func() {
		for range queue {
			// Create a context for this run (allows long OCR+API calls)
			runCtx, cancel := context.WithTimeout(ctx, constants.TimeoutPipelineOverall)
			if err := pipeline.process(runCtx); err != nil && !errors.Is(err, context.Canceled) {
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
				pipeline.logger.Debug("Capture queue full, skipping trigger")
			}
		}
	}
}

func (pipeline *Pipeline) process(ctx context.Context) error {
	// Step 1: Detect the foreground window
	ctxWithTimeout, cancel := context.WithTimeout(ctx, constants.TimeoutForegroundWindow)
	defer cancel()

	window, err := pipeline.capturer.ForegroundWindow(ctxWithTimeout)
	if errors.Is(err, capturer.ErrNoForegroundWindow) {
		pipeline.logger.Debug("No foreground window, skipping")
		return nil
	}
	if err != nil {
		return err
	}
	pipeline.logger.Debug("Foreground window", slog.String("title", window.Title), slog.Int("width", window.Width), slog.Int("height", window.Height), slog.Int("x", window.X), slog.Int("y", window.Y))

	// Step 2: Capture the center of the window
	pipeline.boundsMu.RLock()
	bounds := pipeline.captureBounds
	pipeline.boundsMu.RUnlock()

	if bounds != nil {
		pipeline.logger.Debug("Capturing with custom bounds", slog.Int("minX", bounds.Min.X), slog.Int("minY", bounds.Min.Y), slog.Int("maxX", bounds.Max.X), slog.Int("maxY", bounds.Max.Y))
	} else {
		pipeline.logger.Debug("Capturing center of window (no custom bounds)")
	}
	ctxCapture, cancelCapture := context.WithTimeout(ctx, constants.TimeoutCapture)
	defer cancelCapture()

	imageCh := make(chan *image.RGBA, 1)
	errCh := make(chan error, 1)
	go func() {
		pipeline.logger.Debug("Screenshot goroutine started")
		img, err := pipeline.capturer.CaptureCenter(window, bounds)
		if err != nil {
			pipeline.logger.Error("Screenshot capture failed", "error", err)
			errCh <- err
		} else {
			pipeline.logger.Debug("Screenshot captured successfully", "width", img.Bounds().Dx(), "height", img.Bounds().Dy())
			imageCh <- img
		}
	}()

	var img *image.RGBA
	select {
	case img = <-imageCh:
		pipeline.logger.Debug("Screenshot received from goroutine")
		// Capture succeeded
	case err := <-errCh:
		pipeline.logger.Debug("Screenshot error received from goroutine")
		return err
	case <-ctxCapture.Done():
		pipeline.logger.Error("Screenshot capture timeout", slog.String("timeout", constants.TimeoutCapture.String()))
		return fmt.Errorf("screenshot capture timeout (%s)", constants.TimeoutCapture)
	}

	// Step 3: Extract text via OCR
	pipeline.logger.Debug("Running OCR on captured image", slog.Int("width", img.Bounds().Dx()), slog.Int("height", img.Bounds().Dy()))
	ocrCtx, ocrCancel := context.WithTimeout(ctx, constants.TimeoutOCRExtract)
	defer ocrCancel()

	textCh := make(chan string, 1)
	ocrErrCh := make(chan error, 1)
	go func() {
		text, err := pipeline.extractor.Extract(img)
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
		return fmt.Errorf("OCR timeout (%s)", constants.TimeoutOCRExtract)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		pipeline.logger.Warn("OCR returned empty text, skipping")
		return nil
	}
	pipeline.logger.Info("OCR extracted text", slog.Int("character_count", len(text)))
	pipeline.logger.Debug("OCR text content", slog.String("text", text))

	// Step 4: Process with Anthropic
	agentCtx, agentCancel := context.WithTimeout(ctx, constants.TimeoutAgentProcess)
	defer agentCancel()
	response, err := pipeline.agent.Process(agentCtx, text)
	if err != nil {
		return err
	}

	response = strings.TrimSpace(response)
	if response == "" {
		pipeline.logger.Warn("Anthropic returned empty response, skipping")
		return nil
	}
	pipeline.logger.Info("Anthropic response received", slog.Int("character_count", len(response)))
	pipeline.logger.Debug("Anthropic response content", slog.String("response", response))

	// Step 5: Broadcast to Telegram subscribers
	telegramCtx, telegramCancel := context.WithTimeout(ctx, constants.TimeoutTelegramBroadcast)
	defer telegramCancel()
	if err := pipeline.messenger.Broadcast(telegramCtx, response); err != nil {
		return err
	}
	pipeline.logger.Info("Broadcast to Telegram subscribers successfully")

	return nil
}
