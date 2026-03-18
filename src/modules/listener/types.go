//go:generate mockgen -destination=../../tests/mocks/mock_listener_service.go -package=mocks -source=types.go -mock_names Service=MockListenerService Service

package listener

import (
	"context"
	"image"
	"log/slog"
	"sync"
	"time"
)

// Service abstracts the hotkey listener for testability.
type Service interface {
	Listen(ctx context.Context, logger *slog.Logger, pollInterval time.Duration,
		triggerKeycode, boundsKeycode int) (<-chan struct{}, <-chan image.Rectangle, error)
}

// Listener manages global hotkey detection and bounds tracking.
type Listener struct {
	triggerCh chan struct{}
	boundsCh  chan image.Rectangle
	startOnce sync.Once
}
