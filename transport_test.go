package httpclient_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/globocom/httpclient"
	"github.com/stretchr/testify/assert"
)

func TestHTTPClientTransport(t *testing.T) {
	t.Run("TestDefault", TestNewDefaultTransport)
	t.Run("TestSetProxy", TestSetProxy)
	t.Run("TestDNSCacheBehavior", TestDNSCacheBehavior)
}

func TestNewDefaultTransport(t *testing.T) {
	timeout := 5 * time.Second
	transport := httpclient.NewDefaultTransport(timeout)

	assert.NotNil(t, transport)
	assert.IsType(t, &http.Transport{}, transport.RoundTripper)

	httpTransport := transport.RoundTripper.(*http.Transport)
	assert.Equal(t, 100, httpTransport.MaxIdleConns)
	assert.Equal(t, 90*time.Second, httpTransport.IdleConnTimeout)
	assert.Equal(t, 10*time.Second, httpTransport.TLSHandshakeTimeout)
	assert.Equal(t, 1*time.Second, httpTransport.ExpectContinueTimeout)
}

func TestSetProxy(t *testing.T) {
	transport := httpclient.NewDefaultTransport(5 * time.Second)

	proxyFunc := func(req *http.Request) (*url.URL, error) {
		return url.Parse("http://example.com")
	}

	transport.SetProxy(proxyFunc)

	assert.NotNil(t, transport.Proxy, "Expected Proxy to be non-nil after setting it")
	proxyURL, err := transport.Proxy(&http.Request{})
	assert.NoError(t, err, "Expected no error when calling proxy function")
	assert.Equal(t, "http://example.com", proxyURL.String(), "Expected Proxy URL to match set value")
}

func TestDNSCacheBehavior(t *testing.T) {
	transport := httpclient.NewDefaultTransport(5 * time.Minute)
	keepAliveDuration := 5 * time.Minute
	tr := transport.SetDNSCache(keepAliveDuration, 5*time.Minute)

	ctx := context.Background()

	conn, err := tr.DialContext(ctx, "tcp", "example.com:80")
	assert.NoError(t, err, "Expected no error dialing example.com on first attempt")
	assert.NotNil(t, conn, "Expected a connection object on first attempt")
	if conn != nil {
		conn.Close()
	}

	// cached DNS
	conn, err = tr.DialContext(ctx, "tcp", "example.com:80")
	assert.NoError(t, err, "Expected no error dialing example.com on second attempt")
	assert.NotNil(t, conn, "Expected a connection object on second attempt")
	if conn != nil {
		conn.Close()
	}

}
