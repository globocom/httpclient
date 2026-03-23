package httpclient

import (
	"context"
	"net/http"
	"net/url"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/slok/goresilience/circuitbreaker"
	goresilienceErrors "github.com/slok/goresilience/errors"
	"github.com/slok/goresilience/retry"
	"golang.org/x/oauth2"
	cc "golang.org/x/oauth2/clientcredentials"
)

var ErrCircuitOpen = goresilienceErrors.ErrCircuitOpen

type (
	Callback func(func() (*Response, error)) (*Response, error)

	Opt func(*HTTPClient)

	HTTPClient struct {
		resty         *resty.Client
		hostURL       *url.URL
		metrics       Metrics
		callbackChain Callback
	}
)

// NewHTTPClient instantiates a new HTTPClient.
//
// Parameters:
//
//	logger: interface is used to log request and response details.
//	options: specifies options to HTTPClient.
func NewHTTPClient(logger resty.Logger, options ...Opt) *HTTPClient {
	return newClient(resty.New().SetLogger(logger).GetClient(), options...)
}

func newClient(customClient *http.Client, options ...Opt) *HTTPClient {
	client := &HTTPClient{
		resty:         resty.NewWithClient(customClient),
		callbackChain: noopCallback,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// GetClient returns the current http.Client.
func (c *HTTPClient) GetClient() *http.Client {
	return c.resty.GetClient()
}

func (c *HTTPClient) chainCallback(newCallback Callback) {
	previousCallback := c.callbackChain

	if previousCallback == nil {
		c.callbackChain = newCallback
		return
	}

	c.callbackChain = func(fn func() (*Response, error)) (*Response, error) {
		return newCallback(func() (*Response, error) {
			return previousCallback(fn)
		})
	}
}

func (c *HTTPClient) setTransport(transport http.RoundTripper) {
	c.resty.SetTransport(transport)
}

// WithDefaultTransport sets a custom connection timeout to http.Transport.
// This timeout limits the time spent establishing a TCP connection.
//
// More information about timeout: net.Dialer.
func WithDefaultTransport(transportTimeout time.Duration) func(*HTTPClient) {
	return func(client *HTTPClient) {
		transport := NewDefaultTransport(transportTimeout)
		client.setTransport(transport)
	}
}

// WithTransport configures the client to use a custom *http.Transport
// More information about transport: [net/http.Transport]
func WithTransport(transport *http.Transport) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.setTransport(transport)
	}
}

// WithOAUTHTransport allows the client to make OAuth HTTP requests with custom timeout.
// This timeout limits the time spent establishing a TCP connection.
//
// The oauth2.Transport adds an Authorization header with a token
// using clientcredentials.Config information.
//
// More information about timeout: net.Dialer.
//
// More information about the fields used to create the token: clientcredentials.Config.
func WithOAUTHTransport(conf cc.Config, transportTimeout time.Duration) func(*HTTPClient) {
	return func(client *HTTPClient) {
		transport := &oauth2.Transport{
			Source: conf.TokenSource(context.Background()),
			Base:   NewDefaultTransport(transportTimeout),
		}
		client.setTransport(transport)
	}
}

// WithDefaultTransportWithProxy sets a custom url to use as a proxy to requests.
// The proxyURL is used in the Proxy field. This field specifies a function
// to return a proxy for a given request.
//
// More information about proxy: http.Transport.
func WithDefaultTransportWithProxy(proxyURL *url.URL) func(*HTTPClient) {
	return func(client *HTTPClient) {
		transport := NewDefaultTransport(5 * time.Second)
		transport.SetProxy(http.ProxyURL(proxyURL))
		client.setTransport(transport)
	}
}

// WithDefaultTransportWithDNSCache sets a cache for DNS lookups.
// The TTL of the cache is defined by DNS Server TTL.
// The keepAliveDuration is the time to keep the connection alive.
//
// More information about DNS cache: https://github.com/rs/dnscache.
func WithDefaultTransportWithDNSCache(keepAliveDuration time.Duration) func(*HTTPClient) {
	return func(client *HTTPClient) {
		transport := NewDefaultTransport(5 * time.Second)
		transport.SetDNSCache(keepAliveDuration, 5*time.Minute)
		client.setTransport(transport)
	}
}

// WithTimeout encapsulates the resty library to set a custom request timeout.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithTimeout(timeout time.Duration) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.resty.SetTimeout(timeout)
	}
}

// WithUserAgent encapsulates the resty library to set a custom user agent to requests.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithUserAgent(userAgent string) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.resty.SetHeader("User-Agent", userAgent)
	}
}

// WithBasicAuth encapsulates the resty library to provide basic authentication.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithBasicAuth(username, password string) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.resty.SetBasicAuth(username, password)
	}
}

// WithAuthToken encapsulates the resty library to provide token authentication.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithAuthToken(token string) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.resty.SetAuthToken(token)
	}
}

