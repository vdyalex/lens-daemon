// Package factory constructs Telegram adapter implementations for the pipeline.
package factory

import (
	"log/slog"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// PollerFactory builds the Telegram update polling adapter.
// When Store is nil (Telegram disabled), Build returns a NoopPoller.
// The initial active state is derived from Settings.OutputMethod.
type PollerFactory struct {
	Settings *config.Config
	Store    im.Store
	Logger   *slog.Logger
}

// Build returns a live Telegram poller, or a NoopPoller when Store is nil.
func (f PollerFactory) Build() poller.Service {
	if f.Store == nil {
		return &poller.NoopPoller{}
	}
	active := f.Settings.OutputMethod == constants.OutputMethodTelegram
	return poller.New(
		f.Settings.TelegramBotToken,
		f.Store,
		f.Logger,
		f.Settings.TelegramLongPollTimeout,
		f.Settings.TelegramPollerTimeout,
		f.Settings.TelegramHTTPClientTimeout,
		active,
	)
}
