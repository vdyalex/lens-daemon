package pipeline

import (
	"context"
	"image"
	"time"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/bridges/browser"
	"github.com/vdyalex/lens-daemon/src/bridges/core_graphics"
)

// trackWindowChanges polls the captured window's bounding rectangle at
// TeleprompterWindowMonitorInterval. When the bounds change, it fades out the teleprompter.
// Once the window has been stable for TeleprompterWindowStabilizeDelay, it recalculates the
// canvas bounds, updates the grid spot, and fades the teleprompter back in (if
// it was visible before the move).
//
// Uses CapturedWindowRect — a pure CGWindowListCopyWindowInfo metadata query —
// so it does not capture screen pixels and does not trigger the screen-capture
// indicator.
//
// Exits when ctx is cancelled.
func (p *Pipeline) trackWindowChanges(ctx context.Context) {
	ticker := time.NewTicker(p.settings.TeleprompterWindowMonitorInterval)
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
			// TeleprompterWindowStabilizeDelay before we restore the teleprompter.
			if stableTimer != nil {
				stableTimer.Stop()
			}
			stableTimer = time.AfterFunc(p.settings.TeleprompterWindowStabilizeDelay, func() {
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

	// Always store the raw window bounds so gridSpotFrame has an outer reference
	// for per-side margin calculation regardless of whether a canvas is active.
	appkit.SetOverlayWindowBounds(float64(x), float64(y), float64(w), float64(h))

	if canvas != nil {
		appkit.SetOverlayCanvasBounds(
			float64(canvas.Min.X), float64(canvas.Min.Y),
			float64(canvas.Dx()), float64(canvas.Dy()),
		)
	} else {
		appkit.SetOverlayCanvasBounds(0, 0, 0, 0)
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
