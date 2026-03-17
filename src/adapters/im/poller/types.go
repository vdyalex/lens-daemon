package poller

import (
	"context"
	"log/slog"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
)

// Service abstracts Telegram update polling.
type Service interface {
	Run(ctx context.Context)
}

// Poller long-polls Telegram's getUpdates API and dispatches /start and /stop commands
// to the subscriber store. It runs in a background goroutine.
type Poller struct {
	token           string
	store           im.Store
	client          im.HTTPClient
	logger          *slog.Logger
	offset          int64
	longPollTimeout time.Duration
	pollerTimeout   time.Duration
}
