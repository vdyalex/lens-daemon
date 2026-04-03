package poller_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

func TestPoller_startCommand(t *testing.T) {
	store := mocks.MockStore(t)

	update := im.Update{
		UpdateID: 1,
		Message: &im.Message{
			Text: "/start",
			Chat: im.Chat{ID: 12345},
		},
	}

	done := make(chan struct{})
	once := &sync.Once{}
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK: true,
			}
			// First call returns the update; second call returns empty to stop polling
			if callCount == 1 {
				result.Result = []im.Update{update}
				once.Do(func() {
					close(done)
				})
			} else {
				// Return error on subsequent calls to trigger immediate context check
				return nil, context.Canceled
			}
			body, _ := json.Marshal(result)
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track when poller exits
	runDone := make(chan struct{})
	go func() {
		client.Run(ctx)
		close(runDone)
	}()

	select {
	case <-done:
		// First poll completed; now cancel to stop the poller
		cancel()
		// Wait for Run to exit (should be fast since next poll will error)
		exitCtx, exitCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer exitCancel()
		select {
		case <-runDone:
			// Poller exited - store operations are guaranteed complete
		case <-exitCtx.Done():
			t.Fatal("poller did not exit after cancel")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for first poll")
	}

	all := store.All()
	if len(all) != 1 || (len(all) > 0 && all[0] != 12345) {
		t.Errorf("expected chatID 12345 in store, got %v", all)
	}
}

func TestPoller_stopCommand(t *testing.T) {
	store := mocks.MockStore(t)
	store.Add(12345)

	update := im.Update{
		UpdateID: 1,
		Message: &im.Message{
			Text: "/stop",
			Chat: im.Chat{ID: 12345},
		},
	}

	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount > 1 {
				// Return empty result after first call to stop polling
				result := struct {
					OK     bool        `json:"ok"`
					Result []im.Update `json:"result"`
				}{
					OK:     true,
					Result: []im.Update{},
				}
				body, _ := json.Marshal(result)
				return mocks.NewJSONResponse(200, string(body)), nil
			}
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK:     true,
				Result: []im.Update{update},
			}
			body, _ := json.Marshal(result)
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	// Wait until the store is empty: the HTTP response is returned before
	// handleUpdate calls store.Remove, so we poll the store to avoid a race.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(store.All()) == 0 {
				return
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for store to become empty, got %v", store.All())
		}
	}
}

func TestPoller_unknownCommand(t *testing.T) {
	store := mocks.MockStore(t)

	update := im.Update{
		UpdateID: 1,
		Message: &im.Message{
			Text: "/help",
			Chat: im.Chat{ID: 12345},
		},
	}

	done := make(chan struct{})
	once := &sync.Once{}
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK:     true,
				Result: []im.Update{update},
			}
			body, _ := json.Marshal(result)
			once.Do(func() {
				close(done)
			})
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	select {
	case <-done:
		// First poll completed
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
	}

	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected store to remain empty for unknown command, got %v", all)
	}
}

func TestPoller_nilMessage(t *testing.T) {
	store := mocks.MockStore(t)

	update := im.Update{
		UpdateID: 1,
		Message:  nil,
	}

	done := make(chan struct{})
	once := &sync.Once{}
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK:     true,
				Result: []im.Update{update},
			}
			body, _ := json.Marshal(result)
			once.Do(func() {
				close(done)
			})
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Should not panic
	go client.Run(ctx)

	select {
	case <-done:
		// First poll completed without panic
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
	}
}

func TestPoller_emptyResult(t *testing.T) {
	store := mocks.MockStore(t)

	done := make(chan struct{})
	once := &sync.Once{}
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount > 1 {
				// Return empty result and error on subsequent calls to stop polling
				return nil, fmt.Errorf("test complete")
			}
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK:     true,
				Result: []im.Update{},
			}
			body, _ := json.Marshal(result)
			once.Do(func() {
				close(done)
			})
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	select {
	case <-done:
		// First poll completed
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
	}

	// Should handle empty result without error
	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected empty store, got %v", all)
	}
}

