package pipeline_test

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/vdyalex/lens-daemon/src/pipeline"
)

// captureHandler is a minimal slog.Handler that records all log records for inspection.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	h.records = append(h.records, r)
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *captureHandler) errorCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	count := 0
	for _, r := range h.records {
		if r.Level == slog.LevelError {
			count++
		}
	}
	return count
}

// TestRunAnalyseWorker_processesItemsConcurrently verifies that multiple queued items
// are processed in parallel, not serially. It uses atomic counters to track concurrent
// goroutines without sleeps.
func TestRunAnalyseWorker_processesItemsConcurrently(t *testing.T) {
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	// inFlight counts goroutines currently inside Extract().
	var inFlight atomic.Int32
	// peak records the maximum observed concurrent count.
	var peak atomic.Int32
	// release unblocks all waiting Extract() calls.
	release := make(chan struct{})
	// started signals that goroutines have entered Extract(); buffered to non-block senders.
	started := make(chan struct{}, 2)

	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		DoAndReturn(func(img *image.RGBA) (string, error) {
			current := inFlight.Add(1)
			defer inFlight.Add(-1)

			// Update peak if we've increased the concurrent count.
			for {
				p := peak.Load()
				if current > p {
					if peak.CompareAndSwap(p, current) {
						break
					}
				} else {
					break
				}
			}

			started <- struct{}{}
			<-release
			return "extracted text", nil
		}).
		Times(2)

	// Mock the AI and broadcast to succeed without blocking.
	testMocks.agent.EXPECT().
		Process(gomock.Any(), gomock.Any()).
		Return("response", nil).
		Times(2)

	testMocks.broadcaster.BroadcastFunc = func(_ context.Context, _ string) error {
		return nil
	}

	p := createTestPipeline(t, testMocks)

	ctx := context.Background()
	workerDone := make(chan struct{})

	go pipeline.RunAnalyseWorker(p, ctx, workerDone)

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	queue := pipeline.AnalyseQueue(p)
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "A"}
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "B"}
	close(queue)

	// Wait for both goroutines to reach the blocking point.
	<-started
	<-started

	// Both should be in-flight; peak must be >= 2.
	if peak.Load() < 2 {
		t.Errorf("expected at least 2 concurrent goroutines, peak was %d", peak.Load())
	}

	close(release)
	<-workerDone
}

// TestRunAnalyseWorker_drainsAllBeforeClosingDone verifies that workerDone is not
// closed until all in-flight goroutines have completed (the wg.Wait() guarantee).
func TestRunAnalyseWorker_drainsAllBeforeClosingDone(t *testing.T) {
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	var completed atomic.Int32
	release := make(chan struct{})

	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		DoAndReturn(func(img *image.RGBA) (string, error) {
			<-release
			completed.Add(1)
			return "extracted text", nil
		}).
		Times(3)

	// Mock the AI and broadcast to succeed.
	testMocks.agent.EXPECT().
		Process(gomock.Any(), gomock.Any()).
		Return("response", nil).
		Times(3)

	testMocks.broadcaster.BroadcastFunc = func(_ context.Context, _ string) error {
		return nil
	}

	p := createTestPipeline(t, testMocks)

	ctx := context.Background()
	workerDone := make(chan struct{})

	go pipeline.RunAnalyseWorker(p, ctx, workerDone)

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	queue := pipeline.AnalyseQueue(p)
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "A"}
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "B"}
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "C"}
	close(queue)

	// workerDone must not be closed while goroutines are still blocked.
	select {
	case <-workerDone:
		t.Fatal("workerDone closed before in-flight goroutines completed")
	default:
		// expected: not closed yet
	}

	close(release)
	<-workerDone

	if completed.Load() != 3 {
		t.Errorf("expected 3 completed, got %d", completed.Load())
	}
}

// TestRunAnalyseWorker_logsErrorOnFatalAnalyseFailure verifies that fatal errors
// from analyse() are logged, and the worker continues processing remaining items.
func TestRunAnalyseWorker_logsErrorOnFatalAnalyseFailure(t *testing.T) {
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	handler := &captureHandler{}
	logger := slog.New(handler)

	settings := createTestConfig()
	p := pipeline.NewWithDependencies(
		settings, logger,
		testMocks.capturer, testMocks.extractor, testMocks.agent,
		testMocks.broadcaster, testMocks.poller, testMocks.listener,
	)

	fatalErr := fmt.Errorf("ocr hardware failure")

	// First item fails with a fatal error.
	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("", fatalErr).
		Times(1)

	// Second item succeeds.
	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		Return("text", nil).
		Times(1)

	testMocks.agent.EXPECT().
		Process(gomock.Any(), "text").
		Return("response", nil).
		Times(1)

	testMocks.broadcaster.BroadcastFunc = func(_ context.Context, _ string) error {
		return nil
	}

	ctx := context.Background()
	workerDone := make(chan struct{})
	go pipeline.RunAnalyseWorker(p, ctx, workerDone)

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	queue := pipeline.AnalyseQueue(p)
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "fail"}
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "ok"}
	close(queue)

	<-workerDone

	if handler.errorCount() != 1 {
		t.Errorf("expected 1 error log, got %d", handler.errorCount())
	}
}

// TestRunAnalyseWorker_contextCancellationPropagates verifies that cancelling the
// context causes in-flight analyse() goroutines to exit cleanly, and workerDone closes.
func TestRunAnalyseWorker_contextCancellationPropagates(t *testing.T) {
	testMocks := createTestMocks(t)
	defer testMocks.ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	// Use a channel to signal when Extract has been entered, before blocking on context.
	started := make(chan struct{})

	// Block Extract() until ctx is cancelled.
	testMocks.extractor.EXPECT().
		Extract(gomock.Any()).
		DoAndReturn(func(img *image.RGBA) (string, error) {
			started <- struct{}{}
			<-ctx.Done()
			return "", ctx.Err()
		}).
		Times(1)

	p := createTestPipeline(t, testMocks)

	workerDone := make(chan struct{})
	go pipeline.RunAnalyseWorker(p, ctx, workerDone)

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	queue := pipeline.AnalyseQueue(p)
	queue <- pipeline.CaptureResult{Image: img, WindowTitle: "cancellable"}
	close(queue)

	// Wait for Extract to be called and blocked.
	<-started

	cancel()

	select {
	case <-workerDone:
		// expected: worker exits after context cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("workerDone not closed after context cancellation (timeout)")
	}
}
