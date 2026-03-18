package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/src/pipeline"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the pipeline (used by start command)",
	Long:  `Runs the pipeline with IPC server. Intended to be called by 'lensd start' via re-exec.`,
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

	// Create log broker for IPC fan-out
	broker := ipc.NewLogBroker()

	// Create logger with multi-writer (stderr + broker)
	handlerOptions := &slog.HandlerOptions{Level: settings.LogLevel}
	handler := slog.NewTextHandler(io.MultiWriter(os.Stderr, broker), handlerOptions)
	logger := slog.New(handler)

	// Write PID file
	pidPath := daemon.DefaultPIDPath()
	if err := daemon.WritePID(pidPath); err != nil {
		logger.Error("failed to write pid file", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := daemon.RemovePID(pidPath); err != nil {
			logger.Warn("failed to remove pid file", "error", err)
		}
	}()

	logger.Info("lensd started", "pid", os.Getpid())

	// Create pipeline
	process, err := pipeline.New(settings, logger)
	if err != nil {
		logger.Error("pipeline initialization error", "error", err)
		os.Exit(1)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wire IPC handler
	socketPath := ipc.DefaultSocketPath()
	ipcHandler := ipc.NewCommandHandler(process, broker, cancel, logger)
	ipcServer := ipc.NewServer(socketPath, ipcHandler, logger)

	// Start IPC server in background
	go func() {
		if err := ipcServer.Serve(ctx); err != nil && err != context.Canceled {
			logger.Error("ipc server error", "error", err)
		}
	}()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
	}()

	// Run pipeline
	if err := process.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("pipeline error", "error", err)
		os.Exit(1)
	}

	logger.Info("lensd stopped")
}
