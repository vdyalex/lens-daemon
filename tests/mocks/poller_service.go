package mocks

import "context"

// MockPollerService satisfies poller.Service interface for testing.
type MockPollerService struct {
	RunFunc func(ctx context.Context)
}

// Run implements poller.Service.
func (m *MockPollerService) Run(ctx context.Context) {
	if m.RunFunc != nil {
		m.RunFunc(ctx)
	} else {
		// Default: just return, letting context handle cancellation
		<-ctx.Done()
	}
}
