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
	}
}

// testMocks holds all mock services for easier setup
type testMocks struct {
	ctrl        *gomock.Controller
	capturer    *mocks.MockCapturerService
	extractor   *mocks.MockExtractorService
	agent       *mocks.MockProcessor
	broadcaster *mocks.MockIMBroadcaster
	poller      *mocks.MockPollerService
}

func createTestMocks(t *testing.T) *testMocks {
	ctrl := gomock.NewController(t)
	return &testMocks{
		ctrl:        ctrl,
		capturer:    mocks.NewMockCapturerService(ctrl),
		extractor:   mocks.NewMockExtractorService(ctrl),
		agent:       mocks.NewMockProcessor(ctrl),
		broadcaster: &mocks.MockIMBroadcaster{},
		poller:      mocks.NewMockPollerService(ctrl),
	}
}

func createTestPipeline(_ *testing.T, testMocks *testMocks) *pipeline.Pipeline {
	settings := createTestConfig()
	logger := mocks.NopLogger()
	return pipeline.NewWithDependencies(settings, logger, testMocks.capturer, testMocks.extractor, testMocks.agent, testMocks.broadcaster, testMocks.poller)
}

func createTestPipelineWithSettings(_ *testing.T, settings *config.Config, logger *slog.Logger, testMocks *testMocks) *pipeline.Pipeline {
	return pipeline.NewWithDependencies(settings, logger, testMocks.capturer, testMocks.extractor, testMocks.agent, testMocks.broadcaster, testMocks.poller)
}

func containsError(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
