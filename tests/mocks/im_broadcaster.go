package mocks

import "context"

// MockIMBroadcaster satisfies im.Broadcaster interface.
type MockIMBroadcaster struct {
	BroadcastFunc func(ctx context.Context, text string) error
	Calls         []string
}

// Broadcast implements im.Broadcaster.
func (mock *MockIMBroadcaster) Broadcast(ctx context.Context, text string) error {
	mock.Calls = append(mock.Calls, text)
	return mock.BroadcastFunc(ctx, text)
}
