package pipeline

import (
	"sync"
	"time"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
)

// trackTeleprompterGridPosition receives 2-D direction events from the hotkey listener and
// maintains the (gridCol, gridRow) position as percentages in [0.0, 1.0].
// Each arrow press moves by p.settings.GridStep (5%). Values wrap circularly.
//
// Animation protocol:
//  1. First arrow press while visible → fade out (alpha=0, window stays open).
//  2. Subsequent presses within GridMoveDebounceDuration → extend the debounce timer;
//     target position advances silently.
//  3. Timer fires → reposition to target spot → fade in (if still intended visible).
//
// When the teleprompter is hidden, arrow presses update the position silently so
// the next show-toggle reveals it at the correct spot.
//
// Exits when the directions channel is closed.
func (p *Pipeline) trackTeleprompterGridPosition(directions <-chan [2]int) {
	var (
		mu            sync.Mutex
		fading        bool
		debounceTimer *time.Timer
	)

	commit := func(col, row float64) {
		appkit.CommitMoveToGridSpot(col, row)

		p.visibleMu.Lock()
		p.movingForGrid = false
		shouldShow := p.intendedVisible
		p.visibleMu.Unlock()

		mu.Lock()
		fading = false
		mu.Unlock()

		if shouldShow {
			appkit.FadeInAfterMove()
		}
		p.logger.Debug("grid move committed", "col", col, "row", row, "show", shouldShow)
	}

	for direction := range directions {
		p.gridMu.Lock()
		p.gridCol = wrapPercent(p.gridCol + float64(direction[0])*p.settings.GridStep)
		p.gridRow = wrapPercent(p.gridRow + float64(direction[1])*p.settings.GridStep)
		col := p.gridCol
		row := p.gridRow
		p.gridMu.Unlock()

		mu.Lock()
		alreadyFading := fading
		if !alreadyFading {
			fading = true
		}
		mu.Unlock()

		if !alreadyFading {
			p.visibleMu.Lock()
			p.movingForGrid = true
			visible := p.intendedVisible
			p.visibleMu.Unlock()

			if visible {
				appkit.FadeOutForMove()
			}
			p.logger.Debug("grid move started", "col", col, "row", row)
		}

		mu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(p.settings.GridMoveDebounceDuration, func() {
			commit(col, row)
		})
		mu.Unlock()
	}
}

// wrapPercent wraps a percentage value to [0.0, 1.0) with circular behavior.
// Values below 0 wrap to the top; values at or above 1 wrap to the bottom.
func wrapPercent(value float64) float64 {
	value = value - float64(int(value))
	if value < 0 {
		value += 1.0
	}
	return value
}
