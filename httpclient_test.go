package httpclient_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/globocom/httpclient"
	"github.com/slok/goresilience/circuitbreaker"
	goresilienceErrors "github.com/slok/goresilience/errors"
	"github.com/stretchr/testify/assert"
)

func TestHTTPClient(t *testing.T) {
	t.Run("CircuitBreaker", testCircuitBreaker)
	t.Run("Retries", testRetries)
	t.Run("Callback", testCallback)
}

func testCircuitBreaker(t *testing.T) {
	openDuration := 500 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(handleFunc))
	defer server.Close()

	errorThreshold := 1

	client := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithDefaultTransport(1*time.Second),
		httpclient.WithTimeout(1*time.Second),
		httpclient.WithCircuitBreaker(circuitbreaker.Config{
			ErrorPercentThresholdToOpen: errorThreshold,
			MinimumRequestToOpen:        errorThreshold,
			WaitDurationInOpenState:     openDuration,
		}),
	)

	_, err := client.NewRequest().Get("")
	assert.Equal(t, "Get \"\": unsupported protocol scheme \"\"", err.Error())

	for range [2]struct{}{} {
		_, err = client.NewRequest().Get("")
		assert.Equal(t, goresilienceErrors.ErrCircuitOpen, err)
	}

	time.Sleep(2 * openDuration)

	_, err = client.NewRequest().Get(server.URL)
	assert.NoError(t, err)
}

func testRetries(t *testing.T) {
	expectedTimes := 3
	waitAmount := 500 * time.Millisecond
	clientLinear := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithDefaultTransport(1*time.Second),
		httpclient.WithTimeout(1*time.Second),
		httpclient.WithLinearBackoff(expectedTimes, waitAmount),
	)

	clientExponential := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithDefaultTransport(1*time.Second),
		httpclient.WithTimeout(1*time.Second),
		httpclient.WithExponentialBackoff(expectedTimes, waitAmount),
	)

	clients := map[string]*httpclient.HTTPClient{
		"Linear":      clientLinear,
		"Exponential": clientExponential,
	}

	for scn, client := range clients {
		t.Run(scn, func(t *testing.T) {
			times := 0
			server := httptest.NewServer(http.HandlerFunc(
				func(rw http.ResponseWriter, req *http.Request) {
					if times < expectedTimes-1 {
						// The client will return an error in this case, because it doesn't
						// follow 302 by default
						rw.Header().Add("Location", "test")
						rw.WriteHeader(302)
					}
					times++
				},
			))

			_, err := client.NewRequest().Get(server.URL)
			assert.NoError(t, err)
			assert.Equal(t, expectedTimes, times)
			server.Close()
		})
	}

}

func testCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleFunc))
	defer server.Close()

	loggerCallback := func(logger io.Writer) httpclient.Callback {
		return func(fn func() (*httpclient.Response, error)) (*httpclient.Response, error) {
			resp, _ := fn()
			logger.Write([]byte(fmt.Sprint(resp.StatusCode())))
			return resp, nil
		}
	}

	var b bytes.Buffer
	writer := io.Writer(&b)

	client := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithChainCallback(loggerCallback(writer)),
	)

	resp, _ := client.NewRequest().Get(server.URL)

	assert.Equal(t, fmt.Sprint(resp.StatusCode()), b.String())
}
