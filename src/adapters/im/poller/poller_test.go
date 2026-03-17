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
		// First poll completed; store.Add has been called synchronously
		time.Sleep(10 * time.Millisecond) // Allow poller to finish processing
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
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
	done := make(chan struct{})
	once := &sync.Once{}
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
		// First poll completed; store.Remove has been called
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
	}

	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected empty store after /stop, got %v", all)
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

	url := capturedRequest.URL.String()
	if url == "" {
		t.Errorf("expected non-empty URL, got %q", url)
	}
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

	callCount := 0
	done := make(chan struct{})
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount > 1 {
				// Return empty result after first call
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
			defer close(done)
			return mocks.NewJSONResponse(200, string(body)), nil
		},
	}

	client := poller.NewWithClient("token", store, mockClient, mocks.NopLogger(), 30*time.Second, 35*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go client.Run(ctx)

	select {
	case <-done:
		// First poll completed; store.Add has been called
	case <-ctx.Done():
		t.Fatal("timed out waiting for poller to process the update")
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

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Just verify that the poller doesn't panic on malformed JSON
	// The error will be logged and the poller will retry
	go func() {
		client.Run(ctx)
		once.Do(func() {
			close(done)
		})
	}()

	select {
	case <-done:
		// Poller completed without panicking - test passes
	case <-time.After(1 * time.Second):
		t.Fatal("poller hung on malformed JSON")
	}
}

