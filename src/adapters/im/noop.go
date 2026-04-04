// Package im provides Telegram broadcast capabilities for sending messages to subscribers.
package im

import "context"

// NoopBroadcaster is a null-object Broadcaster that silently discards all messages.
// Used when Telegram is disabled (no bot token configured).
type NoopBroadcaster struct{}

// Broadcast discards the message and returns nil.
// ctx: unused; text: discarded.
func (n *NoopBroadcaster) Broadcast(_ context.Context, _ string) error {
	return nil
}
