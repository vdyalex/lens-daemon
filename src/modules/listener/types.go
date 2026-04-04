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
		triggerKeycode, boundsKeycode, teleprompterKeycode int) (<-chan struct{}, <-chan image.Rectangle, <-chan struct{}, <-chan int, error)
}

// Listener manages global hotkey detection, bounds tracking, teleprompter toggling, and position changes.
type Listener struct {
	triggerCh      chan struct{}
	boundsCh       chan image.Rectangle
	teleprompterCh chan struct{}
	positionCh     chan int
	startOnce      sync.Once
}
