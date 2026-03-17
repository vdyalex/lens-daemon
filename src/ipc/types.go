// Package ipc provides inter-process communication via Unix domain socket with length-prefixed JSON.
package ipc

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// CommandName identifies an IPC command.
type CommandName string

const (
	CommandStatus       CommandName = "status"
	CommandShutdown     CommandName = "shutdown"
	CommandLogSubscribe CommandName = "log.subscribe"
)

// Request is the JSON envelope sent from client to server.
type Request struct {
	Command CommandName     `json:"command"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response is the JSON envelope sent from server to client.
type Response struct {
	OK      bool            `json:"ok"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// LogEvent is streamed from server to subscribed clients.
type LogEvent struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// StatusPayload carries runtime status information for the IPC status command.
type StatusPayload struct {
	PID             int       `json:"pid"`
	UptimeSeconds   float64   `json:"uptime_seconds"`
	LastCaptureTime time.Time `json:"last_capture_time,omitempty"`
	LastWindowTitle string    `json:"last_window_title,omitempty"`
	SubscriberCount int       `json:"subscriber_count"`
}

// Client manages a connection to the daemon IPC socket.
type Client struct {
	socketPath string
	timeout    time.Duration
}

// PipelineService abstracts the pipeline for IPC handler testability.
type PipelineService interface {
	Status() (PID int, UptimeSeconds float64, LastCaptureTime time.Time, LastWindowTitle string)
}

// CommandHandler dispatches IPC requests to pipeline and log broker operations.
type CommandHandler struct {
	pipeline   PipelineService
	broker     *LogBroker
	cancelFunc context.CancelFunc
	startTime  time.Time
	logger     *slog.Logger
}

// LogBroker is a fan-out io.Writer that distributes log lines to registered subscribers.
// It parses slog text lines into LogEvent structs and sends them to all subscribers.
// It is safe for concurrent use. Slow subscribers are dropped (non-blocking send).
type LogBroker struct {
	mu          sync.RWMutex
	subscribers map[int]chan LogEvent
	nextID      int
}
