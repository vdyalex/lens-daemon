package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

type mockMessages struct {
	result *anthropic.Message
	err    error
}

func (m *mockMessages) New(_ context.Context, _ anthropic.MessageNewParams, _ ...option.RequestOption) (*anthropic.Message, error) {
	return m.result, m.err
}

func TestProcess_emptyInput(t *testing.T) {
	m := &mockMessages{result: &anthropic.Message{}}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024, mocks.NopLogger())

	response, err := agent.Process(context.Background(), "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if response.Short != "" || response.Detailed.Answer != "" {
		t.Errorf("expected empty response, got short=%q answer=%q", response.Short, response.Detailed.Answer)
	}
}

func TestProcess_apiError(t *testing.T) {
	apiErr := errors.New("api failure")
	m := &mockMessages{err: apiErr}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024, mocks.NopLogger())

	response, err := agent.Process(context.Background(), "hello")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if response.Short != "" || response.Detailed.Answer != "" {
		t.Errorf("expected empty response on error, got short=%q answer=%q", response.Short, response.Detailed.Answer)
	}
}

func TestProcess_noToolUseBlock(t *testing.T) {
	msg := &anthropic.Message{}
	m := &mockMessages{result: msg}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024, mocks.NopLogger())

	_, err := agent.Process(context.Background(), "input")

	if err == nil {
		t.Errorf("expected error when no tool_use block, got nil")
	}
}

// buildToolUseMessage creates an anthropic.Message with a tool_use content block
// containing the given Response as its JSON input.
func buildToolUseMessage(response ai.Response) *anthropic.Message {
	input, _ := json.Marshal(response)
	raw := `{"content":[{"type":"tool_use","id":"test","name":"answer","input":` + string(input) + `}],"id":"msg_test","model":"claude-test","role":"assistant","stop_reason":"tool_use","type":"message","usage":{"input_tokens":10,"output_tokens":20}}`
	var message anthropic.Message
	_ = json.Unmarshal([]byte(raw), &message)
	return &message
}

func TestProcess_validToolUseBlock(t *testing.T) {
	expected := ai.Response{
		Short:    "B",
		Detailed: ai.ResponseDetail{Answer: "B", Reason: "Because X and Y."},
	}
	m := &mockMessages{result: buildToolUseMessage(expected)}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024, mocks.NopLogger())

	response, err := agent.Process(context.Background(), "input")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if response.Short != expected.Short {
		t.Errorf("expected short %q, got %q", expected.Short, response.Short)
	}
	if response.Detailed.Answer != expected.Detailed.Answer {
		t.Errorf("expected detailed answer %q, got %q", expected.Detailed.Answer, response.Detailed.Answer)
	}
	if response.Detailed.Reason != expected.Detailed.Reason {
		t.Errorf("expected detailed reason %q, got %q", expected.Detailed.Reason, response.Detailed.Reason)
	}
}
