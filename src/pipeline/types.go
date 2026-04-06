// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"image"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/modules/capturer"
	"github.com/vdyalex/lens-daemon/src/modules/extractor"
	"github.com/vdyalex/lens-daemon/src/modules/listener"
	"github.com/vdyalex/lens-daemon/src/modules/teleprompter"
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
	// TriggerTime is the wall-clock time the hotkey trigger was received.
	// Used to measure end-to-end latency from trigger to display or broadcast.
	TriggerTime time.Time
}

// Pipeline orchestrates the full screen-monitor workflow.
type Pipeline struct {
	settings          *config.Config
	logger            *slog.Logger
	capturer          capturer.Service
	extractor         extractor.Service
	agent             ai.Processor
	messenger         im.Broadcaster
	poller            poller.Service
	listener          listener.Service
	teleprompter      teleprompter.Service
	boundsMu          sync.RWMutex
	captureBounds     *image.Rectangle
	canvasBounds      *image.Rectangle // content area excluding browser chrome; nil for non-browsers
	lastWindowBounds  image.Rectangle  // raw foreground-window bounds for non-browser grid fallback
	capturedWindowPID int              // PID of the app from the last capture; monitor ignores other apps
	startTime         time.Time
	lastCaptureMu     sync.RWMutex
	lastCaptureTime   time.Time
	lastWindowTitle   string
	analyseQueue      chan CaptureResult

	// Grid positioning state.
	gridMu  sync.Mutex
	gridCol float64 // 0.0–1.0 horizontal position (5% steps), default 0.5
	gridRow float64 // 0.0–1.0 vertical position (5% steps), default 0.5

	// Visibility state shared between trackTeleprompterGridPosition and trackToggles.
	// intendedVisible is the user's desired visibility; movingForGrid is true while a
	// debounce-animation sequence is in progress. Both protected by visibleMu.
	visibleMu       sync.RWMutex
	intendedVisible bool
	movingForGrid   bool

	// outputMethod stores the active output adapter ("telegram" or "teleprompter").
	// Togglable at runtime via IPC. Uses atomic.Value for lock-free reads.
	outputMethod atomic.Value
}
