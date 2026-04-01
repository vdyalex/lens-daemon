package store_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

func TestNewStore_missingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")

	store, err := store.New(storePath, mocks.NopLogger())

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if store == nil {
		t.Fatal("expected store, got nil")
	}
	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected empty store, got %d IDs", len(all))
	}
}

func TestNewStore_loadsExistingIDs(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")

	// Create a subscriber file with existing IDs
	content := []byte("123\n456\n789\n")
	if err := os.WriteFile(storePath, content, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := store.New(storePath, mocks.NopLogger())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	all := store.All()
	if len(all) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(all))
	}

	// Check IDs are present (order doesn't matter in map)
	idMap := make(map[int64]bool)
	for _, id := range all {
		idMap[id] = true
	}
	for _, expected := range []int64{123, 456, 789} {
		if !idMap[expected] {
			t.Errorf("expected ID %d to be loaded, but not found", expected)
		}
	}
}

func TestNewStore_malformedLine(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")

	// Create a file with invalid content
	content := []byte("123\nnotanumber\n")
	if err := os.WriteFile(storePath, content, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := store.New(storePath, mocks.NopLogger())

	if err == nil {
		t.Errorf("expected parse error, got nil")
	}
}

func TestNewStore_blankLinesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")

	// Create a file with blank lines
	content := []byte("123\n\n456\n  \n789\n")
	if err := os.WriteFile(storePath, content, 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store, err := store.New(storePath, mocks.NopLogger())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	all := store.All()
	if len(all) != 3 {
		t.Errorf("expected 3 IDs (blank lines ignored), got %d", len(all))
	}
}

func TestStore_add_persists(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Add(42); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Read file directly
	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read store file: %v", err)
	}
	content := string(data)
	if !bytes.Contains(data, []byte("42")) {
		t.Errorf("expected file to contain '42', got %q", content)
	}
}

func TestStore_add_idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Add(42); err != nil {
		t.Fatalf("first Add failed: %v", err)
	}
	if err := store.Add(42); err != nil {
		t.Fatalf("second Add failed: %v", err)
	}

	all := store.All()
	if len(all) != 1 {
		t.Errorf("expected 1 ID after adding same ID twice, got %d", len(all))
	}
}

func TestStore_remove_persists(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Add(42); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := store.Remove(42); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected empty store after removal, got %d IDs", len(all))
	}
}

func TestStore_remove_idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Remove ID that never existed
	if err := store.Remove(99); err != nil {
		t.Fatalf("Remove on absent ID failed: %v", err)
	}

	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected empty store after removing absent ID, got %d IDs", len(all))
	}
}

func TestStore_all_returnsSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Add(1); err != nil {
		t.Fatalf("Add(1) failed: %v", err)
	}
	if err := store.Add(2); err != nil {
		t.Fatalf("Add(2) failed: %v", err)
	}
	if err := store.Add(3); err != nil {
		t.Fatalf("Add(3) failed: %v", err)
	}

	all := store.All()
	if len(all) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(all))
	}
}

func TestStore_add_createsParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "nested", "deep", "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Add(42); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read store file: %v", err)
	}
	if !bytes.Contains(data, []byte("42")) {
		t.Errorf("expected file to contain '42', got %q", string(data))
	}
}

func TestStore_concurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "subscribers")
	store, err := store.New(storePath, mocks.NopLogger())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	// 10 goroutines adding IDs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if err := store.Add(id); err != nil {
				errCh <- err
			}
		}(int64(i))
	}

	// 10 goroutines removing IDs
	for i := 10; i < 20; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if err := store.Remove(id); err != nil {
				errCh <- err
			}
		}(int64(i))
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Errorf("concurrent access error: %v", err)
		}
	}
}
