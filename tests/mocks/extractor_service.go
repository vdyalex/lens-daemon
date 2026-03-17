package mocks

import "image"

// MockExtractorService satisfies extractor.Service interface.
type MockExtractorService struct {
	ExtractFunc func(file *image.RGBA) (string, error)
}

// Extract implements extractor.Service.
func (m *MockExtractorService) Extract(file *image.RGBA) (string, error) {
	return m.ExtractFunc(file)
}

// Close implements extractor.Service.
func (m *MockExtractorService) Close() error {
	return nil
}
