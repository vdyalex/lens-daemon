package ipc_test

import (
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
	case <-time.After(1 * time.Second):
		t.Error("ch1: timeout waiting for event")
	}

	select {
	case event := <-ch2:
		if event.Message != "test" {
			t.Errorf("ch2: expected message 'test', got %q", event.Message)
		}
	case <-time.After(1 * time.Second):
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
	done := make(chan struct{})
	go func() {
		broker.Write(line)
		close(done)
	}()

	select {
	case <-done:
		// Expected: write returned immediately without blocking
	case <-time.After(5 * time.Second):
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

func TestLogBroker_concurrentSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	broker := ipc.NewLogBroker()

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
	timeoutChan := time.After(5 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-timeoutChan:
			t.Fatal("timeout waiting for goroutines")
		}
	}

	// Broker should have no subscribers
	if count := broker.Count(); count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}
}
