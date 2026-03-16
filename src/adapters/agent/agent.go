package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// Agent communicates with Claude AI to process extracted screen text.
type Agent struct {
	client anthropic.Client
	model  string
	prompt string
}

// New creates a new Claude AI agent.
func New(apiKey, model, prompt string) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Agent{
		client: client,
		model:  model,
		prompt: prompt,
	}
}

// Process sends the extracted text to Claude and returns the response.
func (agent *Agent) Process(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	response, err := agent.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(agent.model),
		MaxTokens: int64(constants.ClaudeMaxResponseTokens),
		System: []anthropic.TextBlockParam{
			{Text: agent.prompt},
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
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}
