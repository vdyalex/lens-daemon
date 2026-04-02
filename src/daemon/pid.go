// Package daemon provides daemon lifecycle management including PID file operations and process daemonization.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
	"github.com/vdyalex/lens-daemon/src/utils/paths"
)

// WritePID atomically writes the current process PID to path.
// Creates parent directories with restricted permissions if absent.
// Fails if a non-stale PID file already exists (ErrDaemonAlreadyRunning).
func WritePID(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, constants.PermissionPIDDirectory); err != nil {
		return err
	}

	// Use O_EXCL to atomically detect if file already exists and is live
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, constants.PermissionPIDFile)
	if err != nil {
		// File already exists; check if it's stale
		if os.IsExist(err) {
			_, staleErr := ReadPID(path)
			if staleErr == exceptions.ErrPIDFileStale {
				// Stale file was removed by ReadPID; retry the open
				return writePIDToFile(path)
			}
			// Process is still running
			return exceptions.ErrDaemonAlreadyRunning
		}
		return err
	}
	defer f.Close()

	return writePIDContent(f, path)
}

// ReadPID reads the PID from path.
// Returns ErrDaemonNotRunning if the file does not exist.
// Returns ErrPIDFileStale if the process is not running; removes the stale file.
func ReadPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, exceptions.ErrDaemonNotRunning
		}
		return 0, err
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, err
	}

	if !isProcessRunning(pid) {
		os.Remove(path) // Best effort; don't fail if removal fails
		return 0, exceptions.ErrPIDFileStale
	}

	return pid, nil
}

// RemovePID deletes the PID file at path.
// Returns nil if the file does not exist.
func RemovePID(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

// WaitStop polls the PID file at path until the daemon process is gone or timeout elapses.
// Returns nil when ReadPID returns ErrDaemonNotRunning or ErrPIDFileStale.
// Returns a non-nil error if the process is still alive after timeout.
func WaitStop(path string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(constants.IntervalDaemonStopPoll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("daemon did not stop within %v", timeout)
		case <-ticker.C:
			_, err := ReadPID(path)
			if errors.Is(err, exceptions.ErrDaemonNotRunning) || errors.Is(err, exceptions.ErrPIDFileStale) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}
}

// DefaultPIDPath returns the canonical PID file path for the compiled binary: $TMPDIR/<binary>-<uid>.pid.
func DefaultPIDPath() string {
	return paths.DaemonPath("pid")
}

// writePIDToFile creates a new PID file at path and writes the current process PID.
// Used for the retry path after stale file removal.
func writePIDToFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, constants.PermissionPIDFile)
	if err != nil {
		return exceptions.ErrDaemonAlreadyRunning
	}
	defer f.Close()

	return writePIDContent(f, path)
}

// writePIDContent writes the current process PID to f and removes path on failure.
func writePIDContent(f *os.File, path string) error {
	currentPID := os.Getpid()
	if _, err := f.WriteString(strconv.Itoa(currentPID)); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}

// isProcessRunning checks if a process with the given PID is alive.
// Uses kill(pid, 0) semantics: no signal is sent; error indicates process absence.
func isProcessRunning(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
