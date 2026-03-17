package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vdyalex/lens-daemon/src/pipeline"
	"github.com/vdyalex/lens-daemon/src/utils/config"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Error("settings error", "error", err)
		os.Exit(1)
	}

	handlerOptions := &slog.HandlerOptions{Level: settings.LogLevel}
	handler := slog.NewTextHandler(os.Stderr, handlerOptions)
	logger := slog.New(handler)

	process, err := pipeline.New(settings, logger)
	if err != nil {
		logger.Error("pipeline initialization error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-channel
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
	}()

	if err := process.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("pipeline error", "error", err)
		os.Exit(1)
	}
}
