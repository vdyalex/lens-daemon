package pipeline

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/vdyalex/test-assistant/internal/agent"
	"github.com/vdyalex/test-assistant/internal/capture"
	"github.com/vdyalex/test-assistant/internal/config"
	"github.com/vdyalex/test-assistant/internal/diff"
	"github.com/vdyalex/test-assistant/internal/ocr"
	"github.com/vdyalex/test-assistant/internal/telegram"
)

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	cfg      *config.Config
	capturer capture.Capturer
	ring     *diff.RingBuffer
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
		ring:     diff.NewRingBuffer(cfg.MaxHistory),
		ocr:      ocrExtractor,
		ai:       agent.New(cfg.ClaudeAPIKey, cfg.ClaudeModel, cfg.SystemPrompt),
		tg:       telegram.New(cfg.TelegramBotToken, cfg.TelegramChatID),
	}, nil
}

// Run starts the pipeline loop. It blocks until the context is cancelled.
func (p *Pipeline) Run(ctx context.Context) error {
	log.Println("pipeline started, polling every", p.cfg.PollInterval)
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()
	defer p.ocr.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("pipeline shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := p.tick(ctx); err != nil {
				log.Printf("pipeline tick error: %v", err)
			}
		}
	}
}

func (p *Pipeline) tick(ctx context.Context) error {
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

	// Step 3: Compare with previous screenshot
	prev := p.ring.Last()
	if !diff.HasChanged(prev, img, p.cfg.DiffThreshold) {
		log.Println("no visual change detected, skipping")
		return nil
	}

	// Store in ring buffer (no files written to disk)
	p.ring.Push(img)
	log.Printf("visual change detected (buffer: %d/%d)", p.ring.Count(), p.cfg.MaxHistory)

	// Step 4: Extract text via OCR
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

	// Step 5: Process with Claude AI
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

	// Step 6: Send to Telegram
	if err := p.tg.Send(ctx, response); err != nil {
		return err
	}
	log.Println("sent to Telegram successfully")

	return nil
}
