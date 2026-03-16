package messenger

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

	"github.com/vdyalex/lens-daemon/src/adapters/messenger/subscriber"
	"github.com/vdyalex/lens-daemon/src/adapters/messenger/types"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

const (
	telegramAPI = "https://api.telegram.org"
)

// Sender broadcasts messages to all subscribers via the Telegram Bot API.
type Sender struct {
	token      string
	store      *subscriber.Store
	client     types.HTTPClient
	logger     *slog.Logger
	chunkSize  int
	maxRetries int
}

// New creates a new Telegram message broadcaster.
func New(botToken string, store *subscriber.Store, logger *slog.Logger, chunkSize, maxRetries int) *Sender {
	return &Sender{
		token:      botToken,
		store:      store,
		client:     &http.Client{Timeout: constants.TimeoutTelegramHTTPClient},
		logger:     logger,
		chunkSize:  chunkSize,
		maxRetries: maxRetries,
	}
}

// Broadcast sends a text message to all subscribers.
// Long messages are automatically split into chunks of 4096 runes.
func (sender *Sender) Broadcast(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	chatIDs := sender.store.All()
	if len(chatIDs) == 0 {
		sender.logger.Warn("No subscribers, skipping broadcast")
		return nil
	}

	var lastErr error
	for _, chatID := range chatIDs {
		if err := sender.sendTo(ctx, chatID, text); err != nil {
			sender.logger.Error("Broadcast send failed", "chat_id", chatID, "error", err)
			lastErr = err
		}
	}
	return lastErr
}

// sendTo sends a message to a specific chat, handling chunking and rate limits.
func (sender *Sender) sendTo(ctx context.Context, chatID int64, text string) error {
	max := sender.chunkSize
	runes := []rune(text)
	for len(runes) > 0 {
		end := max
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[:end])
		runes = runes[end:]

		if err := sender.sendChunk(ctx, chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}

// sendChunk sends a single chunk to a chat ID with rate-limit retry support.
func (sender *Sender) sendChunk(ctx context.Context, chatID int64, text string) error {
	for attempt := 0; attempt <= sender.maxRetries; attempt++ {
		err := sender.doSendChunk(ctx, chatID, text)
		if err == nil {
			return nil
		}

		// Check for rate limit (429) and retry if applicable
		if isRateLimit(err) && attempt < sender.maxRetries {
			retryAfter := parseRetryAfter(sender.logger, err.Error())
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
func (sender *Sender) doSendChunk(ctx context.Context, chatID int64, text string) error {
	payload := types.Request{
		ChatID:    chatID,
		Text:      toTelegramMarkdown(text),
		ParseMode: constants.TelegramParseMode,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPI, sender.token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("Create telegram request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := sender.client.Do(request)
	if err != nil {
		return fmt.Errorf("Telegram HTTP request: %w", err)
	}
	defer response.Body.Close()

	var telegram types.Response
	if err := json.NewDecoder(response.Body).Decode(&telegram); err != nil {
		return fmt.Errorf("Decode telegram response: %w", err)
	}

	if !telegram.OK {
		if response.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("%w: %s", exceptions.MessengerRateLimitException, telegram.Description)
		}
		return fmt.Errorf("%w: %s", exceptions.MessengerTelegramAPIException, telegram.Description)
	}

	return nil
}

// isRateLimit checks if an error is a Telegram rate-limit error.
func isRateLimit(err error) bool {
	return errors.Is(err, exceptions.MessengerRateLimitException)
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
	logger.Warn("Failed to parse retry-after from Telegram error; using default", "error_msg", errMsg)
	return constants.TimeoutTelegramRetryFallback
}
