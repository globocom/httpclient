package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/dnscache"
)

// Transport accepts a custom RoundTripper and acts as a middleware to facilitate logging and
// argument passing to external requests.
type Transport struct {
	RoundTripper http.RoundTripper
	http.Transport
	Proxy    func(*http.Request) (*url.URL, error)
	Resolver interface{}
}

func NewDefaultTransport(transportTimeout time.Duration) *Transport {
	return &Transport{
		RoundTripper: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   transportTimeout,
				KeepAlive: 5 * time.Minute,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				ClientSessionCache: tls.NewLRUClientSessionCache(-1),
			},
		},
	}
}

func (t *Transport) SetProxy(proxy func(*http.Request) (*url.URL, error)) *Transport {
	t.Proxy = proxy
	return t
}

func (t *Transport) SetDNSCache(keepAliveDuration time.Duration, refreshCacheTime time.Duration) *Transport {

	r := &dnscache.Resolver{}
	options := dnscache.ResolverRefreshOptions{
		ClearUnused:      true,
		PersistOnFailure: false,
	}
	r.RefreshWithOptions(options)

	stopTicker := make(chan struct{})
	done := make(chan struct{})

	go func() {
		t := time.NewTicker(refreshCacheTime)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				r.Refresh(true)
			case <-stopTicker:
				close(done)
				return
			}
		}
	}()

	t.DialContext = func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := r.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}

		dialer := net.Dialer{
			KeepAlive: keepAliveDuration,
		}

		var lastErr error
		for _, ip := range ips {
			conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, nil
			}
			lastErr = fmt.Errorf("failed to connect to %s: %w", net.JoinHostPort(ip, port), err)
		}

		return nil, fmt.Errorf("all attempts to connect to %s failed: %w", addr, lastErr)
	}

	defer func() {
		close(stopTicker)
		<-done
	}()

	return t
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
	rID := RequestID(ctx)
	if rID == "" {
		return
	}
	req.Header.Add("X-Request-ID", rID)
}
