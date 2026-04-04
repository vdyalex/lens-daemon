//go:generate mockgen -destination ../../tests/mocks/mock_ai_processor.go -package mocks . Processor

// Package ai provides integration with the Anthropic Claude API for text processing.
package ai

import (
	"context"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// MessagesService abstracts the Anthropic messages API for testability.
type MessagesService interface {
	New(ctx context.Context, params anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error)
}

// Response holds the structured AI response with short and detailed branches.
// Short is displayed on the teleprompter overlay; Detailed is broadcast to Telegram.
type Response struct {
	Short    string         `json:"short"`
	Detailed ResponseDetail `json:"detailed"`
}

// ResponseDetail holds the detailed answer with the raw answer and markdown explanation.
type ResponseDetail struct {
	Answer string `json:"answer"`
	Reason string `json:"reason"`
}

// Processor is the interface for AI text processing adapters.
type Processor interface {
	Process(ctx context.Context, text string) (Response, error)
}

// AI communicates with Claude AI to process extracted screen text.
type AI struct {
	messages          MessagesService
	model             string
	prompt            string
	maxResponseTokens int
	logger            *slog.Logger
}
