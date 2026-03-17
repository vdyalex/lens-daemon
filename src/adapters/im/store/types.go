package store

import (
	"log/slog"
	"sync"
)

// Store manages a file-backed set of subscriber chat IDs.
// Add and Remove operations persist to disk atomically via os.Rename.
type Store struct {
	mu          sync.RWMutex
	subscribers map[int64]struct{}
	path        string
	logger      *slog.Logger
}
