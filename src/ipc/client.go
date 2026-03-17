package ipc

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// NewClient creates a Client for socketPath with a default timeout.
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    constants.TimeoutIPCClient,
	}
}

// Send dials the socket, sends request, reads one Response, and closes the connection.
// Returns ErrIPCNotConnected if the socket is not reachable.
func (c *Client) Send(ctx context.Context, request Request) (Response, error) {
	// Create a context with timeout
	dialCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var dialer net.Dialer
	connection, err := dialer.DialContext(dialCtx, "unix", c.socketPath)
	if err != nil {
		return Response{}, exceptions.ErrIPCNotConnected
	}
	defer connection.Close()

	// Set deadline for operations
	if err := connection.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return Response{}, err
	}

	// Marshal and send request
	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, err
	}

	if err := WriteFrame(connection, payload); err != nil {
		return Response{}, err
	}

	// Read response
	frame, err := ReadFrame(connection)
	if err != nil {
		return Response{}, err
	}

	var response Response
	if err := json.Unmarshal(frame, &response); err != nil {
		return Response{}, err
	}

	return response, nil
}

// Subscribe dials the socket, sends a log.subscribe Request, and streams LogEvents
// to the returned channel. The caller must cancel ctx to stop streaming.
// The channel is closed when the connection ends.
func (c *Client) Subscribe(ctx context.Context) (<-chan LogEvent, error) {
	// Create a context with timeout for initial connection
	dialCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var dialer net.Dialer
	connection, err := dialer.DialContext(dialCtx, "unix", c.socketPath)
	if err != nil {
		return nil, exceptions.ErrIPCNotConnected
	}

	// Send subscribe request
	request := Request{Command: CommandLogSubscribe}
	data, err := json.Marshal(request)
	if err != nil {
		connection.Close()
		return nil, err
	}
	if err := WriteFrame(connection, data); err != nil {
		connection.Close()
		return nil, err
	}

	// Stream events in background
	ch := make(chan LogEvent, constants.IPCLogSubscriberBuffer)

	go func() {
		defer connection.Close()
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			frame, err := ReadFrame(connection)
			if err != nil {
				return
			}

			var event LogEvent
			if err := json.Unmarshal(frame, &event); err != nil {
				return
			}

			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}
