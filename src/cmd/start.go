package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
	"github.com/vdyalex/lens-daemon/src/utils/paths"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon in background",
	Long:  `Daemonizes the process by re-execing 'lensd daemon' in a new session.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStart()
	},
}

// printLastLogLines reads the daemon log file and prints the last 10 lines.
// Silently returns if the file is absent or empty, so the caller can exit cleanly.
func printLastLogLines(logPath string) {
	data, err := os.ReadFile(logPath)
	if err != nil || len(data) == 0 {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	start := max(0, len(lines)-constants.RecentLogLineCount)
	pterm.Info.Println("Last log output:")
	for _, line := range lines[start:] {
		fmt.Println(" ", line)
	}
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
	logPath := paths.DaemonPath("log")

	opts := daemon.Options{
		PIDPath:  pidPath,
		LogPath:  logPath,
		ExtraEnv: buildExtraEnv(),
	}
	if err := daemon.Daemonize(opts); err != nil {
		pterm.Error.Printfln("start failed: %v", err)
		os.Exit(1)
	}

	spinner := NewSpinner("Starting lensd…")

	// Poll for PID file to confirm daemon started
	deadlineCtx, cancel := context.WithTimeout(context.Background(), constants.TimeoutDaemonStartup)
	defer cancel()

	ticker := time.NewTicker(constants.IntervalDaemonStartupPoll)
	defer ticker.Stop()

	for {
		select {
		case <-deadlineCtx.Done():
			spinner.Fail("daemon did not start within timeout")
			printLastLogLines(logPath)
			os.Exit(1)
		case <-ticker.C:
			pid, err := daemon.ReadPID(pidPath)
			if err == nil {
				spinner.Success(fmt.Sprintf("lensd started (pid %d)", pid))
				pterm.Info.Printfln("Socket: %s", socketPath)
				pterm.Info.Printfln("Logs:   %s", logPath)
				return
			}
			if err != exceptions.ErrDaemonNotRunning && err != exceptions.ErrPIDFileStale {
				spinner.Fail(fmt.Sprintf("start failed: %v", err))
				os.Exit(1)
			}
		}
	}
}
