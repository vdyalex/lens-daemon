// Package ai provides integration with the Anthropic Claude API for text processing.
package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const textBlockType = "text"

// New creates a new Claude AI agent.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(apiKey, model, prompt string, maxResponseTokens int, logger *slog.Logger) *AI {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return NewWithMessages(&client.Messages, model, prompt, maxResponseTokens, logger)
}

// NewWithMessages creates a Claude AI agent with an injectable messages service.
// This is primarily used for testing with mock implementations.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func NewWithMessages(messages MessagesService, model, prompt string, maxResponseTokens int, logger *slog.Logger) *AI {
	return &AI{
		messages:          messages,
		model:             model,
		prompt:            prompt,
		maxResponseTokens: maxResponseTokens,
		logger:            logger,
	}
}

// Process sends the extracted text to Claude and returns the response.
// Returns an empty string without calling the API when text is empty.
// Logs request parameters, response metadata (token usage, latency), and errors.
func (a *AI) Process(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	a.logger.Info("sending request to anthropic",
		slog.String("model", a.model),
		slog.Int("input_character_count", len(text)),
		slog.Int("max_response_tokens", a.maxResponseTokens),
	)

	start := time.Now()
	response, err := a.messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(a.maxResponseTokens),
		System: []anthropic.TextBlockParam{
			{Text: a.prompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(text),
			),
		},
	})
	if err != nil {
		a.logger.Error("anthropic api request failed",
			slog.String("model", a.model),
			slog.Duration("latency", time.Since(start)),
			"error", err,
		)
		return "", fmt.Errorf("anthropic api: %w", err)
	}

	a.logger.Info("anthropic api response received",
		slog.String("message_id", response.ID),
		slog.String("model", string(response.Model)),
		slog.String("stop_reason", string(response.StopReason)),
		slog.Int64("input_tokens", response.Usage.InputTokens),
		slog.Int64("output_tokens", response.Usage.OutputTokens),
		slog.Int64("cache_read_input_tokens", response.Usage.CacheReadInputTokens),
		slog.Int64("cache_creation_input_tokens", response.Usage.CacheCreationInputTokens),
		slog.Duration("latency", time.Since(start)),
	)

	var result string
	for _, block := range response.Content {
		if block.Type == textBlockType {
			result += block.Text
		}
	}

	a.logger.Debug("anthropic response content",
		slog.Int("content_blocks", len(response.Content)),
		slog.Int("response_character_count", len(result)),
	)

	return result, nil
}
