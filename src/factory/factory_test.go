// Package factory_test tests factory construction helpers.
package factory_test

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im"
	"github.com/vdyalex/lens-daemon/src/adapters/im/poller"
	"github.com/vdyalex/lens-daemon/src/factory"
	"github.com/vdyalex/lens-daemon/src/utils/config"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/tests/mocks"
)

// BuildStore

func TestBuildStore_noToken_returnsNil(t *testing.T) {
	settings := &config.Config{TelegramBotToken: ""}
	logger := mocks.NopLogger()

	store, err := factory.BuildStore(settings, logger)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store != nil {
		t.Errorf("expected nil store when bot token is empty, got %T", store)
	}
}

func TestBuildStore_withToken_returnsStore(t *testing.T) {
	tmpDir := t.TempDir()
	settings := &config.Config{
		TelegramBotToken:            "token123",
		TelegramSubscriberStorePath: tmpDir + "/subscribers",
	}
	logger := mocks.NopLogger()

	store, err := factory.BuildStore(settings, logger)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store when bot token is set")
	}
}

// BroadcasterFactory

func TestBroadcasterFactory_nilStore_returnsNoop(t *testing.T) {
	factory := factory.BroadcasterFactory{
		Settings: &config.Config{},
		Store:    nil,
		Logger:   mocks.NopLogger(),
	}

	broadcaster := factory.Build()

	if _, ok := broadcaster.(*im.NoopBroadcaster); !ok {
		t.Errorf("expected *im.NoopBroadcaster, got %T", broadcaster)
	}
}

func TestBroadcasterFactory_withStore_returnsSender(t *testing.T) {
	store := mocks.MockStore(t)
	factory := factory.BroadcasterFactory{
		Settings: &config.Config{
			TelegramBotToken:          "token123",
			TelegramMessageChunkSize:  constants.TelegramMessageChunkSize,
			TelegramMaxRetries:        constants.TelegramMaxRetries,
			TelegramHTTPClientTimeout: constants.TimeoutTelegramHTTPClient,
		},
		Store:  store,
		Logger: mocks.NopLogger(),
	}

	broadcaster := factory.Build()

	if broadcaster == nil {
		t.Fatal("expected non-nil broadcaster")
	}
	if _, ok := broadcaster.(*im.NoopBroadcaster); ok {
		t.Errorf("expected real broadcaster, got NoopBroadcaster")
	}
}

// PollerFactory

func TestPollerFactory_nilStore_returnsNoop(t *testing.T) {
	factory := factory.PollerFactory{
		Settings: &config.Config{},
		Store:    nil,
		Logger:   mocks.NopLogger(),
	}

	service := factory.Build()

	if _, ok := service.(*poller.NoopPoller); !ok {
		t.Errorf("expected *poller.NoopPoller, got %T", service)
	}
}

func TestPollerFactory_withStore_returnsPoller(t *testing.T) {
	store := mocks.MockStore(t)
	factory := factory.PollerFactory{
		Settings: &config.Config{
			TelegramBotToken:          "token123",
			OutputMethod:              constants.OutputMethodTeleprompter,
			TelegramLongPollTimeout:   constants.TimeoutTelegramLongPoll,
			TelegramPollerTimeout:     constants.TimeoutTelegramPoller,
			TelegramHTTPClientTimeout: constants.TimeoutTelegramHTTPClient,
		},
		Store:  store,
		Logger: mocks.NopLogger(),
	}

	service := factory.Build()

	if _, ok := service.(*poller.Poller); !ok {
		t.Errorf("expected *poller.Poller, got %T", service)
	}
}

func TestPollerFactory_telegramOutputMethod_activatesPoller(t *testing.T) {
	store := mocks.MockStore(t)
	telegramFactory := factory.PollerFactory{
		Settings: &config.Config{
			TelegramBotToken:          "token123",
			OutputMethod:              constants.OutputMethodTelegram,
			TelegramLongPollTimeout:   constants.TimeoutTelegramLongPoll,
			TelegramPollerTimeout:     constants.TimeoutTelegramPoller,
			TelegramHTTPClientTimeout: constants.TimeoutTelegramHTTPClient,
		},
		Store:  store,
		Logger: mocks.NopLogger(),
	}
	teleprompterFactory := factory.PollerFactory{
		Settings: &config.Config{
			TelegramBotToken:          "token123",
			OutputMethod:              constants.OutputMethodTeleprompter,
			TelegramLongPollTimeout:   constants.TimeoutTelegramLongPoll,
			TelegramPollerTimeout:     constants.TimeoutTelegramPoller,
			TelegramHTTPClientTimeout: constants.TimeoutTelegramHTTPClient,
		},
		Store:  store,
		Logger: mocks.NopLogger(),
	}

	t.Run("telegram output method returns active poller (non-noop)", func(t *testing.T) {
		service := telegramFactory.Build()
		if _, ok := service.(*poller.NoopPoller); ok {
			t.Error("expected real poller for telegram output method, got NoopPoller")
		}
	})

	t.Run("teleprompter output method returns inactive poller (non-noop)", func(t *testing.T) {
		service := teleprompterFactory.Build()
		if _, ok := service.(*poller.NoopPoller); ok {
			t.Error("expected real poller even for teleprompter output method, got NoopPoller")
		}
	})
}
