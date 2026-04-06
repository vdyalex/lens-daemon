package pipeline

import (
	"log/slog"
	"testing"
)

func TestTrackTeleprompterOverlayOpacity_consumesAllDirections(t *testing.T) {
	directions := make(chan int, 3)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default()}

	go func() {
		pipeline.trackTeleprompterOverlayOpacity(directions)
		close(done)
	}()

	directions <- 1
	directions <- -1
	directions <- 0
	close(directions)
	<-done
}

func TestTrackTeleprompterOverlayOpacity_exitsOnChannelClose(t *testing.T) {
	directions := make(chan int)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default()}

	go func() {
		pipeline.trackTeleprompterOverlayOpacity(directions)
		close(done)
	}()

	close(directions)
	<-done
}
