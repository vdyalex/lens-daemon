// Package pipeline orchestrates the screenshot capture, OCR, AI processing, and Telegram broadcast workflow.
package pipeline

import (
	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// SetOutputMethod switches the active output adapter at runtime.
// When switching to telegram, hides the teleprompter overlay immediately.
// When switching to teleprompter, restores the overlay to the user's intended visibility state.
// method: constants.OutputMethodTelegram or constants.OutputMethodTeleprompter.
func (p *Pipeline) SetOutputMethod(method string) {
	p.outputMethod.Store(method)
	p.poller.SetActive(method == constants.OutputMethodTelegram)

	switch method {
	case constants.OutputMethodTeleprompter:
		p.visibleMu.RLock()
		intended := p.intendedVisible
		p.visibleMu.RUnlock()

		if intended {
			appkit.ShowOverlay()
		}
	default:
		appkit.HideOverlay()
	}

	p.logger.Info("output method changed", "method", method)
}

// OutputMethod returns the current runtime output method.
// Returns the runtime override if set via SetOutputMethod, otherwise the config value.
func (p *Pipeline) OutputMethod() string {
	value := p.outputMethod.Load()
	if value != nil {
		return value.(string)
	}
	return p.settings.OutputMethod
}

// isMethodActive reports whether the current output method equals method.
// method: one of constants.OutputMethodTeleprompter or constants.OutputMethodTelegram.
func (p *Pipeline) isMethodActive(method string) bool {
	return p.OutputMethod() == method
}
