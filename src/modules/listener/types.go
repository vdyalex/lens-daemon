//go:generate mockgen -destination=../../tests/mocks/mock_listener_service.go -package=mocks -source=types.go -mock_names Service=MockListenerService Service

package listener

import (
	"context"
	"image"
	"log/slog"
	"sync"
	"time"
)

// Channels holds the read-only event channels returned by Listen.
type Channels struct {
	// Triggers receives a value each time the capture trigger hotkey is pressed.
	Triggers <-chan struct{}
	// Bounds receives updated screen-coordinate rectangles when the bounds hotkey is released.
	Bounds <-chan image.Rectangle
	// Toggles receives a value each time the teleprompter toggle hotkey is pressed.
	Toggles <-chan struct{}
	// Positions receives rotation direction (-1 backward, +1 forward) from arrow keys.
	Positions <-chan int
	// Opacities receives direction (-1 decrease, +1 increase) from minus/plus keys.
	Opacities <-chan int
}

// Service abstracts the hotkey listener for testability.
type Service interface {
	Listen(ctx context.Context, logger *slog.Logger, pollInterval time.Duration,
		triggerKeycode, boundsKeycode, teleprompterKeycode int) (*Channels, error)
}

// Listener manages global hotkey detection, bounds tracking, teleprompter toggling, position changes, and opacity adjustments.
type Listener struct {
	triggerCh      chan struct{}
	boundsCh       chan image.Rectangle
	teleprompterCh chan struct{}
	positionCh     chan int
	opacityCh      chan int
	startOnce      sync.Once
}
