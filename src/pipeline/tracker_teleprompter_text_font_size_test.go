package pipeline

import (
	"log/slog"
	"testing"
)

func TestTrackTeleprompterTextFontSize_consumesAllDirections(t *testing.T) {
	directions := make(chan int, 2)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default()}

	go func() {
		pipeline.trackTeleprompterTextFontSize(directions)
		close(done)
	}()

	directions <- 1
	directions <- -1
	close(directions)
	<-done
}

func TestTrackTeleprompterTextFontSize_exitsOnChannelClose(t *testing.T) {
	directions := make(chan int)
	done := make(chan struct{})
	pipeline := &Pipeline{logger: slog.Default()}

	go func() {
		pipeline.trackTeleprompterTextFontSize(directions)
		close(done)
	}()

	close(directions)
	<-done
}
