package httpclient

import (
	"context"
	"net/http"
)

// Transport accepts a custom RoundTripper and acts as a middleware to facilitate logging and
// argument passing to external requests.
type Transport struct {
	RoundTripper http.RoundTripper
}

// RoundTrip acts as a middleware performing external requests logging and argument passing to
// external requests.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.setRequestIDHeader(req.Context(), req)
	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (t *Transport) setRequestIDHeader(ctx context.Context, req *http.Request) {
	rID := requestID(ctx)
	if rID == "" {
		return
	}
	req.Header.Add("X-Request-ID", rID)
}
