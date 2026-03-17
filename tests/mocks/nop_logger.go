package mocks

import (
	"io"
	"log/slog"
)

// NopLogger returns an slog.Logger that discards all output.
// Use in tests where log output is irrelevant to assertions.
func NopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
