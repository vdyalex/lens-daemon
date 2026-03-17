package ipc

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"time"
)

// NewCommandHandler constructs a CommandHandler.
func NewCommandHandler(
	pipeline PipelineService,
	broker *LogBroker,
	cancelFunc context.CancelFunc,
	logger *slog.Logger,
) *CommandHandler {
	return &CommandHandler{
		pipeline:   pipeline,
		broker:     broker,
		cancelFunc: cancelFunc,
		startTime:  time.Now(),
		logger:     logger,
	}
}

// Handle implements Handler. Dispatches to the appropriate command handler.
func (h *CommandHandler) Handle(ctx context.Context, conn net.Conn, req Request) (Response, error) {
	switch req.Command {
	case CommandStatus:
		return h.handleStatus(ctx)
	case CommandShutdown:
		return h.handleShutdown()
	case CommandLogSubscribe:
		return h.handleLogSubscribe(ctx, conn)
	default:
		return Response{OK: false, Error: "unknown command"}, nil
	}
}

// handleStatus returns the current pipeline status as a StatusPayload.
func (h *CommandHandler) handleStatus(ctx context.Context) (Response, error) {
	pid, uptimeSeconds, lastCaptureTime, lastWindowTitle := h.pipeline.Status()

	payload := StatusPayload{
		PID:             pid,
		UptimeSeconds:   uptimeSeconds,
		LastCaptureTime: lastCaptureTime,
		LastWindowTitle: lastWindowTitle,
		SubscriberCount: h.broker.Count(),
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
	h.cancelFunc()
	return Response{OK: true}, nil
}

// handleLogSubscribe subscribes to log events and streams them to the client.
// The handler writes LogEvent frames directly to the connection; no response envelope is sent.
func (h *CommandHandler) handleLogSubscribe(ctx context.Context, conn net.Conn) (Response, error) {
	id, eventCh := h.broker.Subscribe()
	defer h.broker.Unsubscribe(id)

	// Stream events to client
	for {
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case event := <-eventCh:
			// Send event frame
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("failed to marshal log event", "error", err)
				return Response{}, err
			}
			if err := WriteFrame(conn, data); err != nil {
				return Response{}, err
			}
		}
	}
}
