package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/ipc"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display daemon status",
	Long:  `Checks PID file and queries IPC for daemon status details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

// runStatus checks the daemon PID file and queries IPC for status information.
// Prints a formatted status table to the console with daemon state, PID, uptime, and last window.
// Returns an error if the daemon is not running.
func runStatus() error {
	pid, err := daemon.ReadPID(daemon.DefaultPIDPath())
	if err != nil {
		pterm.Error.Println("Daemon stopped")
		return err
	}

	// Daemon is running; query for detailed status
	ipcClient := ipc.NewClient(ipc.DefaultSocketPath())
	request := ipc.Request{Command: ipc.CommandStatus}
	response, err := ipcClient.Send(context.Background(), request)
	if err != nil || !response.OK {
		pterm.Warning.Printfln("Daemon running | PID %d | Not responding to IPC", pid)
		return fmt.Errorf("daemon not responding")
	}

	var payload ipc.StatusPayload
	if err := json.Unmarshal(response.Payload, &payload); err != nil {
		pterm.Warning.Printfln("Daemon running | PID %d", pid)
		return fmt.Errorf("failed to parse status: %w", err)
	}

	rows := pterm.TableData{
		{"Status", pterm.Green("Running")},
		{"PID", fmt.Sprintf("%d", pid)},
		{"Uptime", formatSeconds(payload.Uptime)},
		{"Subscribers", fmt.Sprintf("%d", payload.Subscribers)},
	}
	if payload.LastWindowTitle != "" {
		rows = append(rows, []string{"Last window", payload.LastWindowTitle})
	}

	_ = pterm.DefaultTable.WithHasHeader(false).WithData(rows).Render()
	return nil
}

// formatSeconds converts uptime in seconds to a human-readable duration string.
// Returns format: "Xs" (seconds), "Xm" (minutes), or "Xh Xm" (hours and minutes).
func formatSeconds(seconds float64) string {
	if seconds < time.Minute.Seconds() {
		return fmt.Sprintf("%ds", int(seconds))
	}
	if seconds < time.Hour.Seconds() {
		return fmt.Sprintf("%dm", int(seconds/time.Minute.Seconds()))
	}
	return fmt.Sprintf("%dh%dm", int(seconds/time.Hour.Seconds()), int((int(seconds)%int(time.Hour.Seconds()))/int(time.Minute.Seconds())))
}
