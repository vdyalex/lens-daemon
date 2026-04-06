package pipeline

import (
	"image"
	"log/slog"
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
