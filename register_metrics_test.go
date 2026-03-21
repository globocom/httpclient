package httpclient_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/globocom/httpclient"
	"github.com/stretchr/testify/assert"
)

type mockMetrics struct {
	pushToSeriesCalls         []string
	incrCounterCalls          []string
	incrCounterWithAttrsCalls []struct {
		key   string
		attrs map[string]string
	}
	lock sync.Mutex
}

func (m *mockMetrics) PushToSeries(key string, value float64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.pushToSeriesCalls = append(m.pushToSeriesCalls, key)
}

func (m *mockMetrics) IncrCounter(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.incrCounterCalls = append(m.incrCounterCalls, key)
}

func (m *mockMetrics) IncrCounterWithAttrs(key string, attrs map[string]string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.incrCounterWithAttrsCalls = append(m.incrCounterWithAttrsCalls, struct {
		key   string
		attrs map[string]string
	}{key, attrs})
}

func TestHTTPClient_MetricsIntegration(t *testing.T) {
	metrics := &mockMetrics{}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	}))
	defer server.Close()

	client := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithHostURL(server.URL),
		httpclient.WithMetrics(metrics),
	)

	req := client.NewRequest()
	resp, err := req.Get("/")
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Eventually(t, func() bool {
		metrics.lock.Lock()
		defer metrics.lock.Unlock()
		return len(metrics.pushToSeriesCalls) > 0 && len(metrics.incrCounterCalls) > 0 && len(metrics.incrCounterWithAttrsCalls) > 0
	}, time.Second, 100*time.Millisecond)

	metrics.lock.Lock()
	defer metrics.lock.Unlock()

	foundPush := false
	for _, key := range metrics.pushToSeriesCalls {
		if strings.HasSuffix(key, "response_time") {
			foundPush = true
			break
		}
	}
	assert.True(t, foundPush, "PushToSeries should register key with suffix response_time")

	foundStatus := false
	for _, key := range metrics.incrCounterCalls {
		if strings.HasSuffix(key, "status.200") {
			foundStatus = true
			break
		}
	}
	assert.True(t, foundStatus, "IncrCounter should register key with suffix status.200")

	foundAttrs := false
	for _, call := range metrics.incrCounterWithAttrsCalls {
		if strings.HasSuffix(call.key, "total") &&
			call.attrs["status"] == "200" &&
			call.attrs["host"] != "" {
			foundAttrs = true
			break
		}
	}
	assert.True(t, foundAttrs, "IncrCounterWithAttrs should register correct attributes and key with suffix total")
}

func TestHTTPClient_Metrics_Errors(t *testing.T) {
	t.Run("circuit open error", func(t *testing.T) {
		metrics := &mockMetrics{}
		client := httpclient.NewHTTPClient(
			&httpclient.LoggerAdapter{Writer: io.Discard},
			httpclient.WithMetrics(metrics),
			httpclient.WithChainCallback(func(fn func() (*httpclient.Response, error)) (*httpclient.Response, error) {
				return nil, httpclient.ErrCircuitOpen
			}),
		)
		_, _ = client.NewRequest().Get("/test-circuit-open")
		time.Sleep(50 * time.Millisecond)
		metrics.lock.Lock()
		defer metrics.lock.Unlock()
		found := false
		for _, call := range metrics.incrCounterCalls {
			if strings.HasSuffix(call, "circuit_open") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should increment circuit_open counter on ErrCircuitOpen")
	})

	t.Run("generic error", func(t *testing.T) {
		metrics := &mockMetrics{}
		client := httpclient.NewHTTPClient(
			&httpclient.LoggerAdapter{Writer: io.Discard},
			httpclient.WithMetrics(metrics),
			httpclient.WithChainCallback(func(fn func() (*httpclient.Response, error)) (*httpclient.Response, error) {
				return nil, assert.AnError
			}),
		)
		_, _ = client.NewRequest().Get("/test-generic-error")
		time.Sleep(50 * time.Millisecond)
		metrics.lock.Lock()
		defer metrics.lock.Unlock()
		foundError := false
		foundAttr := false
		for _, call := range metrics.incrCounterCalls {
			if strings.HasSuffix(call, "errors") {
				foundError = true
				break
			}
		}
		for _, call := range metrics.incrCounterWithAttrsCalls {
			if val, ok := call.attrs["error"]; ok && val == assert.AnError.Error() {
				foundAttr = true
				break
			}
		}
		assert.True(t, foundError, "Should increment errors counter on generic error")
		assert.True(t, foundAttr, "Should add error attribute on generic error")
	})
}
