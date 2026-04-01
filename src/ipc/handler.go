package ipc

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"

	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// CommandHandler dispatches IPC requests to pipeline and log broker operations.
type CommandHandler struct {
	pipeline    PipelineService
	broker      *LogBroker
	cancel      context.CancelFunc
	logger      *slog.Logger
	subscribers func() int
}

// NewCommandHandler constructs a CommandHandler.
// subscribers returns the current number of Telegram subscribers.
func NewCommandHandler(
	pipeline PipelineService,
	broker *LogBroker,
	cancel context.CancelFunc,
	logger *slog.Logger,
	subscribers func() int,
) *CommandHandler {
	return &CommandHandler{
		pipeline:    pipeline,
		broker:      broker,
		cancel:      cancel,
		logger:      logger,
		subscribers: subscribers,
	}
}

// Handle implements Handler. Dispatches to the appropriate command handler.
func (h *CommandHandler) Handle(ctx context.Context, connection net.Conn, request Request) (Response, error) {
	switch request.Command {
	case CommandStatus:
		return h.handleStatus()
	case CommandShutdown:
		return h.handleShutdown()
	case CommandLogSubscribe:
		return h.handleLogSubscribe(ctx, connection)
	default:
		return Response{OK: false, Error: "unknown command"}, nil
	}
}

// handleStatus returns the current pipeline status as a StatusPayload.
func (h *CommandHandler) handleStatus() (Response, error) {
	pid, uptime, lastCaptureTime, lastWindowTitle := h.pipeline.Status()

	payload := StatusPayload{
		PID:             pid,
		Uptime:          uptime,
		LastCaptureTime: lastCaptureTime,
		LastWindowTitle: lastWindowTitle,
		Subscribers:     h.subscribers(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal status payload", "error", err)
		return Response{OK: false, Error: "internal error"}, nil
	}
	return Response{OK: true, Payload: data}, nil
}

// handleShutdown cancels the daemon context to trigger graceful shutdown.
func (h *CommandHandler) handleShutdown() (Response, error) {
	h.cancel()
	return Response{OK: true}, nil
}

// handleLogSubscribe subscribes to log events and streams them to the client.
// The handler writes LogEvent frames directly to the connection; no response envelope is sent.
func (h *CommandHandler) handleLogSubscribe(ctx context.Context, connection net.Conn) (Response, error) {
	id, eventCh := h.broker.Subscribe()
	defer h.broker.Unsubscribe(id)

	// Stream events to client
	for {
		select {
		case <-ctx.Done():
			return Response{}, exceptions.ErrClientDisconnected
		case event := <-eventCh:
			// Send event frame
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("failed to marshal log event", "error", err)
				return Response{}, err
			}
			if err := WriteFrame(connection, data); err != nil {
				return Response{}, exceptions.ErrClientDisconnected
			}
		}
	}
}
