package im

import (
	"context"
	"log/slog"
	"net/http"
)

// Broadcaster is the interface for message broadcasting adapters.
type Broadcaster interface {
	Broadcast(ctx context.Context, text string) error
}

// HTTPClient is the interface satisfied by *http.Client. It is shared across
// im.Sender and poller.Poller to abstract Telegram HTTP interactions,
// enabling dependency injection for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store is the interface for managing subscribers.
// Implementations must be safe for concurrent use.
type Store interface {
	// Add registers a subscriber. Idempotent. May persist to disk.
	Add(chatID int64) error
	// Remove unregisters a subscriber. Idempotent. May persist to disk.
	Remove(chatID int64) error
	// All returns all current subscriber chat IDs.
	All() []int64
}

// Request is the Telegram sendMessage API request payload.
type Request struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// Response is the Telegram API response payload.
type Response struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// Update represents a Telegram update from getUpdates.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message represents a Telegram message.
type Message struct {
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID int64 `json:"id"`
}

// Sender broadcasts messages to all subscribers via the Telegram Bot API.
type Sender struct {
	token      string
	store      Store
	client     HTTPClient
	logger     *slog.Logger
	chunkSize  int
	maxRetries int
}
