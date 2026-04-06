// Package ai provides integration with the Anthropic Claude API for text processing.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// toolName is the name of the structured output tool that Claude is forced to call.
const toolName = "answer"

// responseSchema defines the JSON schema for the structured response tool.
// Claude is forced to call this tool via tool_choice, producing a validated
// JSON object with "short" (raw answer), "detailed" (answer + reason), and
// "deterministic" (confidence flag) fields.
var responseSchema = anthropic.ToolInputSchemaParam{
	Properties: map[string]any{
		"deterministic": map[string]any{
			"type":        "boolean",
			"description": "Set true only when the answer is factually certain and unambiguous. Set false when providing incompatible content, caveats, disclaimers, or errors.",
		},
		"short": map[string]any{
			"type":        "string",
			"description": "The shortest distinguishing fragment of the correct option — just enough to identify it among the alternatives without reproducing the full text. No formatting or explanation.",
		},
		"detailed": map[string]any{
			"type":        "object",
			"description": "Full answer with the correct option and a concise explanation for broadcast.",
			"properties": map[string]any{
				"answer": map[string]any{
					"type":        "string",
					"description": "The correct option reproduced verbatim as it appears in the source text.",
				},
				"reason": map[string]any{
					"type":        "string",
					"description": "Compact markdown explanation of why this is the correct answer. Optimised for quick evaluation.",
				},
			},
			"required": []string{"answer", "reason"},
		},
	},
	Required: []string{"deterministic", "short", "detailed"},
}

// New creates a new Claude AI agent.
// cacheTTL controls the prompt caching TTL for system prompt and tool definitions
// (e.g. CacheControlEphemeralTTLTTL1h). Pass empty string to disable caching.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func New(apiKey, model, prompt string, maxResponseTokens int, cacheTTL anthropic.CacheControlEphemeralTTL, logger *slog.Logger) *AI {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return NewWithMessages(&client.Messages, model, prompt, maxResponseTokens, cacheTTL, logger)
}

// NewWithMessages creates a Claude AI agent with an injectable messages service.
// This is primarily used for testing with mock implementations.
// logger must not be nil; pass slog.Default() if no custom logger is required.
func NewWithMessages(messages MessagesService, model, prompt string, maxResponseTokens int, cacheTTL anthropic.CacheControlEphemeralTTL, logger *slog.Logger) *AI {
	return &AI{
		messages:          messages,
		model:             model,
		prompt:            prompt,
		maxResponseTokens: maxResponseTokens,
		cacheTTL:          cacheTTL,
		logger:            logger,
	}
}

// Process sends the extracted text to Claude and returns a structured Response
// with Deterministic, Short, and Detailed fields. Claude is forced to use the
// "answer" tool via tool_choice, guaranteeing structured JSON output.
// Deterministic indicates factual certainty; the pipeline uses it to gate
// teleprompter display — only certain answers are shown on screen.
//
// The system prompt and tool definition use ephemeral cache control so repeated
// calls within the TTL window read those tokens from cache (90% cheaper, faster
// time-to-first-token). Each call is independent — no conversation history.
//
// Returns an empty Response without calling the API when text is empty.
// Logs request parameters, response metadata (token usage, latency), and errors.
func (a *AI) Process(ctx context.Context, text string) (Response, error) {
	if text == "" {
		return Response{}, nil
	}

	a.logger.Info("sending request to anthropic",
		slog.String("model", a.model),
		slog.Int("input_character_count", len(text)),
		slog.Int("max_response_tokens", a.maxResponseTokens),
	)

	cacheControl := anthropic.NewCacheControlEphemeralParam()
	cacheControl.TTL = a.cacheTTL
	tool := anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:         toolName,
			Description:  anthropic.String("Return a short distinguishing fragment for the teleprompter, a detailed answer with explanation for broadcast, and a deterministic flag indicating factual certainty"),
			InputSchema:  responseSchema,
			CacheControl: cacheControl,
		},
	}

	start := time.Now()
	message, err := a.messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(a.maxResponseTokens),
		System: []anthropic.TextBlockParam{
			{
				Text:         a.prompt,
				CacheControl: cacheControl,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(text),
			),
		},
		Tools:      []anthropic.ToolUnionParam{tool},
		ToolChoice: anthropic.ToolChoiceParamOfTool(toolName),
	})
	if err != nil {
		a.logger.Error("anthropic api request failed",
			slog.String("model", a.model),
			slog.Duration("latency", time.Since(start)),
			"error", err,
		)
		return Response{}, fmt.Errorf("anthropic api: %w", err)
	}

	a.logger.Info("anthropic api response received",
		slog.String("message_id", message.ID),
		slog.String("model", string(message.Model)),
		slog.String("stop_reason", string(message.StopReason)),
		slog.Int64("input_tokens", message.Usage.InputTokens),
		slog.Int64("output_tokens", message.Usage.OutputTokens),
		slog.Int64("cache_read_input_tokens", message.Usage.CacheReadInputTokens),
		slog.Int64("cache_creation_input_tokens", message.Usage.CacheCreationInputTokens),
		slog.Duration("latency", time.Since(start)),
	)

	response, err := extractResponse(message)
	if err != nil {
		a.logger.Error("failed to extract structured response", "error", err)
		return Response{}, err
	}

	a.logger.Debug("anthropic response content",
		slog.Int("content_blocks", len(message.Content)),
		slog.Bool("deterministic", response.Deterministic),
		slog.String("short", response.Short),
		slog.String("detailed_answer", response.Detailed.Answer),
		slog.String("detailed_reason", response.Detailed.Reason),
		slog.Int("detailed_character_count", len(response.Detailed.Reason)),
	)

	return response, nil
}

// extractResponse finds the tool_use block for "answer" in the message content
// and unmarshals its input into a Response.
func extractResponse(message *anthropic.Message) (Response, error) {
	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
			if variant.Name != toolName {
				continue
			}
			var response Response
			if err := json.Unmarshal(variant.Input, &response); err != nil {
				return Response{}, fmt.Errorf("unmarshal tool input: %w", err)
			}
			return response, nil
		}
	}
	return Response{}, fmt.Errorf("no %q tool_use block in response", toolName)
}
