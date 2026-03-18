package cmd

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Long:  `Sends SIGTERM to the daemon via its PID file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStop(); err != nil {
			pterm.Error.Printfln("%v", err)
			os.Exit(1)
		}
	},
}

// runStop reads the daemon PID from the PID file and sends SIGTERM to the process.
// Returns an error if the daemon is not running, the PID file is stale, or the signal cannot be sent.
func runStop() error {
	pid, err := daemon.ReadPID(daemon.DefaultPIDPath())
	if errors.Is(err, exceptions.ErrDaemonNotRunning) || errors.Is(err, exceptions.ErrPIDFileStale) {
		return errors.New("lensd is not running")
	}
	if err != nil {
		return fmt.Errorf("stop failed: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("stop failed: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal failed: %w", err)
	}

	pterm.Success.Printfln("sent SIGTERM to lensd (pid %d)", pid)
	return nil
}
