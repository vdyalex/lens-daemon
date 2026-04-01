package ipc_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

type mockPipeline struct {
	pid             int
	uptime          float64
	lastCaptureTime time.Time
	lastWindowTitle string
}

func (m *mockPipeline) Status() (int, float64, time.Time, string) {
	return m.pid, m.uptime, m.lastCaptureTime, m.lastWindowTitle
}

func TestHandleStatus_subscribers(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{pid: 123, uptime: 60.0}

	subscribers := func() int { return 42 }
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), subscribers)

	request := ipc.Request{Command: ipc.CommandStatus}
	response, err := handler.Handle(nil, nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if !response.OK {
		t.Fatalf("expected OK response, got error: %s", response.Error)
	}

	var payload ipc.StatusPayload
	if err := json.Unmarshal(response.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if payload.Subscribers != 42 {
		t.Errorf("expected Subscribers 42, got %d", payload.Subscribers)
	}
	if payload.PID != 123 {
		t.Errorf("expected PID 123, got %d", payload.PID)
	}
}

func TestHandleStatus_zeroSubscribers(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{pid: 1}

	subscribers := func() int { return 0 }
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), subscribers)

	request := ipc.Request{Command: ipc.CommandStatus}
	response, err := handler.Handle(nil, nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	var payload ipc.StatusPayload
	if err := json.Unmarshal(response.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if payload.Subscribers != 0 {
		t.Errorf("expected Subscribers 0, got %d", payload.Subscribers)
	}
}
