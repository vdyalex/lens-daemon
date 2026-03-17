package daemon

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// Daemonize re-execs the current binary with the "daemon" subcommand, fully detached.
// The child runs in a new session (Setsid: true) with stdin bound to /dev/null.
// The parent returns after cmd.Start() without waiting for the child.
// Returns ErrDaemonAlreadyRunning if a non-stale PID file already exists at opts.PIDPath.
func Daemonize(opts Options) error {
	// Check if daemon is already running before attempting re-exec
	_, err := ReadPID(opts.PIDPath)
	if err == nil {
		return exceptions.ErrDaemonAlreadyRunning
	}
	if err != exceptions.ErrDaemonNotRunning && err != exceptions.ErrPIDFileStale {
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, constants.SubcommandDaemon)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // New session detached from terminal
	}

	// Set stdin to /dev/null
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	cmd.Stdin = devNull

	// Redirect stdout/stderr
	logPath := opts.LogPath
	if logPath == "" {
		logPath = os.DevNull
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, constants.PermissionLogFile)
	if err != nil {
		devNull.Close()
		return err
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Set environment (include current env + extras)
	cmd.Env = append(os.Environ(), opts.ExtraEnv...)

	// Start child process (parent returns immediately)
	if err := cmd.Start(); err != nil {
		devNull.Close()
		logFile.Close()
		return err
	}

	// Close our copies of file descriptors; child has its own
	devNull.Close()
	logFile.Close()

	// Parent exits here; child runs in new session
	return nil
}
