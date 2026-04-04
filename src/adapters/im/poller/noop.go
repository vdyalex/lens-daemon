// Package poller provides long-polling functionality for receiving Telegram messages.
package poller

import "context"

// NoopPoller is a null-object poller that blocks until the context is cancelled.
// Used when Telegram is disabled (no bot token configured).
type NoopPoller struct{}

// Run blocks until ctx is cancelled. No polling or network activity occurs.
// ctx: parent context; Run returns when ctx.Done() fires.
func (n *NoopPoller) Run(ctx context.Context) {
	<-ctx.Done()
}
