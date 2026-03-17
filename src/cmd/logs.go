package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/ipc"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream daemon logs",
	Long:  `Subscribes to daemon logs via IPC and prints level-colorized output. Press Ctrl-C to stop.`,
	Run: func(cmd *cobra.Command, args []string) {
		runLogs()
	},
}

// attrsToArgs converts a map of attributes into a flattened slice of alternating key-value pairs
// suitable for use with pterm logger methods. Values are sanitized to escape control characters.
func attrsToArgs(attrs map[string]any) []any {
	pairs := make([]any, 0, len(attrs)*2)
	for key, value := range attrs {
		sanitized := sanitizeAttrValue(value)
		pairs = append(pairs, key, sanitized)
	}
	return pairs
}

// sanitizeAttrValue escapes control characters in string attribute values to prevent multiline output.
// Escapes: newline (\n), tab (\t), carriage return (\r), and backslash (\\).
// Non-string values are returned unchanged.
func sanitizeAttrValue(value any) any {
	s, ok := value.(string)
	if !ok {
		return value
	}
	// Escape backslash first to avoid double-escaping
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

// runLogs subscribes to the IPC log stream and prints colorized log output to the terminal.
// Runs until Ctrl-C is pressed or the daemon connection is closed.
func runLogs() {
	client := ipc.NewClient(ipc.DefaultSocketPath())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logChan, err := client.Subscribe(ctx)
	if err != nil {
		pterm.Fatal.Printfln("failed to subscribe to logs: %v", err)
	}

	logger := pterm.DefaultLogger.WithLevel(pterm.LogLevelDebug)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			cancel()
			return
		case <-ctx.Done():
			return
		case event, ok := <-logChan:
			if !ok {
				return
			}
			args := logger.Args(attrsToArgs(event.Attrs)...)
			switch event.Level {
			case "DEBUG":
				logger.Debug(event.Message, args)
			case "INFO":
				logger.Info(event.Message, args)
			case "WARN":
				logger.Warn(event.Message, args)
			case "ERROR":
				logger.Error(event.Message, args)
			default:
				logger.Info(event.Message, args)
			}
		}
	}
}
