package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		client: &http.Client{},
	}
}

type sendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// Send sends a text message to the configured Telegram chat.
// Long messages are automatically split into chunks of 4096 characters
// (Telegram's message limit).
func (s *Sender) Send(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	const maxLen = 4096
	for len(text) > 0 {
		chunk := text
		if len(chunk) > maxLen {
			chunk = text[:maxLen]
		}
		text = text[len(chunk):]

		if err := s.sendChunk(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sender) sendChunk(ctx context.Context, text string) error {
	payload := sendMessageRequest{
		ChatID: s.chatID,
		Text:   text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", telegramAPI, s.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram HTTP request: %w", err)
	}
	defer resp.Body.Close()

	var tgResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return fmt.Errorf("decode telegram response: %w", err)
	}

	if !tgResp.OK {
		return fmt.Errorf("telegram API error: %s", tgResp.Description)
	}

	return nil
}
