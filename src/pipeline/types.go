// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"image"
	"log/slog"
	"sync"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	settings      *config.Config
	logger        *slog.Logger
	capturer      capturer.Service
	extractor     extractor.Service
	agent         ai.Processor
	messenger     im.Broadcaster
	poller        poller.Service
	boundsMu      sync.RWMutex
	captureBounds *image.Rectangle
}
