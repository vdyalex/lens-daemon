package mocks

import "image"

// MockExtractorService satisfies extractor.Service interface.
type MockExtractorService struct {
	ExtractFunc func(file *image.RGBA) (string, error)
}

// Extract implements extractor.Service.
func (mock *MockExtractorService) Extract(file *image.RGBA) (string, error) {
	return mock.ExtractFunc(file)
}

// Close implements extractor.Service.
func (mock *MockExtractorService) Close() error {
	return nil
}
