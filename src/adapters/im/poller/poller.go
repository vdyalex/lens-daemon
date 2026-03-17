// Package poller provides long-polling functionality for receiving Telegram messages.
package poller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// NewWithClient creates a Poller with a custom HTTP client.
// This is primarily used for testing with mock HTTP clients.
func NewWithClient(token string, subscriberStore im.Store, client im.HTTPClient, logger *slog.Logger, longPollTimeout, pollerTimeout time.Duration) *Poller {
	return &Poller{
		token:           token,
		store:           subscriberStore,
		client:          client,
		logger:          logger,
		offset:          0,
		longPollTimeout: longPollTimeout,
		pollerTimeout:   pollerTimeout,
	}
}

// New creates a Poller. The offset starts at 0 and is advanced as updates are processed.
func New(token string, subscriberStore im.Store, logger *slog.Logger, longPollTimeout, pollerTimeout, httpClientTimeout time.Duration) *Poller {
	return NewWithClient(token, subscriberStore, &http.Client{Timeout: httpClientTimeout}, logger, longPollTimeout, pollerTimeout)
}

// Run polls for updates indefinitely until ctx is cancelled.
// Expected deadline/canceled errors from the polling timeout are not logged;
// other errors are logged as warnings and retried with a 5-second backoff.
func (p *Poller) Run(ctx context.Context) {
	for {
		if err := p.poll(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			// Our own pollerTimeout fired — expected for long-polling when no updates
			// arrive before the timeout. Resume immediately without logging.
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				continue
			}
			p.logger.Warn("poller error", "error", err)
			select {
			case <-time.After(constants.TimeoutTelegramRetryBackoff):
			case <-ctx.Done():
				return
			}
		}
	}
}

// poll fetches updates, processes them, and advances the offset.
func (p *Poller) poll(ctx context.Context) error {
	httpCtx, cancel := context.WithTimeout(ctx, p.pollerTimeout)
	defer cancel()

	parsed := url.URL{
		Scheme: "https",
		Host:   "api.telegram.org",
		Path:   fmt.Sprintf("/bot%s/getUpdates", p.token),
	}
	q := parsed.Query()
	q.Set("offset", fmt.Sprintf("%d", p.offset))
	q.Set("timeout", fmt.Sprintf("%.0f", p.longPollTimeout.Seconds()))
	parsed.RawQuery = q.Encode()

	request, err := http.NewRequestWithContext(httpCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return fmt.Errorf("create poll request: %w", err)
	}

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("telegram poll request: %w", err)
	}
	defer response.Body.Close()

	var result struct {
		OK          bool        `json:"ok"`
		Result      []im.Update `json:"result"`
		Description string      `json:"description"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode telegram response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("%w: %s", exceptions.ErrIMTelegramAPI, result.Description)
	}

	for _, update := range result.Result {
		if err := p.handleUpdate(update); err != nil {
			p.logger.Error("handle update error", "update_id", update.UpdateID, "error", err)
		}
		p.offset = update.UpdateID + 1
	}

	return nil
}

// handleUpdate processes a single Telegram update.
// /start adds the chat to the store; /stop removes it.
func (p *Poller) handleUpdate(update im.Update) error {
	if update.Message == nil || update.Message.Text == "" {
		return nil
	}
	text := strings.TrimSpace(update.Message.Text)
	chatID := update.Message.Chat.ID

	switch text {
	case "/start":
		return p.store.Add(chatID)
	case "/stop":
		return p.store.Remove(chatID)
	}
	return nil
}
