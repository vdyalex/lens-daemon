package mocks

import (
	"bytes"
	"io"
	"net/http"
)

// NewJSONResponse builds an *http.Response with the given status code and JSON body.
// The body is wrapped with io.NopCloser so it can be read and closed safely.
func NewJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
