package httpclient_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
		"Get":          testGet,
		"Post":         testPost,
		"Put":          testPut,
		"Delete":       testDelete,
	}

	client := httpclient.NewHTTPClient(
		io.Discard,
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
