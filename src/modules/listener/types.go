//go:generate mockgen -destination=../../../tests/mocks/mock_listener_service.go -package=mocks -source=types.go -mock_names Service=MockListenerService Service

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
	// TeleprompterGridPositions receives a [2]int{dx, dy} grid direction from arrow keys.
	// dx: -1 = left, +1 = right. dy: -1 = up, +1 = down.
	TeleprompterGridPositions <-chan [2]int
	// TeleprompterOverlayOpacities receives direction (-1 decrease, +1 increase, 0 reset) from minus/plus/zero keys.
	TeleprompterOverlayOpacities <-chan int
	// TeleprompterTextFontSizes receives direction (-1 decrease, +1 increase) from comma/period keys.
	TeleprompterTextFontSizes <-chan int
}

// Service abstracts the hotkey listener for testability.
type Service interface {
	Listen(ctx context.Context, logger *slog.Logger, pollInterval time.Duration,
		triggerKeycode, boundsKeycode, toggleKeycode int) (*Channels, error)
}

// Listener manages global hotkey detection, bounds tracking, teleprompter toggling, position changes, opacity adjustments, and font size changes.
type Listener struct {
	triggerCh                    chan struct{}
	boundsCh                     chan image.Rectangle
	togglesCh                    chan struct{}
	teleprompterGridPositionCh   chan [2]int
	teleprompterOverlayOpacityCh chan int
	teleprompterTextFontSizeCh   chan int
	startOnce                    sync.Once
}
