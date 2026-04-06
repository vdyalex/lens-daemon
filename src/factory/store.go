// Package factory constructs Telegram adapter implementations for the pipeline.
package factory

import (
	"log/slog"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

// BuildStore opens the subscriber store when TelegramBotToken is set, or returns nil when it is not.
// Returns an error only when the store cannot be initialised.
// settings: application configuration.
// logger: structured logger for the disabled-Telegram info message.
func BuildStore(settings *config.Config, logger *slog.Logger) (im.Store, error) {
	if settings.TelegramBotToken == "" {
		logger.Info("telegram disabled (no bot token configured)")
		return nil, nil
	}
	return store.New(settings.TelegramSubscriberStorePath, logger)
}
