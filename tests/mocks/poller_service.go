package mocks

import "context"

// MockPollerService satisfies poller.Service interface for testing.
type MockPollerService struct {
	RunFunc func(ctx context.Context)
}

// Run implements poller.Service.
func (mock *MockPollerService) Run(ctx context.Context) {
	if mock.RunFunc != nil {
		mock.RunFunc(ctx)
	} else {
		// Default: just return, letting context handle cancellation
		<-ctx.Done()
	}
}
