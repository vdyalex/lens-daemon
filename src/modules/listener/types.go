package listener

import (
	"image"
	"sync"
)

// Listener manages global hotkey detection and bounds tracking.
type Listener struct {
	triggerCh chan struct{}
	boundsCh  chan image.Rectangle
	startOnce sync.Once
}
