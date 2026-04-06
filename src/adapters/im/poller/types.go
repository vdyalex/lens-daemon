//go:generate mockgen -destination ../../../../tests/mocks/mock_poller_service.go -package mocks -mock_names Service=MockPollerService . Service

// Package poller provides long-polling functionality for receiving Telegram messages.
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
// When enabled returns false, the poller skips polling until the next cycle.
type Poller struct {
	token           string
	store           im.Store
	client          im.HTTPClient
	logger          *slog.Logger
	offset          int64
	longPollTimeout time.Duration
	pollerTimeout   time.Duration
	enabled         func() bool
}
