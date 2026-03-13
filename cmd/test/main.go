package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vdyalex/test-assistant/internal/capture"
	"github.com/vdyalex/test-assistant/internal/config"
	"github.com/vdyalex/test-assistant/internal/pipeline"
)

func main() {
	// Hide the process from taskbar / Alt-Tab
	capture.HideProcess()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stderr)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	p, err := pipeline.New(cfg)
	if err != nil {
		log.Fatalf("pipeline initialization error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)
		cancel()
	}()

	if err := p.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("pipeline error: %v", err)
	}
}
