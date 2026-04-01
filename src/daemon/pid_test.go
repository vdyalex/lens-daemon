package daemon_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestWaitStop_fileAbsentReturnsImmediately(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "nonexistent.pid")

	if err := daemon.WaitStop(pidPath, 1*time.Second); err != nil {
		t.Errorf("WaitStop() expected nil for absent file, got: %v", err)
	}
}

func TestWaitStop_staleFileReturnsImmediately(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "stale.pid")

	// Write a PID that is guaranteed not to be running
	if err := os.WriteFile(pidPath, []byte("99999999"), 0600); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	if err := daemon.WaitStop(pidPath, 1*time.Second); err != nil {
		t.Errorf("WaitStop() expected nil for stale PID, got: %v", err)
	}
}

func TestWaitStop_processStillRunningTimesOut(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "live.pid")

	// Write current process PID — this process is alive for the duration of the test
	if err := daemon.WritePID(pidPath); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}
	defer daemon.RemovePID(pidPath)

	start := time.Now()
	err := daemon.WaitStop(pidPath, 300*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("WaitStop() expected non-nil error when process is still running, got nil")
	}
	if elapsed < 280*time.Millisecond {
		t.Errorf("WaitStop() returned too early: elapsed %v, want >= 280ms", elapsed)
	}
}

func TestWaitStop_fileDisappearsBeforeTimeout(t *testing.T) {
	tmpdir := t.TempDir()
	pidPath := filepath.Join(tmpdir, "disappearing.pid")

	// Write current process PID so ReadPID initially sees a live process
	if err := daemon.WritePID(pidPath); err != nil {
		t.Fatalf("WritePID() error: %v", err)
	}

	// Remove the file after 150ms to simulate the daemon exiting
	go func() {
		time.Sleep(150 * time.Millisecond)
		daemon.RemovePID(pidPath)
	}()

	if err := daemon.WaitStop(pidPath, 1*time.Second); err != nil {
		t.Errorf("WaitStop() expected nil after file disappears, got: %v", err)
	}
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
