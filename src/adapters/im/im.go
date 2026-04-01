// Package im provides Telegram broadcast capabilities for sending messages to subscribers.
package im

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/im/helpers"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// NewWithClient creates a new Telegram message broadcaster with a custom HTTP client.
// This is primarily used for testing with mock HTTP clients.
func NewWithClient(botToken string, store Store, client HTTPClient, logger *slog.Logger, chunkSize, maxRetries int) *Sender {
	return &Sender{
		token:      botToken,
		store:      store,
		client:     client,
		logger:     logger,
		chunkSize:  chunkSize,
		maxRetries: maxRetries,
	}
}

// New creates a new Telegram message broadcaster.
func New(botToken string, store Store, logger *slog.Logger, chunkSize, maxRetries int, httpClientTimeout time.Duration) *Sender {
	return NewWithClient(botToken, store, &http.Client{Timeout: httpClientTimeout}, logger, chunkSize, maxRetries)
}

// sendTo sends a message to a specific chat, handling chunking and rate limits.
func (s *Sender) sendTo(ctx context.Context, chatID int64, text string) error {
	max := s.chunkSize
	runes := []rune(text)
	for len(runes) > 0 {
		end := max
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[:end])
		runes = runes[end:]

		if err := s.sendChunk(ctx, chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}

// sendChunk sends a single chunk to a chat ID with rate-limit retry support.
func (s *Sender) sendChunk(ctx context.Context, chatID int64, text string) error {
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		err := s.doSendChunk(ctx, chatID, text)
		if err == nil {
			return nil
		}

		// Check for rate limit (429) and retry if applicable
		if isRateLimit(err) && attempt < s.maxRetries {
			retryAfter := parseRetryAfter(s.logger, err.Error())
			select {
			case <-time.After(retryAfter):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return err
	}
	return nil
}

// doSendChunk performs the actual HTTP call to Telegram.
func (s *Sender) doSendChunk(ctx context.Context, chatID int64, text string) error {
	payload := Request{
		ChatID:    chatID,
		Text:      helpers.ToTelegramMarkdown(text),
		ParseMode: constants.TelegramParseMode,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := constants.TelegramAPIBaseURL + fmt.Sprintf(constants.TelegramPathSendMessage, s.token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("telegram http request: %w", err)
	}
	defer response.Body.Close()

	var telegram Response
	if err := json.NewDecoder(response.Body).Decode(&telegram); err != nil {
		return fmt.Errorf("decode telegram response: %w", err)
	}

	if !telegram.OK {
		if response.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("%w: %s", exceptions.ErrIMRateLimit, telegram.Description)
		}
		return fmt.Errorf("%w: %s", exceptions.ErrIMTelegramAPI, telegram.Description)
	}

	return nil
}

// isRateLimit checks if an error is a Telegram rate-limit error.
func isRateLimit(err error) bool {
	return errors.Is(err, exceptions.ErrIMRateLimit)
}

// parseRetryAfter extracts the retry-after duration from a Telegram error message.
// Expected format: "Too Many Requests: retry after 23" (where 23 is seconds).
// If parsing fails, logs a warning and falls back to 1 second.
func parseRetryAfter(logger *slog.Logger, errMsg string) time.Duration {
	parts := strings.Fields(errMsg)
	if len(parts) > 0 {
		seconds, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	logger.Warn("failed to parse retry-after from telegram error; using default", "error_msg", errMsg)
	return constants.TimeoutTelegramRetryFallback
}

// Broadcast sends a text message to all subscribers.
// Long messages are automatically split into chunks of 4096 runes.
func (s *Sender) Broadcast(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	chatIDs := s.store.All()
	if len(chatIDs) == 0 {
		s.logger.Warn("no subscribers, skipping broadcast")
		return nil
	}

	var lastErr error
	for _, chatID := range chatIDs {
		if err := s.sendTo(ctx, chatID, text); err != nil {
			s.logger.Warn("broadcast send failed", "chat_id", chatID, "error", err)
			lastErr = err
		}
	}
	return lastErr
}
