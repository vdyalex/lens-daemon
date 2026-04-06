package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/ipc"
)

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a runtime setting on the running daemon",
	Long:  `Sends a set command to the running daemon via IPC. Supported keys: output-method (telegram, teleprompter).`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSet(args[0], args[1])
	},
}

// runSet sends a set command to the running daemon.
// key: setting key (e.g. "output-method").
// value: setting value (e.g. "telegram", "teleprompter").
// Returns an error if the daemon is unreachable or the key/value is rejected.
func runSet(key string, value string) error {
	payload := ipc.SetPayload{
		Key:   key,
		Value: value,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	request := ipc.Request{
		Command: ipc.CommandSet,
		Payload: data,
	}
	client := ipc.NewClient(ipc.DefaultSocketPath())
	response, err := client.Send(context.Background(), request)
	if err != nil {
		pterm.Error.Printfln("daemon not reachable: %v", err)
		return err
	}
	if !response.OK {
		pterm.Error.Printfln("set failed: %s", response.Error)
		return fmt.Errorf("set failed: %s", response.Error)
	}

	pterm.Success.Printfln("%s = %s", key, value)
	return nil
}
