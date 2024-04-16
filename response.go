package httpclient

import (
	"net/http"
	"time"

	resty "github.com/go-resty/resty/v2"
)

type Response struct {
	statusCode   int
	body         []byte
	header       http.Header
	cookies      []*http.Cookie
	request      *Request
	responseTime time.Duration
}

// StatusCode returns the response status code.
func (r Response) StatusCode() int {
	return r.statusCode
}

// Body returns the response body.
func (r Response) Body() []byte {
	return r.body
}

// Header returns the response header.
func (r Response) Header() http.Header {
	return r.header
}

// Cookies returns the response cookies.
func (r Response) Cookies() []*http.Cookie {
	return r.cookies
}

// Cookie finds a cookie by a name and returns it.
func (r Response) Cookie(name string) *http.Cookie {
	for _, c := range r.cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Request returns the received request.
func (r Response) Request() *Request {
	return r.request
}

// ResponseTime returns the request response time.
func (r Response) ResponseTime() time.Duration {
	return r.responseTime
}

func wrapResponse(request *Request, restyResponse *resty.Response) *Response {
	return &Response{
		statusCode:   restyResponse.StatusCode(),
		header:       restyResponse.Header(),
		body:         restyResponse.Body(),
		cookies:      restyResponse.Cookies(),
		request:      request,
		responseTime: time.Since(request.startTime),
	}
}
