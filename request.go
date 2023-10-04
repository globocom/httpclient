package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	resty "gopkg.in/resty.v1"
)

type Request struct {
	alias         string
	chainCallback Callback
	hostURL       *url.URL
	metrics       Metrics
	restyRequest  *resty.Request
	startTime     time.Time
}

// NewRequest creates a request for the specified HTTP method.
func (c *HTTPClient) NewRequest() *Request {
	return &Request{
		restyRequest:  c.resty.NewRequest(),
		chainCallback: c.callbackChain,
		metrics:       c.metrics,
		hostURL:       c.hostURL,
	}
}

// HostURL returns the setted host url.
func (r *Request) HostURL() *url.URL {
	return r.hostURL
}

// SetAlias sets the alias to replace the hostname in metrics.
func (r *Request) SetAlias(alias string) *Request {
	r.alias = alias
	return r
}

// SetBody sets the body for the request.
func (r *Request) SetBody(body interface{}) *Request {
	r.restyRequest.SetBody(body)
	return r
}

// SetContext sets the context for the request.
func (r *Request) SetContext(context context.Context) *Request {
	r.restyRequest.SetContext(context)
	return r
}

// SetHeader sets the header for the request.
func (r *Request) SetHeader(name, value string) *Request {
	r.restyRequest.SetHeader(name, value)
	return r
}

// SetBasicAuth sets the basic authentication header for the request.
func (r *Request) SetBasicAuth(username, password string) *Request {
	r.restyRequest.SetBasicAuth(username, password)
	return r
}

// SetAuthToken sets the bearer authentication header for the request.
func (r *Request) SetAuthToken(bearer string) *Request {
	r.restyRequest.SetAuthToken(bearer)
	return r
}

// SetQueryParams sets parameters to form a query string for the request.
func (r *Request) SetQueryParams(params map[string]string) *Request {
	r.restyRequest.SetQueryParams(params)
	return r
}

// SetPathParams sets multiple key-value pairs to form the path for the request.
func (r *Request) SetPathParams(params map[string]string) *Request {
	r.restyRequest.SetPathParams(params)
	return r
}

// RestyRequest RestyRequest gives access to the underlying *resty.Request.
func (r *Request) RestyRequest() *resty.Request {
	return r.restyRequest
}

// Get performs an HTTP method GET request given an url.
func (r *Request) Get(url string) (*Response, error) {
	return r.Execute("GET", url)
}

// Post performs an HTTP method POST request given an url.
func (r *Request) Post(url string) (*Response, error) {
	return r.Execute("POST", url)
}

// Put performs an HTTP method PUT request given an url.
func (r *Request) Put(url string) (*Response, error) {
	return r.Execute("PUT", url)
}

// Patch performs an HTTP method PATCH request given an url.
func (r *Request) Patch(url string) (*Response, error) {
	return r.Execute("PATCH", url)
}

// Delete performs an HTTP method DELETE request given an url.
func (r *Request) Delete(url string) (*Response, error) {
	return r.Execute("DELETE", url)
}

// Execute performs the HTTP request with given HTTP method and URL.
// It also registers metrics, metrics fields are:
// host/alias occurrences, response time,
// response status code, quantity of occurrence of a circuit breaker open and
// errors occurred.
func (r *Request) Execute(method string, url string) (*Response, error) {
	metricsAlias := url
	if len(r.alias) > 0 {
		metricsAlias = r.alias
	} else if r.hostURL != nil {
		hostname := r.hostURL.Hostname()
		metricsAlias = fmt.Sprintf("%s.%s", method, path.Join(hostname, url))
	}

	metricsAlias = strings.Replace(metricsAlias, ".", "-", -1)

	return registerMetrics(metricsAlias, r.metrics, func() (*Response, error) {
		execute := func() (*Response, error) {
			r.startTime = time.Now()
			restyResponse, err := r.restyRequest.Execute(method, url)
			if restyResponse == nil {
				return nil, err
			}
			return wrapResponse(r, restyResponse), err
		}

		return r.chainCallback(execute)
	})
}

func registerMetrics(key string, metrics Metrics, f func() (*Response, error)) (*Response, error) {
	resp, err := f()

	if metrics != nil {
		go func(resp *Response, err error) {
			attrs := map[string]string{}
			if resp != nil {
				metrics.PushToSeries(fmt.Sprintf("%s.%s", key, "response_time"), resp.ResponseTime().Seconds())
				if resp.statusCode != 0 {
					metrics.IncrCounter(fmt.Sprintf("%s.status.%d", key, resp.StatusCode()))
					attrs["status"] = fmt.Sprintf("%d", resp.StatusCode())
				}
			}
			if err != nil {
				if errors.Is(err, ErrCircuitOpen) {
					metrics.IncrCounter(fmt.Sprintf("%s.%s", key, "circuit_open"))
				} else {
					metrics.IncrCounter(fmt.Sprintf("%s.%s", key, "errors"))
				}
			}
			metrics.IncrCounterWithAttrs(fmt.Sprintf("%s.%s", key, "total"), attrs)
		}(resp, err)
	}

	return resp, err
}
