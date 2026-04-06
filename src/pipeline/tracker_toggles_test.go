package pipeline

import (
	"log/slog"
	"testing"
)

func TestTrackToggles_flipsVisibility(t *testing.T) {
	toggles := make(chan struct{}, 3)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default(), intendedVisible: false}

	go func() {
		pipeline.trackToggles(toggles)
		close(done)
	}()

	// First toggle: false → true.
	toggles <- struct{}{}
	// Second toggle: true → false.
	toggles <- struct{}{}
	// Third toggle: false → true.
	toggles <- struct{}{}
	close(toggles)
	<-done

	pipeline.visibleMu.RLock()
	defer pipeline.visibleMu.RUnlock()

	if !pipeline.intendedVisible {
		t.Error("intendedVisible should be true after 3 toggles from false")
	}
}

func TestTrackToggles_startsVisible(t *testing.T) {
	toggles := make(chan struct{}, 1)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default(), intendedVisible: true}

	go func() {
		pipeline.trackToggles(toggles)
		close(done)
	}()

	toggles <- struct{}{}
	close(toggles)
	<-done

	pipeline.visibleMu.RLock()
	defer pipeline.visibleMu.RUnlock()

	if pipeline.intendedVisible {
		t.Error("intendedVisible should be false after 1 toggle from true")
	}
}
