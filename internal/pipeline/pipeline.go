package pipeline

import (
	"context"
	"log"
	"strings"

	"github.com/vdyalex/ccat-assistant/internal/agent"
	"github.com/vdyalex/ccat-assistant/internal/capture"
	"github.com/vdyalex/ccat-assistant/internal/config"
	"github.com/vdyalex/ccat-assistant/internal/hotkey"
	"github.com/vdyalex/ccat-assistant/internal/ocr"
	"github.com/vdyalex/ccat-assistant/internal/telegram"
)

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	cfg      *config.Config
	capturer capture.Capturer
	ocr      *ocr.Extractor
	ai       *agent.Agent
	tg       *telegram.Sender
}

// New creates a fully wired pipeline from configuration.
func New(cfg *config.Config) (*Pipeline, error) {
	ocrExtractor, err := ocr.New(cfg.TesseractLang)
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		cfg:      cfg,
		capturer: capture.New(),
		ocr:      ocrExtractor,
		ai:       agent.New(cfg.ClaudeAPIKey, cfg.ClaudeModel, cfg.SystemPrompt),
		tg:       telegram.New(cfg.TelegramBotToken, cfg.TelegramChatID),
	}, nil
}

// Run starts listening for the hotkey and processes on each trigger.
// It blocks until the context is cancelled.
func (p *Pipeline) Run(ctx context.Context) error {
	defer p.ocr.Close()

	triggers, err := hotkey.Listen(ctx)
	if err != nil {
		return err
	}

	log.Println("pipeline ready — press right Option key to capture")

	for {
		select {
		case <-ctx.Done():
			log.Println("pipeline shutting down")
			return ctx.Err()
		case <-triggers:
			log.Println("hotkey triggered, capturing screen")
			if err := p.process(ctx); err != nil {
				log.Printf("pipeline error: %v", err)
			}
		}
	}
}

func (p *Pipeline) process(ctx context.Context) error {
	// Step 1: Detect the foreground window
	win, err := p.capturer.ForegroundWindow()
	if err != nil {
		return err
	}
	log.Printf("foreground window: %q (%dx%d at %d,%d)", win.Title, win.Width, win.Height, win.X, win.Y)

	// Step 2: Capture the center of the window
	img, err := p.capturer.CaptureCenter(win)
	if err != nil {
		return err
	}

	// Step 3: Extract text via OCR
	text, err := p.ocr.Extract(img)
	if err != nil {
		return err
	}

	text = strings.TrimSpace(text)
	if text == "" {
		log.Println("OCR returned empty text, skipping")
		return nil
	}
	log.Printf("OCR extracted %d characters", len(text))

	// Step 4: Process with Claude AI
	response, err := p.ai.Process(ctx, text)
	if err != nil {
		return err
	}

	response = strings.TrimSpace(response)
	if response == "" {
		log.Println("Claude returned empty response, skipping")
		return nil
	}
	log.Printf("Claude response: %d characters", len(response))

	// Step 5: Send to Telegram
	if err := p.tg.Send(ctx, response); err != nil {
		return err
	}
	log.Println("sent to Telegram successfully")

	return nil
}
