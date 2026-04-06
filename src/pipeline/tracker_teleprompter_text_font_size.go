package pipeline

import "github.com/vdyalex/lens-daemon/src/bridges/appkit"

// trackTeleprompterTextFontSize receives direction events from comma/period keys and adjusts the
// teleprompter font size by 0.5pt per step. direction: -1 = decrease (comma),
// +1 = increase (period).
// Exits when the channel is closed.
func (p *Pipeline) trackTeleprompterTextFontSize(directions <-chan int) {
	const step = 0.5
	for direction := range directions {
		if !p.isTeleprompterActive() {
			continue
		}
		delta := step * float64(direction)
		size := appkit.SetFontSize(delta)
		p.logger.Debug("teleprompter font size adjusted", "size", size)
	}
}
