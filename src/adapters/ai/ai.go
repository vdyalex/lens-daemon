package ai

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const textBlockType = "text"

// New creates a new Claude AI agent.
func New(apiKey, model, prompt string, maxResponseTokens int) *AI {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return NewWithMessages(&client.Messages, model, prompt, maxResponseTokens)
}

// NewWithMessages creates a Claude AI agent with an injectable messages service.
// This is primarily used for testing.
func NewWithMessages(messages MessagesService, model, prompt string, maxResponseTokens int) *AI {
	return &AI{
		messages:          messages,
		model:             model,
		prompt:            prompt,
		maxResponseTokens: maxResponseTokens,
	}
}

// Process sends the extracted text to Claude and returns the response.
func (ai *AI) Process(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	response, err := ai.messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(ai.model),
		MaxTokens: int64(ai.maxResponseTokens),
		System: []anthropic.TextBlockParam{
			{Text: ai.prompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(text),
			),
		},
	})
	if err != nil {
		return "", fmt.Errorf("Anthropic API: %w", err)
	}

	var result string
	for _, block := range response.Content {
		if block.Type == textBlockType {
			result += block.Text
		}
	}

	return result, nil
}
