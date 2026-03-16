package types

import "net/http"

// HTTPClient is the interface satisfied by *http.Client. It is shared across
// messenger.Sender and poller.Poller to abstract Telegram HTTP interactions,
// enabling dependency injection for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
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
