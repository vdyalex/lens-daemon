package mocks

import "context"

// MockIMBroadcaster satisfies im.Broadcaster interface.
type MockIMBroadcaster struct {
	BroadcastFunc func(ctx context.Context, text string) error
	Calls         []string
}

// Broadcast implements im.Broadcaster.
func (m *MockIMBroadcaster) Broadcast(ctx context.Context, text string) error {
	m.Calls = append(m.Calls, text)
	return m.BroadcastFunc(ctx, text)
}