// WithCookie encapsulates the resty library to set a cookie to client instance.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithCookie(name, value string) func(*HTTPClient) {
	return func(client *HTTPClient) {
		c := http.Cookie{Name: name, Value: value, MaxAge: 3600}
		client.resty.SetCookie(&c)
	}
}

// WithHostURL encapsulates the resty library to set a host url.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithHostURL(baseURL string) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.hostURL, _ = url.Parse(baseURL)
		client.resty.SetBaseURL(baseURL)
	}
}

// WithCircuitBreaker enables circuit breaker strategy based on circuitbreaker.Config.
// This functionality relies on https://github.com/slok/goresilience/tree/master/circuitbreaker library.
//
//	The config fields are:
//	    ErrorPercentThresholdToOpen        int
//	    MinimumRequestToOpen               int
//	    SuccessfulRequiredOnHalfOpen       int
//	    WaitDurationInOpenState            time.Duration
//	    MetricsSlidingWindowBucketQuantity int
//	    MetricsBucketDuration              time.Duration
//
// More information about circuitbreaker config: circuitbreaker.Config
func WithCircuitBreaker(config circuitbreaker.Config) func(*HTTPClient) {
	runner := circuitbreaker.New(config)
	circuitBreakerCallback := func(fn func() (*Response, error)) (*Response, error) {
		var resp *Response
		err := runner.Run(context.Background(), func(ctx context.Context) error {
			var err error
			resp, err = fn()
			return err
		})
		return resp, err
	}
	return func(client *HTTPClient) {
		client.chainCallback(circuitBreakerCallback)
	}
}

func WithLinearBackoff(retries int, waitTime time.Duration) func(*HTTPClient) {
	return WithBackoff(retries, waitTime, false)
}

func WithExponentialBackoff(retries int, waitTime time.Duration) func(*HTTPClient) {
	return WithBackoff(retries, waitTime, true)
}

// WithBackoff sets a retry strategy based on its configuration.
// This functionality relies on:
//
//	https://github.com/slok/goresilience/tree/master/circuitbreaker
//	https://github.com/go-resty/resty/tree/v1.x
//
// Parameters:
//
//	retries: is used to set the number of retries after an error occurred.
//	waitTime: is the amount of time to wait for a new retry.
//	exponential: this field is used to specify which kind of backoff is used.
func WithBackoff(retries int, waitTime time.Duration, exponential bool) func(*HTTPClient) {
	r := retry.New(retry.Config{
		WaitBase:       waitTime,
		DisableBackoff: !exponential,
		Times:          retries,
	})
	backoffCallback := func(fn func() (*Response, error)) (*Response, error) {
		var resp *Response
		err := r.Run(context.Background(), func(ctx context.Context) error {
			var err error
			resp, err = fn()
			return err
		})

		return resp, err
	}
	return func(client *HTTPClient) {
		client.resty.SetRetryCount(retries)
		client.chainCallback(backoffCallback)
	}
}

// WithMetrics creates a layer to facilitate the metrics use.
//
//	Metrics interface implements
//	    IncrCounter(name string)
//	    PushToSeries(name string, value float64)
func WithMetrics(m Metrics) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.metrics = m
	}
}

// WithProxy encapsulates the resty library to set a proxy URL and port.
//
// More information about this feature: https://github.com/go-resty/resty/tree/v1.x
func WithProxy(proxyAddress string) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.resty.SetProxy(proxyAddress)
	}
}

// WithRetries sets a retry strategy based on its configuration.
// This functionality relies on:
//
//	https://github.com/go-resty/resty/tree/v1.x
//
// Parameters:
//
//	retries: is used to set the number of retries after an error occurred.
//	waitTime: is the amount of time to wait for a new retry.
//	maxWaitTime: is the MAX amount of time to wait for a new retry.
func WithRetries(retries int, waitTime time.Duration, maxWaitTime time.Duration) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.resty.SetRetryCount(retries)
		client.resty.SetRetryWaitTime(waitTime)
		client.resty.SetRetryMaxWaitTime(maxWaitTime)
	}
}

// WithRetryConditions sets conditions to retry strategy. The conditions will be
// checked for a new retry.
// This functionality relies on:
//
//	https://github.com/go-resty/resty/tree/v1.x
//
// More information about conditions: resty.RetryConditionFunc
func WithRetryConditions(conditions ...resty.RetryConditionFunc) func(*HTTPClient) {
	return func(client *HTTPClient) {
		for _, condition := range conditions {
			client.resty.AddRetryCondition(condition)
		}
	}
}

// WithChainCallback provides a callback functionality that takes as input a Callback type.
func WithChainCallback(fn Callback) func(*HTTPClient) {
	return func(client *HTTPClient) {
		client.chainCallback(fn)
	}
}

func noopCallback(fn func() (*Response, error)) (*Response, error) {
	return fn()
}
