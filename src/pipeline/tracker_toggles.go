package pipeline

import (
	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// trackToggles receives toggle events from the hotkey listener and flips the
// teleprompter visibility. When a grid-move animation is in progress, the toggle
// updates gIntendedVisible (via Show/HideOverlay) so the move's completion handler
// applies the correct final state without a visual conflict.
// Exits when the channel is closed.
func (p *Pipeline) trackToggles(toggles <-chan struct{}) {
	for range toggles {
		if !p.isMethodActive(constants.OutputMethodTeleprompter) {
			continue
		}

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
