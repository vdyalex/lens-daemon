// Package factory constructs Telegram adapter implementations for the pipeline.
package factory

import (
	"log/slog"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

// BroadcasterFactory builds the IM broadcaster adapter.
// When Store is nil (Telegram disabled), Build returns a NoopBroadcaster.
type BroadcasterFactory struct {
	Settings *config.Config
	Store    im.Store
	Logger   *slog.Logger
}

// Build returns a live Telegram broadcaster, or a NoopBroadcaster when Store is nil.
func (f BroadcasterFactory) Build() im.Broadcaster {
	if f.Store == nil {
		return &im.NoopBroadcaster{}
	}
	return im.New(
		f.Settings.TelegramBotToken,
		f.Store,
		f.Logger,
		f.Settings.TelegramMessageChunkSize,
		f.Settings.TelegramMaxRetries,
		f.Settings.TelegramHTTPClientTimeout,
	)
}
