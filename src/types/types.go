package types

import (
	"context"
	"net/http"
)

// Agent interface types

// AgentProcessor is the interface for AI text processing adapters.
type AgentProcessor interface {
	Process(ctx context.Context, text string) (string, error)
}

// Messenger interface types

// MessengerBroadcaster is the interface for message broadcasting adapters.
type MessengerBroadcaster interface {
	Broadcast(ctx context.Context, text string) error
}

// Messenger/Telegram API types

// MessengerHTTPClient is the interface satisfied by *http.Client. It is shared across
// messenger.Sender and poller.Poller to abstract Telegram HTTP interactions,
// enabling dependency injection for testing.
type MessengerHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MessengerRequest is the Telegram sendMessage API request payload.
type MessengerRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// MessengerResponse is the Telegram API response payload.
type MessengerResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// MessengerUpdate represents a Telegram update from getUpdates.
type MessengerUpdate struct {
	UpdateID int64             `json:"update_id"`
	Message  *MessengerMessage `json:"message"`
}

// MessengerMessage represents a Telegram message.
type MessengerMessage struct {
	Chat MessengerChat `json:"chat"`
	Text string        `json:"text"`
}

// MessengerChat represents a Telegram chat.
type MessengerChat struct {
	ID int64 `json:"id"`
}
