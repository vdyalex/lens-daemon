// Package ipc provides inter-process communication via Unix domain socket with length-prefixed JSON.
package ipc

import (
	"context"
	"encoding/json"
	"net"
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
	Uptime          float64   `json:"uptime"`
	LastCaptureTime time.Time `json:"last_capture_time,omitempty"`
	LastWindowTitle string    `json:"last_window_title,omitempty"`
	Subscribers     int       `json:"subscribers"`
}

// PipelineService abstracts the pipeline for IPC handler testability.
type PipelineService interface {
	Status() (PID int, Uptime float64, LastCaptureTime time.Time, LastWindowTitle string)
}

// Handler is the interface that processes an IPC Request and returns a Response.
//
// Special case: for CommandLogSubscribe, the handler streams LogEvent frames directly
// to connection and returns (Response{}, error); the Response is ignored by the server.
// This asymmetry is intentional — a future StreamHandler interface can formalize it
// when a second streaming command is added.
type Handler interface {
	Handle(ctx context.Context, connection net.Conn, request Request) (Response, error)
}
