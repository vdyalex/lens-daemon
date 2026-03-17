package mocks

import "net/http"

// MockIMHTTPClient satisfies im.HTTPClient interface.
// It records all requests and allows tests to specify behavior via DoFunc.
type MockIMHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
	Calls  []*http.Request
}

// Do implements im.HTTPClient.
func (mock *MockIMHTTPClient) Do(req *http.Request) (*http.Response, error) {
	mock.Calls = append(mock.Calls, req)
	return mock.DoFunc(req)
}
