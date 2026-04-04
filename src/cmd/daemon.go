package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/bridges/appkit"
	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/helpers/render"
	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/src/pipeline"
	"github.com/vdyalex/lens-daemon/src/utils/buildinfo"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the pipeline (used by start command)",
	Long:  fmt.Sprintf("Runs the pipeline with IPC server. Intended to be called by '%s start' via re-exec.", buildinfo.BinaryName),
	Run: func(cmd *cobra.Command, args []string) {
		runDaemon()
	},
}

// applyFlags applies global CLI flags to environment variables.
// This allows config to be passed as command-line arguments to daemon/start/restart commands.
// Returns an error if any environment variable cannot be set.
func applyFlags() error {
	for _, pair := range FlagEnvPairs() {
		if err := os.Setenv(pair.Key, pair.Value); err != nil {
			return err
		}
	}
	return nil
}

// runDaemon initializes and runs the daemon: loads config, starts IPC server, and runs the pipeline.
// The main goroutine is yielded to the AppKit run loop for teleprompter rendering;
// all daemon logic runs in a background goroutine.
// Exits with status 1 on configuration or runtime errors.
func runDaemon() {
	// Apply flags as environment variables before loading config
	if err := applyFlags(); err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Error("failed to apply flags", "error", err)
		os.Exit(1)
	}

	settings, err := config.Load()
	if err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Error("settings error", "error", err)
		os.Exit(1)
	}

	// Run all daemon logic in a background goroutine so the main goroutine
	// (pinned to the main OS thread) can run the AppKit event loop.
	go runPipeline(settings)

	// Block the main OS thread with the Cocoa event loop.
	// This is required for NSWindow operations in the appkit bridge.
	// StartRunLoop never returns.
	appkit.StartRunLoop()
}

// runPipeline runs the pipeline, IPC server, and signal handling.
// Intended to run in a background goroutine while the main thread runs AppKit.
func runPipeline(settings *config.Config) {
	// Create log broker for IPC fan-out
	broker := ipc.NewLogBroker()

	// Create context before logger so NewDaemonHandler can bound the renderer goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create logger with render handler (TTY: pterm colorized, non-TTY: plain text to stderr + broker)
	handlerOptions := &slog.HandlerOptions{Level: settings.LogLevel}
	logger := slog.New(render.NewDaemonHandler(ctx, broker, handlerOptions))

	// PID is written inside onReady (after IPC socket is listening).
	// This makes PID file presence a reliable "daemon ready" signal for the start command.
	pidPath := daemon.DefaultPIDPath()
	var pidWritten bool
	defer func() {
		if pidWritten {
			if err := daemon.RemovePID(pidPath); err != nil {
				logger.Warn("failed to remove pid file", "error", err)
			}
		}
	}()

	onReady := func() {
		if err := daemon.WritePID(pidPath); err != nil {
			logger.Error("failed to write pid file", "error", err)
			cancel()
			return
		}
		pidWritten = true
		logger.Info(fmt.Sprintf("%s started", buildinfo.BinaryName), "pid", os.Getpid())
	}

	// Create pipeline
	process, subscriberStore, err := pipeline.New(settings, logger)
	if err != nil {
		logger.Error("pipeline initialization error", "error", err)
		os.Exit(1)
	}

	// Wire IPC handler
	socketPath := ipc.DefaultSocketPath()
	subscribers := func() int {
		if subscriberStore == nil {
			return 0
		}
		return len(subscriberStore.All())
	}
	ipcHandler := ipc.NewCommandHandler(process, broker, cancel, logger, subscribers)
	ipcServer := ipc.NewServer(socketPath, ipcHandler, logger)

	// Start IPC server in background
	go func() {
		if err := ipcServer.Serve(ctx, onReady); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("ipc server error", "error", err)
		}
	}()

	// Handle signals for graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalCh
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
	}()

	// Run pipeline
	if err := process.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("pipeline error", "error", err)
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("%s stopped", buildinfo.BinaryName))
	os.Exit(0)
}
