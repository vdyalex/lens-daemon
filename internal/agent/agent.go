package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Agent communicates with Claude AI to process extracted screen text.
type Agent struct {
	client       anthropic.Client
	model        string
	systemPrompt string
}

// New creates a new Claude AI agent.
func New(apiKey, model, systemPrompt string) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Agent{
		client:       client,
		model:        model,
		systemPrompt: systemPrompt,
	}
}

// Process sends the extracted text to Claude and returns the response.
func (a *Agent) Process(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: a.systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(text),
			),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API: %w", err)
	}

	var result string
	for _, block := range resp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}
