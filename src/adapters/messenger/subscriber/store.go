package subscriber

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Store manages a file-backed set of subscriber chat IDs.
// Add and Remove operations persist to disk atomically via os.Rename.
type Store struct {
	mu          sync.RWMutex
	subscribers map[int64]struct{}
	path        string
}

// NewStore loads existing subscribers from path (if it exists),
// optionally seeds with seed (if non-zero and not already present),
// and returns a ready-to-use Store.
func NewStore(path string, seed int64) (*Store, error) {
	s := &Store{
		subscribers: make(map[int64]struct{}),
		path:        path,
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if seed != 0 {
		s.subscribers[seed] = struct{}{}
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Add registers a subscriber. Idempotent. Persists to disk.
func (s *Store) Add(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[chatID] = struct{}{}
	return s.persist()
}

// Remove unregisters a subscriber. Idempotent. Persists to disk.
func (s *Store) Remove(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, chatID)
	return s.persist()
}

// All returns a snapshot of all current subscriber chat IDs.
func (s *Store) All() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]int64, 0, len(s.subscribers))
	for id := range s.subscribers {
		result = append(result, id)
	}
	return result
}

// load reads subscribers from the file as newline-delimited int64s.
// If the file does not exist, it is not an error.
func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // skip blank lines
		}
		id, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return err
		}
		s.subscribers[id] = struct{}{}
	}
	return scanner.Err()
}

// persist writes subscribers to a temp file and atomically renames it into place.
// Format: newline-delimited int64s (one per line).
func (s *Store) persist() error {
	var buf strings.Builder
	for id := range s.subscribers {
		buf.WriteString(strconv.FormatInt(id, 10))
		buf.WriteString("\n")
	}
	data := []byte(buf.String())

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
