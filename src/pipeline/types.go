// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"image"
	"log/slog"
	"sync"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/modules/listener"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

// CaptureResult carries the output of Phase 1 (capture) to Phase 2 (analyse).
// All fields are immutable after construction; no synchronisation required for reads.
type CaptureResult struct {
	// Image is the RGBA screenshot taken at trigger time.
	Image *image.RGBA
	// WindowTitle is the foreground window title at capture time.
	WindowTitle string
	// Timestamp is the wall-clock time of the screenshot.
	Timestamp time.Time
}

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	settings        *config.Config
	logger          *slog.Logger
	capturer        capturer.Service
	extractor       extractor.Service
	agent           ai.Processor
	messenger       im.Broadcaster
	poller          poller.Service
	listener        listener.Service
	boundsMu        sync.RWMutex
	captureBounds   *image.Rectangle
	startTime       time.Time
	lastCaptureMu   sync.RWMutex
	lastCaptureTime time.Time
	lastWindowTitle string
	captureGroup    sync.WaitGroup
	analyseQueue    chan CaptureResult
}
