package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
	"github.com/vdyalex/lens-daemon/src/utils/paths"
)

// Server listens on a Unix domain socket and dispatches requests.
type Server struct {
	socketPath string
	handler    Handler
	logger     *slog.Logger
}

// NewServer creates a Server bound to socketPath with mode 0600.
func NewServer(socketPath string, handler Handler, logger *slog.Logger) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
		logger:     logger,
	}
}

// Serve starts the accept loop. Blocks until ctx is cancelled.
// The socket file is removed on return. Each accepted connection is handled
// in its own goroutine. onReady is called after the socket is listening and
// permissions are set; pass nil if no callback is needed.
// Returns ctx.Err() on normal shutdown.
func (s *Server) Serve(ctx context.Context, onReady func()) error {
	// Clean up stale socket
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("failed to remove stale socket", "path", s.socketPath, "error", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()

	// Set permissions (owner read/write only)
	if err := os.Chmod(s.socketPath, constants.PermissionSocket); err != nil {
		return err
	}

	s.logger.Info("ipc server listening", "socket", s.socketPath)

	if onReady != nil {
		onReady()
	}

	// Goroutine to close listener when context is cancelled
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		connection, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// Normal shutdown
				if removeErr := os.Remove(s.socketPath); removeErr != nil && !os.IsNotExist(removeErr) {
					s.logger.Warn("failed to remove socket on shutdown", "path", s.socketPath, "error", removeErr)
				}
				return ctx.Err()
			default:
				s.logger.Error("accept failed", "error", err)
				return err
			}
		}

		// Handle connection in goroutine
		go s.handleConnection(ctx, connection)
	}
}

// handleConnection reads one request frame from the connection, dispatches it to the handler, and writes the response.
// It sets a read deadline to prevent slow clients from blocking. On ErrClientDisconnected, returns without logging.
func (s *Server) handleConnection(ctx context.Context, connection net.Conn) {
	defer connection.Close()

	// Set read deadline to prevent slow/malicious clients from holding goroutines
	if err := connection.SetReadDeadline(time.Now().Add(constants.IPCReadTimeout)); err != nil {
		s.logger.Warn("failed to set read deadline", "error", err)
		return
	}

	// Read request frame
	requestData, err := ReadFrame(connection)
	if err != nil {
		s.logger.Warn("failed to read request frame", "error", err)
		return
	}

	// Unmarshal request
	var request Request
	if err := json.Unmarshal(requestData, &request); err != nil {
		s.logger.Warn("failed to unmarshal request", "error", err)
		response := Response{OK: false, Error: "invalid request"}
		data, err := json.Marshal(response)
		if err != nil {
			s.logger.Error("failed to marshal error response", "error", err)
			return
		}
		if err := WriteFrame(connection, data); err != nil {
			s.logger.Warn("failed to write error response", "error", err)
		}
		return
	}

	// Handle request
	response, err := s.handler.Handle(ctx, connection, request)
	if err != nil {
		// Skip logging for normal client disconnections
		if !errors.Is(err, exceptions.ErrClientDisconnected) {
			s.logger.Warn("handler error", "command", request.Command, "error", err)
		}
		return
	}

	// Special case: log.subscribe writes events directly to connection, not a response
	if request.Command == CommandLogSubscribe && response.OK {
		return // Handler already wrote frames
	}

	// Send response
	responseData, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("failed to marshal response", "error", err)
		return
	}

	if err := WriteFrame(connection, responseData); err != nil {
		s.logger.Warn("failed to write response frame", "error", err)
	}
}

// DefaultSocketPath returns $TMPDIR/lensd-<uid>.sock
func DefaultSocketPath() string {
	return paths.DaemonPath("sock")
}
