// Package daemon provides daemon lifecycle management including PID file operations and process daemonization.
package daemon

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
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
				f, retryErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, constants.PermissionPIDFile)
				if retryErr != nil {
					return exceptions.ErrDaemonAlreadyRunning
				}
				defer f.Close()
				currentPID := os.Getpid()
				if _, err := f.WriteString(strconv.Itoa(currentPID)); err != nil {
					os.Remove(path)
					return err
				}
				return nil
			}
			// Process is still running
			return exceptions.ErrDaemonAlreadyRunning
		}
		return err
	}
	defer f.Close()

	currentPID := os.Getpid()
	if _, err := f.WriteString(strconv.Itoa(currentPID)); err != nil {
		os.Remove(path)
		return err
	}
	return nil
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

// DefaultPIDPath returns the canonical PID file path: $TMPDIR/lensd-<uid>.pid
func DefaultPIDPath() string {
	tmpdir := os.TempDir()
	uid := "unknown"
	if u, err := user.Current(); err == nil {
		uid = u.Uid
	}
	return filepath.Join(tmpdir, fmt.Sprintf("lensd-%s.pid", uid))
}

// isProcessRunning checks if a process with the given PID is alive.
// Uses kill(pid, 0) semantics: no signal is sent; error indicates process absence.
func isProcessRunning(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
