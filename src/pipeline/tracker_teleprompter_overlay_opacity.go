package pipeline

import "github.com/vdyalex/lens-daemon/src/bridges/appkit"

// trackTeleprompterOverlayOpacity receives direction events from +/- keys and adjusts the teleprompter
// text opacity by 0.01 per step. direction: -1 = decrease (minus), +1 = increase (plus),
// 0 = reset to configured default.
// Exits when the channel is closed.
func (p *Pipeline) trackTeleprompterOverlayOpacity(directions <-chan int) {
	const step = 0.01
	for direction := range directions {
		if direction == 0 {
			appkit.ResetTextOpacity()
			p.logger.Debug("teleprompter text opacity reset to default")
			continue
		}
		delta := step * float64(direction)
		opacity := appkit.SetTextOpacity(delta)
		p.logger.Debug("teleprompter text opacity adjusted", "opacity", opacity)
	}
}
