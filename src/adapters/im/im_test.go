package im_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

func TestBroadcast_emptyText(t *testing.T) {
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP call should not be made for empty text")
			return nil, nil
		},
	}
	store := mocks.MockStore(t)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(mockClient.Calls) > 0 {
		t.Errorf("expected no HTTP calls, got %d", len(mockClient.Calls))
	}
}

func TestBroadcast_noSubscribers(t *testing.T) {
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP call should not be made with no subscribers")
			return nil, nil
		},
	}
	store := mocks.MockStore(t)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "hello")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(mockClient.Calls) > 0 {
		t.Errorf("expected no HTTP calls, got %d", len(mockClient.Calls))
	}
}

func TestBroadcast_singleSubscriber(t *testing.T) {
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if !bytes.Contains([]byte(req.URL.String()), []byte("token")) {
				t.Errorf("URL should contain token")
			}
			return mocks.NewJSONResponse(200, `{"ok":true}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(12345)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "hello")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}

func TestBroadcast_multipleSubscribers(t *testing.T) {
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return mocks.NewJSONResponse(200, `{"ok":true}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(111)
	store.Add(222)
	store.Add(333)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "hello")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 HTTP calls, got %d", callCount)
	}
}

func TestBroadcast_partialFailure(t *testing.T) {
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 2 {
				return mocks.NewJSONResponse(200, `{"ok":false,"description":"Error"}`), nil
			}
			return mocks.NewJSONResponse(200, `{"ok":true}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(111)
	store.Add(222)
	store.Add(333)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "hello")

	if err == nil {
		t.Errorf("expected error from partial failure, got nil")
	}
	if callCount != 3 {
		t.Errorf("expected 3 HTTP calls (all attempted despite failure), got %d", callCount)
	}
}

func TestBroadcast_chunksLargeMessage(t *testing.T) {
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return mocks.NewJSONResponse(200, `{"ok":true}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(111)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 50, 1)

	// Send a message larger than chunk size
	largeMsg := "a"
	for i := 0; i < 150; i++ {
		largeMsg += "a"
	}

	err := sender.Broadcast(context.Background(), largeMsg)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 HTTP calls for chunked message, got %d", callCount)
	}
}

func TestBroadcast_rateLimitRetry_contextCancel(t *testing.T) {
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			// Return 429 rate limit response
			return mocks.NewJSONResponse(http.StatusTooManyRequests,
				`{"ok":false,"description":"retry after 30"}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(12345)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled; the retry select will hit ctx.Done before time.After(30s)

	err := sender.Broadcast(ctx, "hello")

	if err == nil {
		t.Errorf("expected error from cancelled context during rate-limit retry")
	}
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call before context cancel, got %d", callCount)
	}
}

func TestBroadcast_rateLimitRetry_success(t *testing.T) {
	// This test exercises the 429 retry path; it sleeps ~1s (fallback retry delay).
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return mocks.NewJSONResponse(http.StatusTooManyRequests,
					`{"ok":false,"description":"rate limit"}`), nil
			}
			return mocks.NewJSONResponse(200, `{"ok":true}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(12345)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "hello")

	if err != nil {
		t.Errorf("expected no error after retry, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 HTTP calls (1 fail + 1 retry), got %d", callCount)
	}
}

func TestBroadcast_nonRateLimitError_noRetry(t *testing.T) {
	callCount := 0
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return mocks.NewJSONResponse(200, `{"ok":false,"description":"Bad Request"}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(12345)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 3)

	err := sender.Broadcast(context.Background(), "hello")

	if err == nil {
		t.Errorf("expected error from non-200 response, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 HTTP call (no retry on non-rate-limit error), got %d", callCount)
	}
}

func TestBroadcast_contextCancelled(t *testing.T) {
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// This will be called with a cancelled context
			return nil, context.Canceled
		},
	}
	store := mocks.MockStore(t)
	store.Add(111)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sender.Broadcast(ctx, "hello")

	if err == nil {
		t.Errorf("expected context cancelled error, got nil")
	}
}

func TestBroadcast_payloadJSON(t *testing.T) {
	var capturedRequest *http.Request
	mockClient := &mocks.MockIMHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedRequest = req
			return mocks.NewJSONResponse(200, `{"ok":true}`), nil
		},
	}
	store := mocks.MockStore(t)
	store.Add(12345)
	sender := im.NewWithClient("token", store, mockClient, mocks.NopLogger(), 100, 1)

	err := sender.Broadcast(context.Background(), "test message")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify request body contains expected JSON
	if capturedRequest == nil {
		t.Fatal("expected captured request, got nil")
	}

	body, _ := io.ReadAll(capturedRequest.Body)
	if !bytes.Contains(body, []byte("12345")) {
		t.Errorf("expected request body to contain chat_id, got %s", string(body))
	}
	if !bytes.Contains(body, []byte(constants.TelegramParseMode)) {
		t.Errorf("expected request body to contain parse_mode, got %s", string(body))
	}
}
