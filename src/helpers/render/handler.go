// Package render provides log rendering for the daemon and CLI.
package render

import (
	"context"
	"io"
	"log/slog"
	"os"

	"golang.org/x/term"

	"github.com/vdyalex/lens-daemon/src/helpers/logger"
	"github.com/vdyalex/lens-daemon/src/ipc"
)

// NewDaemonHandler creates a slog.Handler appropriate for the current environment.
// If stderr is a terminal (interactive), it writes slog lines to the broker only
// and starts a pterm rendering goroutine so log events are colorized — the same
// rendering path used by 'lensd logs'. Otherwise (daemonized, stderr redirected
// to a log file), it writes plain text to both stderr and the broker.
func NewDaemonHandler(ctx context.Context, broker *ipc.LogBroker, options *slog.HandlerOptions) slog.Handler {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return slog.NewTextHandler(io.MultiWriter(os.Stderr, broker), options)
	}

	go func() {
		id, eventCh := broker.Subscribe()
		defer broker.Unsubscribe(id)
		ptermLogger := logger.New()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				logger.Print(ptermLogger, event)
			}
		}
	}()

	return slog.NewTextHandler(broker, options)
}
