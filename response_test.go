package httpclient_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/globocom/httpclient"
	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handleFunc))
	defer server.Close()

	client := httpclient.NewHTTPClient(
		io.Discard,
		httpclient.WithHostURL(server.URL),
	)

	target, err := client.NewRequest().Get("/")
	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, target.StatusCode())
		assert.Equal(t, []byte("OK"), target.Body())
		assert.Equal(t, "test", target.Header().Get("Testheader"))
		assert.Equal(t, &http.Cookie{Name: "testCookie", Value: "test", Raw: "testCookie=test"}, target.Cookies()[0])
		assert.Equal(t, "test", target.Cookie("testCookie").Value)
	}
}
