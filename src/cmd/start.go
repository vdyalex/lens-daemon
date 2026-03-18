package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon in background",
	Long:  `Daemonizes the process by re-execing 'lensd daemon' in a new session.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStart()
	},
}

// buildExtraEnv converts global CLI flags to "KEY=VALUE" strings for the daemon subprocess.
// This allows configuration to be passed through re-exec to the child daemon process.
// Returns a slice of environment variable strings in the form "KEY=VALUE".
func buildExtraEnv() []string {
	pairs := FlagEnvPairs()
	env := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		env = append(env, pair.Key+"="+pair.Value)
	}
	return env
}

// runStart daemonizes the process by calling daemon.Daemonize and then polling
// for PID file creation to confirm the daemon has started.
func runStart() {
	pidPath := daemon.DefaultPIDPath()
	socketPath := ipc.DefaultSocketPath()

	opts := daemon.Options{
		PIDPath:  pidPath,
		ExtraEnv: buildExtraEnv(),
	}
	if err := daemon.Daemonize(opts); err != nil {
		pterm.Error.Printfln("start failed: %v", err)
		os.Exit(1)
	}

	spinner := NewSpinner("Starting lensd…")

	// Poll for PID file to confirm daemon started
	deadline := time.Now().Add(constants.TimeoutDaemonStartup)
	ticker := time.NewTicker(constants.IntervalDaemonStartupPoll)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pid, err := daemon.ReadPID(pidPath)
			if err == nil {
				spinner.Success(fmt.Sprintf("lensd started (pid %d, socket %s)", pid, socketPath))
				return
			}
			if err != exceptions.ErrDaemonNotRunning && err != exceptions.ErrPIDFileStale {
				spinner.Fail(fmt.Sprintf("start failed: %v", err))
				os.Exit(1)
			}
		default:
			if time.Now().After(deadline) {
				spinner.Fail("daemon did not start within timeout")
				os.Exit(1)
			}
			// Brief sleep to avoid busy-waiting
			time.Sleep(10 * time.Millisecond)
		}
	}
}
