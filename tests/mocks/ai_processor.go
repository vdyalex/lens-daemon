package mocks

import "context"

// MockAIProcessor satisfies ai.Processor interface.
type MockAIProcessor struct {
	ProcessFunc func(ctx context.Context, text string) (string, error)
	Calls       []string
}

// Process implements ai.Processor.
func (m *MockAIProcessor) Process(ctx context.Context, text string) (string, error) {
	m.Calls = append(m.Calls, text)
	return m.ProcessFunc(ctx, text)
}
