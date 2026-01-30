package httpclient_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/globocom/httpclient"

	"github.com/stretchr/testify/assert"
)

var gReq *http.Request

func handleFunc(rw http.ResponseWriter, req *http.Request) {
	gReq = req
	http.SetCookie(rw, &http.Cookie{Name: "testCookie", Value: "test"})
	rw.Header().Add("testHeader", "test")
	if _, err := rw.Write([]byte(`OK`)); err != nil {
		rw.WriteHeader(http.StatusServiceUnavailable)
	}
}

func TestRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleFunc))
	defer server.Close()

	tests := map[string]func(*httpclient.Request) func(*testing.T){
		"SetBody":      testSetBody,
		"SetHeader":    testSetHeader,
		"SetBasicAuth": testSetBasicAuth,
		"SetHostURL":   testSetHostURL,
		"Get":          testGet,
		"Post":         testPost,
		"Put":          testPut,
		"Delete":       testDelete,
	}

	client := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithHostURL(server.URL),
	)

	for name, test := range tests {
		target := client.NewRequest()
		t.Run(name, test(target))
	}
}

func testSetBody(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetBody([]byte("test"))
		resp, err := target.Post("/")

		assert.NoError(t, err)
		if assert.NotNil(t, gReq.Method) {
			assert.Equal(t, "POST", gReq.Method)

			assert.NoError(t, err)
			assert.Equal(t, []byte("OK"), resp.Body())
		}
	}
}

func testSetHeader(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetHeader("MyHeader", "MyValue")
		_, err := target.Get("/")

		assert.NoError(t, err)
		if assert.NotNil(t, gReq.Method) {
			assert.Equal(t, "GET", gReq.Method)

			assert.NoError(t, err)
			assert.Equal(t, "MyValue", gReq.Header.Get("MyHeader"))
		}
	}
}

func testSetBasicAuth(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetBasicAuth("Username", "Password")
		_, err := target.Get("/")

		assert.NoError(t, err)
		if assert.NotNil(t, gReq.Method) {
			assert.Equal(t, "GET", gReq.Method)

			assert.NoError(t, err)
			assert.Equal(t, "Basic VXNlcm5hbWU6UGFzc3dvcmQ=", gReq.Header.Get("Authorization"))
		}
	}
}

func testGet(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetBody([]byte("test"))
		_, err := target.Get("/")

		assert.NoError(t, err)
		assert.Equal(t, "GET", gReq.Method)
	}
}

func testPost(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetBody([]byte("test"))
		_, err := target.Post("/")

		assert.NoError(t, err)
		assert.Equal(t, "POST", gReq.Method)
	}
}

func testPut(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetBody([]byte("test"))
		_, err := target.Put("/")

		assert.NoError(t, err)
		assert.Equal(t, "PUT", gReq.Method)
	}
}

func testDelete(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		target.SetBody([]byte("test"))
		_, err := target.Delete("/")

		assert.NoError(t, err)
		assert.Equal(t, "DELETE", gReq.Method)
	}
}

func testSetHostURL(target *httpclient.Request) func(*testing.T) {
	return func(t *testing.T) {
		// Create a new URL to set
		newURL, err := url.Parse("https://example.com:8080")
		assert.NoError(t, err)

		// Test setting the host URL
		result := target.SetHostURL(newURL)

		// Verify the method returns the request instance (for chaining)
		assert.Equal(t, target, result)

		// Verify the host URL was set correctly
		hostURL := target.HostURL()
		assert.NotNil(t, hostURL)
		assert.Equal(t, "https://example.com:8080", hostURL.String())
		assert.Equal(t, "example.com", hostURL.Hostname())
		assert.Equal(t, "8080", hostURL.Port())
		assert.Equal(t, "https", hostURL.Scheme)

		// Test with nil URL
		result2 := target.SetHostURL(nil)
		assert.Equal(t, target, result2)
		assert.Nil(t, target.HostURL())
	}
}

func TestSetMetricsAttrs_PropagatesToMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleFunc))
	defer server.Close()

	metrics := &mockMetrics{}
	client := httpclient.NewHTTPClient(
		&httpclient.LoggerAdapter{Writer: io.Discard},
		httpclient.WithHostURL(server.URL),
		httpclient.WithMetrics(metrics),
	)
	request := client.NewRequest()

	attrs := map[string]string{"foo": "bar", "baz": "qux"}
	request.SetMetricsAttrs(attrs)
	_, _ = request.Get("/")

	time.Sleep(100 * time.Millisecond)

	found := false
	for _, calledAttrs := range metrics.incrCounterWithAttrsCalls {
		if calledAttrs.attrs["foo"] == "bar" && calledAttrs.attrs["baz"] == "qux" {
			found = true
			break
		}
	}
	assert.True(t, found, "Attributes passed to SetMetricsAttrs must be propagated to IncrCounterWithAttrs")
}
