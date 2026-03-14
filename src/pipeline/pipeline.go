package pipeline

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"strings"
	"time"

	"github.com/vdyalex/ccat-assistant/src/adapters/agent"
	"github.com/vdyalex/ccat-assistant/src/adapters/messenger"
	"github.com/vdyalex/ccat-assistant/src/modules/capturer"
	"github.com/vdyalex/ccat-assistant/src/modules/extractor"
	"github.com/vdyalex/ccat-assistant/src/modules/listener"
	config "github.com/vdyalex/ccat-assistant/src/utils"
)

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	settings  *config.Config
	logger    *slog.Logger
	capturer  capturer.Capturer
	extractor *extractor.Extractor
	agent     *agent.Agent
	messenger *messenger.Sender
}

// New creates a fully wired pipeline from settings.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(settings *config.Config, logger *slog.Logger) (*Pipeline, error) {
	extractor, err := extractor.New(settings.TesseractLang)
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		settings:  settings,
		logger:    logger,
		capturer:  capturer.New(),
		extractor: extractor,
		agent:     agent.New(settings.AnthropicAPIKey, settings.ClaudeModel, settings.SystemPrompt),
		messenger: messenger.New(settings.TelegramBotToken, settings.TelegramChatID),
	}, nil
}

// Run starts listening for the hotkey and processes on each trigger.
// It blocks until the context is cancelled.
func (pipeline *Pipeline) Run(ctx context.Context) error {
	defer pipeline.extractor.Close()

	triggers, err := listener.Listen(ctx, pipeline.logger)
	if err != nil {
		return err
	}

	pipeline.logger.Info("Pipeline ready — press right Option key to capture")

	// Worker goroutine processes captures sequentially while main loop stays responsive.
	queue := make(chan struct{})
	go func() {
		for range queue {
			if err := pipeline.process(ctx); err != nil {
				pipeline.logger.Error("Pipeline error", "error", err)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			pipeline.logger.Info("Pipeline shutting down")
			close(queue)
			return ctx.Err()
		case <-triggers:
			pipeline.logger.Debug("Hotkey triggered, queueing capture")
			select {
			case queue <- struct{}{}:
			case <-ctx.Done():
				close(queue)
				return ctx.Err()
			default:
				pipeline.logger.Debug("Capture already queued, skipping")
			}
		}
	}
}

func (pipeline *Pipeline) process(ctx context.Context) error {
	// Step 1: Detect the foreground window (with 5-second timeout)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
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

	// Step 2: Capture the center of the window (with 30-second timeout)
	pipeline.logger.Debug("Capturing center of window")
	ctxCapture, cancelCapture := context.WithTimeout(ctx, 30*time.Second)
	defer cancelCapture()

	imageCh := make(chan *image.RGBA, 1)
	errCh := make(chan error, 1)
	go func() {
		img, err := pipeline.capturer.CaptureCenter(window)
		if err != nil {
			errCh <- err
		} else {
			imageCh <- img
		}
	}()

	var img *image.RGBA
	select {
	case img = <-imageCh:
		// Capture succeeded
	case err := <-errCh:
		return err
	case <-ctxCapture.Done():
		return fmt.Errorf("screenshot capture timeout (30s)")
	}

	// Step 3: Extract text via OCR
	pipeline.logger.Debug("Running OCR on captured image", slog.Int("width", img.Bounds().Dx()), slog.Int("height", img.Bounds().Dy()))
	text, err := pipeline.extractor.Extract(img)
	if err != nil {
		return err
	}

	text = strings.TrimSpace(text)
	if text == "" {
		pipeline.logger.Warn("OCR returned empty text, skipping")
		return nil
	}
	pipeline.logger.Info("OCR extracted text", slog.Int("character_count", len(text)))
	pipeline.logger.Debug("OCR text content", slog.String("text", text))

	// Step 4: Process with Anthropic
	response, err := pipeline.agent.Process(ctx, text)
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

	// Step 5: Send to Telegram
	if err := pipeline.messenger.Send(ctx, response); err != nil {
		return err
	}
	pipeline.logger.Info("Sent to Telegram successfully")

	return nil
}
