package mocks

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
)

// MockStore creates a store.Store backed by a temporary directory.
// It calls test.Cleanup to remove the directory after the test.
func MockStore(t *testing.T) *store.Store {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/subscribers"
	store, err := store.New(storePath, NopLogger())
	if err != nil {
		t.Fatalf("failed to create temp store: %v", err)
	}
	return store
}
