package pipeline

import (
	"image"
	"log/slog"
	"testing"
)

func TestTrackBounds_updatesCaptureBounds(t *testing.T) {
	bounds := make(chan image.Rectangle, 1)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default()}

	go func() {
		pipeline.trackBounds(bounds)
		close(done)
	}()

	expected := image.Rect(10, 20, 300, 400)
	bounds <- expected
	close(bounds)
	<-done

	pipeline.boundsMu.RLock()
	defer pipeline.boundsMu.RUnlock()

	if pipeline.captureBounds == nil {
		t.Fatal("captureBounds is nil after trackBounds received a rectangle")
	}
	if *pipeline.captureBounds != expected {
		t.Errorf("captureBounds = %v, want %v", *pipeline.captureBounds, expected)
	}
}

func TestTrackBounds_lastRectangleWins(t *testing.T) {
	bounds := make(chan image.Rectangle, 2)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default()}

	go func() {
		pipeline.trackBounds(bounds)
		close(done)
	}()

	bounds <- image.Rect(0, 0, 10, 10)
	bounds <- image.Rect(50, 60, 500, 600)
	close(bounds)
	<-done

	pipeline.boundsMu.RLock()
	defer pipeline.boundsMu.RUnlock()

	expected := image.Rect(50, 60, 500, 600)
	if pipeline.captureBounds == nil || *pipeline.captureBounds != expected {
		t.Errorf("captureBounds = %v, want %v", pipeline.captureBounds, expected)
	}
}
