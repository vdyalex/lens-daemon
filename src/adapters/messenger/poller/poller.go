package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/messenger/subscriber"
)

// Poller long-polls Telegram's getUpdates API and dispatches /start and /stop commands
// to the subscriber store. It runs in a background goroutine.
type Poller struct {
	token  string
	store  *subscriber.Store
	client *http.Client
	logger *slog.Logger
	offset int64
}

// New creates a Poller. The offset starts at 0 and is advanced as updates are processed.
func New(token string, store *subscriber.Store, logger *slog.Logger) *Poller {
	return &Poller{
		token:  token,
		store:  store,
		client: &http.Client{},
		logger: logger,
		offset: 0,
	}
}

// Run polls for updates indefinitely until ctx is cancelled.
// It logs warnings on errors and retries with a 5-second backoff.
func (p *Poller) Run(ctx context.Context) {
	for {
		if err := p.poll(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			p.logger.Warn("Poller error", "error", err)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}
}

// poll fetches updates, processes them, and advances the offset.
func (p *Poller) poll(ctx context.Context) error {
	httpCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	u := url.URL{
		Scheme: "https",
		Host:   "api.telegram.org",
		Path:   fmt.Sprintf("/bot%s/getUpdates", p.token),
	}
	q := u.Query()
	q.Set("offset", fmt.Sprintf("%d", p.offset))
	q.Set("timeout", "30")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(httpCtx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool      `json:"ok"`
		Result []Update  `json:"result"`
		Desc   string    `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram error: %s", result.Desc)
	}

	for _, update := range result.Result {
		if err := p.handleUpdate(update); err != nil {
			p.logger.Error("Handle update error", "update_id", update.UpdateID, "error", err)
		}
		p.offset = update.UpdateID + 1
	}

	return nil
}

// handleUpdate processes a single Telegram update.
// /start adds the chat to the store; /stop removes it.
func (p *Poller) handleUpdate(u Update) error {
	if u.Message == nil || u.Message.Text == "" {
		return nil
	}
	text := strings.TrimSpace(u.Message.Text)
	chatID := u.Message.Chat.ID

	switch text {
	case "/start":
		return p.store.Add(chatID)
	case "/stop":
		return p.store.Remove(chatID)
	}
	return nil
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
