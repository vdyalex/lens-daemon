package mocks

import "context"

// MockAIProcessor satisfies ai.Processor interface.
type MockAIProcessor struct {
	ProcessFunc func(ctx context.Context, text string) (string, error)
	Calls       []string
}

// Process implements ai.Processor.
func (mock *MockAIProcessor) Process(ctx context.Context, text string) (string, error) {
	mock.Calls = append(mock.Calls, text)
	return mock.ProcessFunc(ctx, text)
}
