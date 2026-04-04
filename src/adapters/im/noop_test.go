package im_test

import (
	"context"
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
)

func TestNoopBroadcaster_Broadcast_returnsNil(t *testing.T) {
	broadcaster := &im.NoopBroadcaster{}

	err := broadcaster.Broadcast(context.Background(), "some message")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestNoopBroadcaster_Broadcast_emptyText(t *testing.T) {
	broadcaster := &im.NoopBroadcaster{}

	err := broadcaster.Broadcast(context.Background(), "")

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
