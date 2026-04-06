package pipeline

import "github.com/vdyalex/lens-daemon/src/bridges/appkit"
import "github.com/vdyalex/lens-daemon/src/utils/constants"

// trackTeleprompterTextFontSize receives direction events from comma/period/slash keys and adjusts the
// teleprompter font size by 0.5pt per step. direction: -1 = decrease (comma),
// +1 = increase (period), 0 = reset to configured default (slash/question mark).
// Exits when the channel is closed.
func (p *Pipeline) trackTeleprompterTextFontSize(directions <-chan int) {
	const step = 0.5
	for direction := range directions {
		if !p.isMethodActive(constants.OutputMethodTeleprompter) {
			continue
		}
		switch direction {
		case 0:
			appkit.ResetFontSize()
			p.logger.Debug("teleprompter font size reset to default")
		default:
			delta := step * float64(direction)
			size := appkit.SetFontSize(delta)
			p.logger.Debug("teleprompter font size adjusted", "size", size)
		}
	}
}
