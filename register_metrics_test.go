package httpclient_test

import (
	"io"
	"net/http"
	"net/http/httptest"
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
}
