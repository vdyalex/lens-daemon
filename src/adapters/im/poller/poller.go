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

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/store"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// NewWithClient creates a Poller with a custom HTTP client.
// This is primarily used for testing with mock HTTP clients.
func NewWithClient(token string, store *store.Store, client im.HTTPClient, logger *slog.Logger, longPollTimeout, pollerTimeout time.Duration) *Poller {
	return &Poller{
		token:           token,
		store:           store,
		client:          client,
		logger:          logger,
		offset:          0,
		longPollTimeout: longPollTimeout,
		pollerTimeout:   pollerTimeout,
	}
}

// New creates a Poller. The offset starts at 0 and is advanced as updates are processed.
func New(token string, store *store.Store, logger *slog.Logger, longPollTimeout, pollerTimeout, httpClientTimeout time.Duration) *Poller {
	return NewWithClient(token, store, &http.Client{Timeout: httpClientTimeout}, logger, longPollTimeout, pollerTimeout)
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
	httpCtx, cancel := context.WithTimeout(ctx, poller.pollerTimeout)
	defer cancel()

	parsed := url.URL{
		Scheme: "https",
		Host:   "api.telegram.org",
		Path:   fmt.Sprintf("/bot%s/getUpdates", poller.token),
	}
	q := parsed.Query()
	q.Set("offset", fmt.Sprintf("%d", poller.offset))
	q.Set("timeout", fmt.Sprintf("%.0f", poller.longPollTimeout.Seconds()))
	parsed.RawQuery = q.Encode()

	request, err := http.NewRequestWithContext(httpCtx, "GET", parsed.String(), nil)
	if err != nil {
		return err
	}

	response, err := poller.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	var result struct {
		OK          bool        `json:"ok"`
		Result      []im.Update `json:"result"`
		Description string      `json:"description"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("%w: %s", exceptions.IMTelegramAPIException, result.Description)
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
func (poller *Poller) handleUpdate(update im.Update) error {
	if update.Message == nil || update.Message.Text == "" {
		return nil
	}
	text := strings.TrimSpace(update.Message.Text)
	chatID := update.Message.Chat.ID

	switch text {
	case "/start":
		return poller.store.Add(chatID)
	case "/stop":
		return poller.store.Remove(chatID)
	}
	return nil
}
