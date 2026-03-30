package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/helpers/logger"
	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// logsCmd subscribes to and displays daemon logs.
// It waits for the daemon to start if not running, reconnects automatically
// after restarts, and streams colorized log output until interrupted.
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream daemon logs",
	Long:  `Subscribes to daemon logs via IPC. Waits for the daemon to start if not running and reconnects on restart. Press Ctrl-C to stop.`,
	Run: func(cmd *cobra.Command, args []string) {
		runLogs()
	},
}

// runLogs polls for the daemon and streams log events, reconnecting automatically on restart.
// Prints a warning when the daemon is not reachable and "connected to daemon" on each successful
// connection. Runs until Ctrl-C or SIGTERM is received.
func runLogs() {
	ipcClient := ipc.NewClient(ipc.DefaultSocketPath())
	ptermLogger := logger.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	waiting := false

	for {
		select {
		case <-signalCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		logChannel, err := ipcClient.Subscribe(ctx)
		if err != nil {
			if !errors.Is(err, exceptions.ErrIPCNotConnected) {
				pterm.Error.Printfln("failed to subscribe to logs: %v", err)
				os.Exit(1)
			}
			if !waiting {
				pterm.Warning.Println("daemon is not running — waiting for it to start...")
				waiting = true
			}
			select {
			case <-time.After(constants.LogsReconnectInterval):
			case <-signalCh:
				return
			case <-ctx.Done():
				return
			}
			continue
		}

		pterm.Info.Println("connected to daemon")

		disconnected := drainLogs(ctx, logChannel, ptermLogger, signalCh)
		if !disconnected {
			return
		}
		pterm.Warning.Println("disconnected from daemon — waiting for it to restart...")
		waiting = true
	}
}

// drainLogs reads from channel and prints colorized log events until channel closes or ctx/signal exits.
// Returns true when the channel is closed by the server (daemon stopped); false on signal or ctx exit.
func drainLogs(ctx context.Context, channel <-chan ipc.LogEvent, ptermLogger *pterm.Logger, signalCh <-chan os.Signal) bool {
	for {
		select {
		case <-signalCh:
			return false
		case <-ctx.Done():
			return false
		case event, ok := <-channel:
			if !ok {
				return true
			}
			logger.Print(ptermLogger, event)
		}
	}
}
