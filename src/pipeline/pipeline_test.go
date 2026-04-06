package pipeline_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/vdyalex/lens-daemon/src/pipeline"
	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

// Helper functions

func createTestConfig() *config.Config {
	return &config.Config{
		TimeoutForegroundWindow:  5 * time.Second,
		TimeoutCapture:           30 * time.Second,
		TimeoutOCRExtract:        30 * time.Second,
		TimeoutAIProcess:         60 * time.Second,
		TelegramBroadcastTimeout: 30 * time.Second,
		TimeoutPipelineOverall:   5 * time.Minute,
		TimeoutCapturePhase:      40 * time.Second,
		TimeoutAnalysePhase:      5 * time.Minute,
		AnalyseQueueCapacity:     16,
		OutputMethod:             "teleprompter",
	}
}

// testMocks holds all mock services for easier setup
type testMocks struct {
	ctrl         *gomock.Controller
	capturer     *mocks.MockCapturerService
	extractor    *mocks.MockExtractorService
	agent        *mocks.MockProcessor
	broadcaster  *mocks.MockIMBroadcaster
	poller       *mocks.MockPollerService
	listener     *mocks.MockListenerService
	teleprompter *mocks.MockTeleprompterService
}

func createTestMocks(t *testing.T) *testMocks {
	ctrl := gomock.NewController(t)
	return &testMocks{
		ctrl:         ctrl,
		capturer:     mocks.NewMockCapturerService(ctrl),
		extractor:    mocks.NewMockExtractorService(ctrl),
		agent:        mocks.NewMockProcessor(ctrl),
		broadcaster:  &mocks.MockIMBroadcaster{},
		poller:       mocks.NewMockPollerService(ctrl),
		listener:     mocks.NewMockListenerService(ctrl),
		teleprompter: mocks.NewMockTeleprompterService(ctrl),
	}
}

func createTestPipeline(_ *testing.T, testMocks *testMocks) *pipeline.Pipeline {
	settings := createTestConfig()
	return pipeline.NewBuilder(settings, mocks.NopLogger()).
		WithCapturer(testMocks.capturer).
		WithExtractor(testMocks.extractor).
		WithAgent(testMocks.agent).
		WithBroadcaster(testMocks.broadcaster).
		WithPoller(testMocks.poller).
		WithListener(testMocks.listener).
		WithTeleprompter(testMocks.teleprompter).
		Build()
}

func createTestPipelineWithSettings(_ *testing.T, settings *config.Config, logger *slog.Logger, testMocks *testMocks) *pipeline.Pipeline {
	return pipeline.NewBuilder(settings, logger).
		WithCapturer(testMocks.capturer).
		WithExtractor(testMocks.extractor).
		WithAgent(testMocks.agent).
		WithBroadcaster(testMocks.broadcaster).
		WithPoller(testMocks.poller).
		WithListener(testMocks.listener).
		WithTeleprompter(testMocks.teleprompter).
		Build()
}

func containsError(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
