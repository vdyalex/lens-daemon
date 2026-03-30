package ipc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vdyalex/lens-daemon/src/ipc"
)

func TestLogBroker_broadcastsToSubscribers(t *testing.T) {
	broker := ipc.NewLogBroker()

	id1, ch1 := broker.Subscribe()
	id2, ch2 := broker.Subscribe()
	defer broker.Unsubscribe(id1)
	defer broker.Unsubscribe(id2)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	line := []byte("time=2026-03-17T12:00:00Z level=INFO msg=\"test\"")
	if _, err := broker.Write(line); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Both subscribers should receive the event (channels are buffered, so reads are immediate)
	select {
	case event := <-ch1:
		if event.Message != "test" {
			t.Errorf("ch1: expected message 'test', got %q", event.Message)
		}
	case <-ctx.Done():
		t.Error("ch1: timeout waiting for event")
	}

	select {
	case event := <-ch2:
		if event.Message != "test" {
			t.Errorf("ch2: expected message 'test', got %q", event.Message)
		}
	case <-ctx.Done():
		t.Error("ch2: timeout waiting for event")
	}
}

func TestLogBroker_dropsSlowSubscriber(t *testing.T) {
	broker := ipc.NewLogBroker()

	id, _ := broker.Subscribe()
	defer broker.Unsubscribe(id)

	// Fill the channel (buffered to 64)
	line := []byte("level=INFO msg=\"test\"")
	for i := 0; i < 64; i++ {
		broker.Write(line)
	}

	// Next write should not block (drops slow subscriber)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		broker.Write(line)
		close(done)
	}()

	select {
	case <-done:
		// Expected: write returned immediately without blocking
	case <-ctx.Done():
		t.Fatal("Write() blocked on slow subscriber (expected non-blocking drop)")
	}
}

func TestLogBroker_unsubscribeStopsDelivery(t *testing.T) {
	broker := ipc.NewLogBroker()

	id, _ := broker.Subscribe()
	broker.Unsubscribe(id)

	// Write should not panic or deadlock
	line := []byte("level=INFO msg=\"test\"")
	if _, err := broker.Write(line); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func TestLogBroker_replayRingBufferOnSubscribe(t *testing.T) {
	broker := ipc.NewLogBroker()

	// Write 5 events before subscribing
	for i := range 5 {
		line := []byte(fmt.Sprintf("time=2026-03-17T12:00:00Z level=INFO msg=\"event %d\"", i))
		if _, err := broker.Write(line); err != nil {
			t.Fatalf("Write() error: %v", err)
		}
	}

	// Subscribe after events are written — should replay all 5
	id, ch := broker.Subscribe()
	defer broker.Unsubscribe(id)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	for i := range 5 {
		select {
		case event := <-ch:
			expected := fmt.Sprintf("event %d", i)
			if event.Message != expected {
				t.Errorf("replayed event %d: got %q, want %q", i, event.Message, expected)
			}
		case <-ctx.Done():
			t.Fatalf("timeout waiting for replayed event %d", i)
		}
	}

	// Write one more live event — should arrive after replays
	if _, err := broker.Write([]byte("time=2026-03-17T12:00:01Z level=INFO msg=\"live\"")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	select {
	case event := <-ch:
		if event.Message != "live" {
			t.Errorf("live event: got %q, want %q", event.Message, "live")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for live event")
	}
}

func TestLogBroker_ringBufferOverflow(t *testing.T) {
	broker := ipc.NewLogBroker()

	// Write 150 events (exceeds ring buffer of 100)
	for i := range 150 {
		line := []byte(fmt.Sprintf("level=INFO msg=\"event %d\"", i))
		if _, err := broker.Write(line); err != nil {
			t.Fatalf("Write() error: %v", err)
		}
	}

	// Subscribe — should receive only the last 100
	id, ch := broker.Subscribe()
	defer broker.Unsubscribe(id)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// First replayed event should be event 50 (150 - 100)
	select {
	case event := <-ch:
		if event.Message != "event 50" {
			t.Errorf("first replayed event: got %q, want %q", event.Message, "event 50")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for first replayed event")
	}

	// Drain remaining 99 replayed events
	for range 99 {
		select {
		case <-ch:
		case <-ctx.Done():
			t.Fatal("timeout draining replayed events")
		}
	}
}

func TestLogBroker_concurrentSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	broker := ipc.NewLogBroker()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			id, _ := broker.Subscribe()
			// Simulate some work by writing to the channel
			broker.Unsubscribe(id)
		}()
	}

	// Wait for all goroutines with a timeout guard
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("timeout waiting for goroutines")
		}
	}

	// Broker should have no subscribers
	if count := broker.Count(); count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}
}
