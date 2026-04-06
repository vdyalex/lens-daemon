package ipc_test

import (
	"context"
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
	outputMethod    string
}

func (m *mockPipeline) Status() (int, float64, time.Time, string) {
	return m.pid, m.uptime, m.lastCaptureTime, m.lastWindowTitle
}

func (m *mockPipeline) SetOutputMethod(method string) { m.outputMethod = method }
func (m *mockPipeline) OutputMethod() string          { return m.outputMethod }

func TestHandleStatus_subscribers(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{pid: 123, uptime: 60.0, outputMethod: "teleprompter"}

	subscribers := func() int { return 42 }
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), subscribers)

	request := ipc.Request{Command: ipc.CommandStatus}
	response, err := handler.Handle(context.TODO(), nil, request)
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
	pipeline := &mockPipeline{pid: 1, outputMethod: "teleprompter"}

	subscribers := func() int { return 0 }
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), subscribers)

	request := ipc.Request{Command: ipc.CommandStatus}
	response, err := handler.Handle(context.TODO(), nil, request)
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

func TestHandleSet_outputMethodTelegram(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{outputMethod: "teleprompter"}
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), func() int { return 0 })

	payload, _ := json.Marshal(ipc.SetPayload{Key: "output-method", Value: "telegram"})
	request := ipc.Request{Command: ipc.CommandSet, Payload: payload}
	response, err := handler.Handle(context.TODO(), nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if !response.OK {
		t.Fatalf("expected OK response, got error: %s", response.Error)
	}
	if pipeline.outputMethod != "telegram" {
		t.Errorf("expected output method telegram, got %s", pipeline.outputMethod)
	}
}

func TestHandleSet_outputMethodTeleprompter(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{outputMethod: "telegram"}
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), func() int { return 0 })

	payload, _ := json.Marshal(ipc.SetPayload{Key: "output-method", Value: "teleprompter"})
	request := ipc.Request{Command: ipc.CommandSet, Payload: payload}
	response, err := handler.Handle(context.TODO(), nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if !response.OK {
		t.Fatalf("expected OK response, got error: %s", response.Error)
	}
	if pipeline.outputMethod != "teleprompter" {
		t.Errorf("expected output method teleprompter, got %s", pipeline.outputMethod)
	}
}

func TestHandleSet_invalidOutputMethod(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{outputMethod: "teleprompter"}
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), func() int { return 0 })

	payload, _ := json.Marshal(ipc.SetPayload{Key: "output-method", Value: "invalid"})
	request := ipc.Request{Command: ipc.CommandSet, Payload: payload}
	response, err := handler.Handle(context.TODO(), nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if response.OK {
		t.Error("expected error response for invalid output method")
	}
}

func TestHandleSet_unknownKey(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{}
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), func() int { return 0 })

	payload, _ := json.Marshal(ipc.SetPayload{Key: "unknown", Value: "value"})
	request := ipc.Request{Command: ipc.CommandSet, Payload: payload}
	response, err := handler.Handle(context.TODO(), nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if response.OK {
		t.Error("expected error response for unknown key")
	}
}

func TestHandleSet_invalidPayload(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{}
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), func() int { return 0 })

	request := ipc.Request{Command: ipc.CommandSet, Payload: json.RawMessage(`invalid`)}
	response, err := handler.Handle(context.TODO(), nil, request)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if response.OK {
		t.Error("expected error response for invalid payload")
	}
}

func TestHandleStatus_outputMethod(t *testing.T) {
	broker := ipc.NewLogBroker()
	pipeline := &mockPipeline{pid: 1, outputMethod: "telegram"}
	handler := ipc.NewCommandHandler(pipeline, broker, func() {}, mocks.NopLogger(), func() int { return 0 })

	request := ipc.Request{Command: ipc.CommandStatus}
	response, _ := handler.Handle(context.TODO(), nil, request)

	var payload ipc.StatusPayload
	_ = json.Unmarshal(response.Payload, &payload)

	if payload.OutputMethod != "telegram" {
		t.Errorf("expected output method telegram, got %s", payload.OutputMethod)
	}
}
