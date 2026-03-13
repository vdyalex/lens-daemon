package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vdyalex/ccat-assistant/internal/capture"
	"github.com/vdyalex/ccat-assistant/internal/config"
	"github.com/vdyalex/ccat-assistant/internal/pipeline"
)

func main() {
	// Hide the process from taskbar / Alt-Tab
	capture.HideProcess()

	cfg, err := config.Load()
	if err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	handlerOptions := &slog.HandlerOptions{Level: cfg.LogLevel}
	handler := slog.NewTextHandler(os.Stderr, handlerOptions)
	logger := slog.New(handler)

	p, err := pipeline.New(cfg, logger)
	if err != nil {
		logger.Error("pipeline initialization error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", slog.String("signal", sig.String()))
		cancel()
	}()

	if err := p.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("pipeline error", "error", err)
		os.Exit(1)
	}
}
