// Package pipeline trackers: hotkey-driven and polling-based event handlers for
// capture bounds, visibility, opacity, and window-move/resize evasion.
package pipeline

import (
	"context"
	"image"
	"log/slog"
	"time"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/bridges/browser"
	"github.com/vdyalex/lens-daemon/src/bridges/core_graphics"
)

// trackBounds receives updated capture rectangles from the hotkey listener
// and stores them under boundsMu. Exits when the bounds channel is closed.
func (p *Pipeline) trackBounds(bounds <-chan image.Rectangle) {
	for rect := range bounds {
		p.boundsMu.Lock()
		p.captureBounds = &rect
		p.boundsMu.Unlock()
		p.logger.Info("capture bounds updated",
			slog.Int("minX", rect.Min.X),
			slog.Int("minY", rect.Min.Y),
			slog.Int("maxX", rect.Max.X),
			slog.Int("maxY", rect.Max.Y),
		)
	}
}

// trackVisibility receives toggle events from the hotkey listener and flips the
// teleprompter visibility. When a grid-move animation is in progress, the toggle
// updates gIntendedVisible (via Show/HideOverlay) so the move's completion handler
// applies the correct final state without a visual conflict.
// Exits when the channel is closed.
func (p *Pipeline) trackVisibility(toggles <-chan struct{}) {
	for range toggles {
		p.visibleMu.Lock()
		p.intendedVisible = !p.intendedVisible
		intended := p.intendedVisible
		p.visibleMu.Unlock()

		// ShowOverlay/HideOverlay update gIntendedVisible and defer the visual change
		// when gMoveInProgress is YES — the grid commit will handle fade-in/out.
		if intended {
			appkit.ShowOverlay()
		} else {
			appkit.HideOverlay()
		}
		p.logger.Debug("teleprompter visibility toggled", "visible", intended)
	}
}

// trackOpacity receives direction events from +/- keys and adjusts the teleprompter
// text opacity by 0.01 per step. direction: -1 = decrease (minus), +1 = increase (plus),
// 0 = reset to configured default.
// Exits when the channel is closed.
func (p *Pipeline) trackOpacity(directions <-chan int) {
	const step = 0.01
	for direction := range directions {
		if direction == 0 {
			appkit.ResetTextOpacity()
			p.logger.Debug("teleprompter text opacity reset to default")
			continue
		}
		delta := step * float64(direction)
		appkit.SetTextOpacity(delta)
		p.logger.Debug("teleprompter text opacity adjusted", "delta", delta)
	}
}

// trackWindowChanges polls the captured window's bounding rectangle at
// WindowMonitorInterval. When the bounds change, it fades out the teleprompter.
// Once the window has been stable for WindowStabilizeDelay, it recalculates the
// canvas bounds, updates the grid spot, and fades the teleprompter back in (if
// it was visible before the move).
//
// Uses CapturedWindowRect — a pure CGWindowListCopyWindowInfo metadata query —
// so it does not capture screen pixels and does not trigger the screen-capture
// indicator.
//
// Exits when ctx is cancelled.
func (p *Pipeline) trackWindowChanges(ctx context.Context) {
	ticker := time.NewTicker(p.settings.WindowMonitorInterval)
	defer ticker.Stop()

	var (
		stableTimer *time.Timer
		unstable    bool
	)

	for {
		select {
		case <-ctx.Done():
			if stableTimer != nil {
				stableTimer.Stop()
			}
			return

		case <-ticker.C:
			p.boundsMu.RLock()
			pid := p.capturedWindowPID
			last := p.lastWindowBounds
			p.boundsMu.RUnlock()

			// No captured window yet — nothing to track.
			if pid == 0 {
				continue
			}

			bounds := core_graphics.CapturedWindowRect(pid)
			if bounds == nil {
				continue
			}

			if boundsUnchanged(*bounds, last) {
				continue
			}

			// Bounds changed: record new bounds and start the evasion fade-out.
			settled := *bounds
			p.boundsMu.Lock()
			p.lastWindowBounds = settled
			p.boundsMu.Unlock()

			if !unstable {
				unstable = true
				appkit.FadeOutForMove()
				p.logger.Debug("window moved/resized, teleprompter fading out")
			}

			// Reset the stability timer: the window must be stable for
			// WindowStabilizeDelay before we restore the teleprompter.
			if stableTimer != nil {
				stableTimer.Stop()
			}
			stableTimer = time.AfterFunc(p.settings.WindowStabilizeDelay, func() {
				unstable = false
				p.restoreAfterWindowSettle(ctx, settled)
			})
		}
	}
}

// restoreAfterWindowSettle recalculates canvas bounds for the settled window,
// updates the appkit layer, repositions the teleprompter at the current grid spot,
// and fades it back in if it was intended to be visible.
func (p *Pipeline) restoreAfterWindowSettle(ctx context.Context, windowBounds image.Rectangle) {
	// Derive the title from the last known title (set during capture); it is used
	// for browser canvas detection only. If stale, the worst case is a non-browser
	// fallback which uses the raw window bounds.
	p.lastCaptureMu.RLock()
	title := p.lastWindowTitle
	p.lastCaptureMu.RUnlock()

	x := windowBounds.Min.X
	y := windowBounds.Min.Y
	w := windowBounds.Dx()
	h := windowBounds.Dy()

	canvas := browser.CanvasBounds(title, x, y, w, h)

	p.boundsMu.Lock()
	p.canvasBounds = canvas
	p.boundsMu.Unlock()

	if canvas != nil {
		appkit.SetOverlayCanvasBounds(
			float64(canvas.Min.X), float64(canvas.Min.Y),
			float64(canvas.Dx()), float64(canvas.Dy()),
		)
	} else {
		appkit.SetOverlayCanvasBounds(0, 0, 0, 0)
		appkit.SetOverlayWindowBounds(float64(x), float64(y), float64(w), float64(h))
	}

	p.gridMu.Lock()
	col := p.gridCol
	row := p.gridRow
	p.gridMu.Unlock()

	appkit.CommitMoveToGridSpot(col, row)

	p.visibleMu.RLock()
	shouldShow := p.intendedVisible
	p.visibleMu.RUnlock()

	if shouldShow {
		appkit.FadeInAfterMove()
	}

	p.logger.Debug("window settled, teleprompter restored", "col", col, "row", row, "show", shouldShow)
}

// boundsUnchanged returns true when a and b have identical coordinates.
func boundsUnchanged(a, b image.Rectangle) bool {
	return a.Min.X == b.Min.X &&
		a.Min.Y == b.Min.Y &&
		a.Max.X == b.Max.X &&
		a.Max.Y == b.Max.Y
}
