package daemon_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

func TestWritePID_createsFile(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "test.pid")

	if err := daemon.WritePID(pidPath); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(pidPath); err != nil {
		t.Fatalf("PID file not created: %v", err)
	}

	// Clean up
	daemon.RemovePID(pidPath)
}

func TestWritePID_alreadyRunningFails(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "test.pid")

	// Write first PID (current process)
	if err := daemon.WritePID(pidPath); err != nil {
		t.Fatalf("First WritePID() error: %v", err)
	}

	// Try to write again (same process still running)
	err := daemon.WritePID(pidPath)
	if !errors.Is(err, exceptions.ErrDaemonAlreadyRunning) {
		t.Errorf("Expected ErrDaemonAlreadyRunning, got: %v", err)
	}

	daemon.RemovePID(pidPath)
}

func TestReadPID_fileAbsentReturnsNotRunning(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "nonexistent.pid")

	_, err := daemon.ReadPID(pidPath)
	if !errors.Is(err, exceptions.ErrDaemonNotRunning) {
		t.Errorf("Expected ErrDaemonNotRunning, got: %v", err)
	}
}

func TestReadPID_validPIDReturnsValue(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "test.pid")

	if err := daemon.WritePID(pidPath); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	pid, err := daemon.ReadPID(pidPath)
	if err != nil {
		t.Fatalf("ReadPID() error: %v", err)
	}

	currentPID := os.Getpid()
	if pid != currentPID {
		t.Errorf("Expected PID %d, got %d", currentPID, pid)
	}

	daemon.RemovePID(pidPath)
}

func TestRemovePID_idempotent(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "test.pid")

	if err := daemon.WritePID(pidPath); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	// Remove once
	if err := daemon.RemovePID(pidPath); err != nil {
		t.Fatalf("First RemovePID() error: %v", err)
	}

	// Remove again (file already gone, should be no-op)
	if err := daemon.RemovePID(pidPath); err != nil {
		t.Fatalf("Second RemovePID() error: %v", err)
	}
}
