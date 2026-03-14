package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/vdyalex/test-assistant/src/adapters/agent"
	"github.com/vdyalex/test-assistant/src/adapters/messenger"
	"github.com/vdyalex/test-assistant/src/modules/capturer"
	"github.com/vdyalex/test-assistant/src/modules/extractor"
	"github.com/vdyalex/test-assistant/src/modules/listener"
	config "github.com/vdyalex/test-assistant/src/utils"
)

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	settings  *config.Config
	logger    *slog.Logger
	capturer  capturer.Capturer
	extractor *extractor.Extractor
	anthropic *agent.Agent
	telegram  *messenger.Sender
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
		anthropic: agent.New(settings.AnthropicAPIKey, settings.ClaudeModel, settings.SystemPrompt),
		telegram:  messenger.New(settings.TelegramBotToken, settings.TelegramChatID),
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

	for {
		select {
		case <-ctx.Done():
			pipeline.logger.Info("Pipeline shutting down")
			return ctx.Err()
		case <-triggers:
			pipeline.logger.Debug("Hotkey triggered, capturing screen")
			if err := pipeline.process(ctx); err != nil {
				pipeline.logger.Error("Pipeline error", "error", err)
			}
		}
	}
}

func (pipeline *Pipeline) process(ctx context.Context) error {
	// Step 1: Detect the foreground window
	window, err := pipeline.capturer.ForegroundWindow()
	if errors.Is(err, capturer.ErrNoForegroundWindow) {
		pipeline.logger.Debug("No foreground window, skipping")
		return nil
	}
	if err != nil {
		return err
	}
	pipeline.logger.Debug("Foreground window", slog.String("title", window.Title), slog.Int("width", window.Width), slog.Int("height", window.Height), slog.Int("x", window.X), slog.Int("y", window.Y))

	// Step 2: Capture the center of the window
	pipeline.logger.Debug("Capturing center of window")
	image, err := pipeline.capturer.CaptureCenter(window)
	if err != nil {
		return err
	}

	// Step 3: Extract text via OCR
	pipeline.logger.Debug("Running OCR on captured image", slog.Int("width", image.Bounds().Dx()), slog.Int("height", image.Bounds().Dy()))
	text, err := pipeline.extractor.Extract(image)
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
	response, err := pipeline.anthropic.Process(ctx, text)
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
	if err := pipeline.telegram.Send(ctx, response); err != nil {
		return err
	}
	pipeline.logger.Info("Sent to Telegram successfully")

	return nil
}
