package mocks

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
)

// MockStore creates a subscriber.Store backed by a temporary directory.
// It calls test.Cleanup to remove the directory after the test.
func MockStore(test *testing.T) *store.Store {
	tmpDir := test.TempDir()
	storePath := tmpDir + "/subscribers"
	store, err := store.NewStore(storePath, NopLogger())
	if err != nil {
		test.Fatalf("failed to create temp store: %v", err)
	}
	return store
}