func TestPoller_apiError(t *testing.T) {
	store := mocks.MockStore(t)

	done := make(chan struct{})
	once := &sync.Once{}
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount > 1 {
				// Stop polling after first call
				return nil, fmt.Errorf("test complete")
			}
			result := struct {
				OK          bool   `json:"ok"`
				Description string `json:"description"`
			}{
				OK:          false,
				Description: "Unauthorized",
			}
			body, _ := json.Marshal(result)
			once.Do(func() {
				close(done)
			})
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	select {
	case <-done:
		// First poll completed with error handled
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
	}

	// Poller should handle API errors and continue (with backoff)
	// Just verify it doesn't crash
}

func TestPoller_requestURL(t *testing.T) {
	store := mocks.MockStore(t)

	var capturedRequest *http.Request
	done := make(chan struct{})
	once := &sync.Once{}
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedRequest = req
			callCount++
			if callCount > 1 {
				// Stop polling after first call
				return nil, fmt.Errorf("test complete")
			}
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK:     true,
				Result: []im.Update{},
			}
			body, _ := json.Marshal(result)
			once.Do(func() {
				close(done)
			})
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token123", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	select {
	case <-done:
		// First poll completed
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
	}

	if capturedRequest == nil {
		t.Fatal("expected captured request, got nil")
	}

	requestURL := capturedRequest.URL.String()
	if requestURL == "" {
		t.Errorf("expected non-empty URL, got %q", requestURL)
	}

	t.Run("includes allowed_updates parameter", func(t *testing.T) {
		allowedUpdates := capturedRequest.URL.Query().Get("allowed_updates")
		if allowedUpdates != `["message"]` {
			t.Errorf("expected allowed_updates=[\"message\"], got %q", allowedUpdates)
		}
	})
}

func TestPoller_whitespaceCommand(t *testing.T) {
	store := mocks.MockStore(t)

	update := im.Update{
		UpdateID: 1,
		Message: &im.Message{
			Text: "  /start  ",
			Chat: im.Chat{ID: 12345},
		},
	}

	done := make(chan struct{})
	once := &sync.Once{}
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK: true,
			}
			if callCount == 1 {
				result.Result = []im.Update{update}
				once.Do(func() {
					close(done)
				})
			} else {
				return nil, context.Canceled
			}
			body, _ := json.Marshal(result)
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		client.Run(ctx)
		close(runDone)
	}()

	select {
	case <-done:
		cancel()
		exitCtx, exitCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer exitCancel()
		select {
		case <-runDone:
		case <-exitCtx.Done():
			t.Fatal("poller did not exit after cancel")
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for first poll")
	}

	all := store.All()
	if len(all) != 1 || (len(all) > 0 && all[0] != 12345) {
		t.Errorf("expected chatID 12345 in store (whitespace trimmed), got %v", all)
	}
}

func TestPoller_deadlineExceeded(t *testing.T) {
	store := mocks.MockStore(t)

	callCount := 0
	done := make(chan struct{})
	once := &sync.Once{}
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call returns deadline exceeded (simulating long-poll timeout)
				return nil, context.DeadlineExceeded
			}
			// Second call succeeds; signal completion
			result := struct {
				OK     bool        `json:"ok"`
				Result []im.Update `json:"result"`
			}{
				OK:     true,
				Result: []im.Update{},
			}
			body, _ := json.Marshal(result)
			once.Do(func() {
				close(done)
			})
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	logger := mocks.NopLogger()
	client := poller.NewWithClient("token", store, mockClient, logger, 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	select {
	case <-done:
		// Second poll succeeded after deadline exceeded
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to resume after deadline exceeded")
	}

	// Poller should have made at least 2 calls (first deadline exceeded, then success)
	if callCount < 2 {
		t.Errorf("expected at least 2 calls, got %d", callCount)
	}
}

func TestPoller_malformedJSONResponse(t *testing.T) {
	store := mocks.MockStore(t)

	done := make(chan struct{})
	once := &sync.Once{}
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Return invalid JSON - this will cause a decode error
			// The poller should wrap this error and continue retrying
			return mocks.NewJSONResponse(200, "{invalid json"), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	pollerCtx, pollerCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer pollerCancel()

	// Just verify that the poller doesn't panic on malformed JSON
	// The error will be logged and the poller will retry
	go func() {
		client.Run(pollerCtx)
		once.Do(func() {
			close(done)
		})
	}()

	// Use a longer test timeout to allow poller goroutine to finish after context expires
	testCtx, testCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer testCancel()

	select {
	case <-done:
		// Poller completed without panicking - test passes
	case <-testCtx.Done():
		t.Fatal("poller hung on malformed JSON")
	}
}
