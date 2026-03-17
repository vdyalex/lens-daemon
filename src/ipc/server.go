package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// Handler is the interface that processes an IPC Request and returns a Response.
type Handler interface {
	Handle(ctx context.Context, conn net.Conn, req Request) (Response, error)
}

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
// in its own goroutine. Returns ctx.Err() on normal shutdown.
func (s *Server) Serve(ctx context.Context) error {
	// Clean up stale socket
	os.Remove(s.socketPath)

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

	// Goroutine to close listener when context is cancelled
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// Normal shutdown
				os.Remove(s.socketPath)
				return ctx.Err()
			default:
				s.logger.Error("accept failed", "error", err)
				return err
			}
		}

		// Handle connection in goroutine
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// Read request frame
	reqData, err := ReadFrame(conn)
	if err != nil {
		s.logger.Warn("failed to read request frame", "error", err)
		return
	}

	// Unmarshal request
	var req Request
	if err := json.Unmarshal(reqData, &req); err != nil {
		s.logger.Warn("failed to unmarshal request", "error", err)
		resp := Response{OK: false, Error: "invalid request"}
		data, err := json.Marshal(resp)
		if err != nil {
			s.logger.Error("failed to marshal error response", "error", err)
			return
		}
		if err := WriteFrame(conn, data); err != nil {
			s.logger.Warn("failed to write error response", "error", err)
		}
		return
	}

	// Handle request
	resp, err := s.handler.Handle(ctx, conn, req)
	if err != nil {
		s.logger.Warn("handler error", "command", req.Command, "error", err)
		resp = Response{OK: false, Error: err.Error()}
	}

	// Special case: log.subscribe writes events directly to conn, not a response
	if req.Command == CommandLogSubscribe && resp.OK {
		return // Handler already wrote frames
	}

	// Send response
	respData, err := json.Marshal(resp)
	if err != nil {
		s.logger.Error("failed to marshal response", "error", err)
		return
	}

	if err := WriteFrame(conn, respData); err != nil {
		s.logger.Warn("failed to write response frame", "error", err)
	}
}

// DefaultSocketPath returns $TMPDIR/lensd-<uid>.sock
func DefaultSocketPath() string {
	tmpdir := os.TempDir()
	uid := "unknown"
	if u, err := user.Current(); err == nil {
		uid = u.Uid
	}
	return filepath.Join(tmpdir, fmt.Sprintf("lensd-%s.sock", uid))
}
