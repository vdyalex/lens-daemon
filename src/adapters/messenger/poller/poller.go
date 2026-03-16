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
	"github.com/vdyalex/lens-daemon/src/adapters/messenger/types"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// Poller long-polls Telegram's getUpdates API and dispatches /start and /stop commands
// to the subscriber store. It runs in a background goroutine.
type Poller struct {
	token  string
	store  *subscriber.Store
	client types.HTTPClient
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
func (poller *Poller) Run(ctx context.Context) {
	for {
		if err := poller.poll(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			poller.logger.Warn("Poller error", "error", err)
			select {
			case <-time.After(constants.TimeoutTelegramRetryBackoff):
			case <-ctx.Done():
				return
			}
		}
	}
}

// poll fetches updates, processes them, and advances the offset.
func (poller *Poller) poll(ctx context.Context) error {
	httpCtx, cancel := context.WithTimeout(ctx, constants.TimeoutTelegramPoller)
	defer cancel()

	u := url.URL{
		Scheme: "https",
		Host:   "api.telegram.org",
		Path:   fmt.Sprintf("/bot%s/getUpdates", poller.token),
	}
	q := u.Query()
	q.Set("offset", fmt.Sprintf("%d", poller.offset))
	q.Set("timeout", fmt.Sprintf("%.0f", constants.TimeoutTelegramLongPoll.Seconds()))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(httpCtx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := poller.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool          `json:"ok"`
		Result []types.Update `json:"result"`
		Desc   string        `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram error: %s", result.Desc)
	}

	for _, update := range result.Result {
		if err := poller.handleUpdate(update); err != nil {
			poller.logger.Error("Handle update error", "update_id", update.UpdateID, "error", err)
		}
		poller.offset = update.UpdateID + 1
	}

	return nil
}

// handleUpdate processes a single Telegram update.
// /start adds the chat to the store; /stop removes it.
func (poller *Poller) handleUpdate(u types.Update) error {
	if u.Message == nil || u.Message.Text == "" {
		return nil
	}
	text := strings.TrimSpace(u.Message.Text)
	chatID := u.Message.Chat.ID

	switch text {
	case "/start":
		return poller.store.Add(chatID)
	case "/stop":
		return poller.store.Remove(chatID)
	}
	return nil
}
