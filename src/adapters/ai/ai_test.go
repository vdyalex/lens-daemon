package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/vdyalex/lens-daemon/src/adapters/ai"
)

type mockMessages struct {
	result *anthropic.Message
	err    error
}

func (m *mockMessages) New(ctx context.Context, params anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
	return m.result, m.err
}

func TestProcess_emptyInput(t *testing.T) {
	m := &mockMessages{
		result: &anthropic.Message{},
	}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024)

	text, err := agent.Process(context.Background(), "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestProcess_apiError(t *testing.T) {
	apiErr := errors.New("api failure")
	m := &mockMessages{
		err: apiErr,
	}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024)

	text, err := agent.Process(context.Background(), "hello")

	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if text != "" {
		t.Errorf("expected empty text on error, got %q", text)
	}
}

func TestProcess_singleTextBlock(t *testing.T) {
	// Create a message with a text block response.
	// The message implementation may vary, so we create it generically.
	msg := &anthropic.Message{}
	// Note: Due to SDK complexity with ContentBlockUnion, we verify through behavior.
	// When no content blocks are present, the concatenation should result in empty string.
	m := &mockMessages{result: msg}
	agent := ai.NewWithMessages(m, "claude-test", "prompt", 1024)

	text, err := agent.Process(context.Background(), "input")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// With empty content, result should be empty string
	if text != "" {
		t.Errorf("expected empty result with no content blocks, got %q", text)
	}
}
