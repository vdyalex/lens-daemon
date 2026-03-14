package messenger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const telegramAPI = "https://api.telegram.org"

// Sender sends messages to a Telegram chat via the Bot API.
type Sender struct {
	token  string
	chatID int64
	client *http.Client
}

// New creates a new Telegram message sender.
func New(botToken string, chatID int64) *Sender {
	return &Sender{
		token:  botToken,
		chatID: chatID,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type Request struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type Response struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// Send sends a text message to the configured Telegram chat.
// Long messages are automatically split into chunks of 4096 runes
// (Telegram's message limit), preserving Unicode character boundaries.
func (sender *Sender) Send(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	const max = 4096
	runes := []rune(text)
	for len(runes) > 0 {
		end := max
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[:end])
		runes = runes[end:]

		if err := sender.sendChunk(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (sender *Sender) sendChunk(ctx context.Context, text string) error {
	payload := Request{
		ChatID: sender.chatID,
		Text:   text,
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

	var telegram Response
	if err := json.NewDecoder(response.Body).Decode(&telegram); err != nil {
		return fmt.Errorf("Decode telegram response: %w", err)
	}

	if !telegram.OK {
		return fmt.Errorf("Telegram API error: %s", telegram.Description)
	}

	return nil
}
