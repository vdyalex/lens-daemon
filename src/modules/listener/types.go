package listener

import (
	"image"
	"sync"
)

// Listener manages global hotkey detection and bounds tracking.
type Listener struct {
	triggerChannel chan struct{}
	boundsChannel  chan image.Rectangle
	startOnce      sync.Once
}
