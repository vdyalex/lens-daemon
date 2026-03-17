package ai

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// MessagesService abstracts the Anthropic messages API for testability.
type MessagesService interface {
	New(ctx context.Context, params anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error)
}

// Processor is the interface for AI text processing adapters.
type Processor interface {
	Process(ctx context.Context, text string) (string, error)
}

// AI communicates with Claude AI to process extracted screen text.
type AI struct {
	messages          MessagesService
	model             string
	prompt            string
	maxResponseTokens int
}
