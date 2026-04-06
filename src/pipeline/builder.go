// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"log/slog"
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

// Builder constructs a Pipeline with injectable dependencies.
// Used directly in tests to inject mock implementations.
// settings and logger are required; all other dependencies must be set before Build is called.
type Builder struct {
	settings     *config.Config
	logger       *slog.Logger
	capturer     capturer.Service
	extractor    extractor.Service
	agent        ai.Processor
	broadcaster  im.Broadcaster
	poller       poller.Service
	listener     listener.Service
	teleprompter teleprompter.Service
}

// NewBuilder returns a Builder with the required settings and logger set.
// settings: application configuration; must not be nil.
// logger: structured logger; must not be nil.
func NewBuilder(settings *config.Config, logger *slog.Logger) *Builder {
	return &Builder{settings: settings, logger: logger}
}

// WithCapturer sets the screen capture service.
func (b *Builder) WithCapturer(s capturer.Service) *Builder {
	b.capturer = s
	return b
}

// WithExtractor sets the OCR text extraction service.
func (b *Builder) WithExtractor(s extractor.Service) *Builder {
	b.extractor = s
	return b
}

// WithAgent sets the AI processing service.
func (b *Builder) WithAgent(s ai.Processor) *Builder {
	b.agent = s
	return b
}

// WithBroadcaster sets the IM broadcast service.
func (b *Builder) WithBroadcaster(s im.Broadcaster) *Builder {
	b.broadcaster = s
	return b
}

// WithPoller sets the Telegram update polling service.
func (b *Builder) WithPoller(s poller.Service) *Builder {
	b.poller = s
	return b
}

// WithListener sets the hotkey event listener service.
func (b *Builder) WithListener(s listener.Service) *Builder {
	b.listener = s
	return b
}

// WithTeleprompter sets the teleprompter overlay service.
func (b *Builder) WithTeleprompter(s teleprompter.Service) *Builder {
	b.teleprompter = s
	return b
}

// Build constructs and returns the Pipeline.
// All With* dependencies must be set before calling Build.
func (b *Builder) Build() *Pipeline {
	return &Pipeline{
		settings:        b.settings,
		logger:          b.logger,
		capturer:        b.capturer,
		extractor:       b.extractor,
		agent:           b.agent,
		messenger:       b.broadcaster,
		poller:          b.poller,
		listener:        b.listener,
		teleprompter:    b.teleprompter,
		startTime:       time.Now(),
		analyseQueue:    make(chan CaptureResult, b.settings.AnalyseQueueCapacity),
		intendedVisible: b.settings.TeleprompterVisible,
		gridCol:         b.settings.TeleprompterGridInitialCol,
		gridRow:         b.settings.TeleprompterGridInitialRow,
	}
}
