# httpclient

A HTTP client implementation in GoLang.

# Examples

### 1. OAuth Authorization

```go
package example

import (
	"context"
	"log"
	"time"
	
	"github.com/globocom/httpclient"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
    timeout := 200 * time.Millisecond
    contextTimeout, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    credentials := clientcredentials.Config{
        ClientID:     "client_id",
        ClientSecret: "client_secret",
        TokenURL:     "client_url/token",
        Scopes:       []string{"grant_permissions:client_credentials"},
    }

    client := httpclient.NewHTTPClient(log.Writer(),
        httpclient.WithOAUTHTransport(credentials, timeout))
    
    resp, err := client.NewRequest().
        SetContext(contextTimeout).
        Put("/authorize")
    
    log.Printf("resp: %#v", resp)
    log.Printf("err: %s", err)
}
```
### 2. Circuit Breaker with Timeout and Retries
```go
package example

import (
	"log"
	"time"
	
	"github.com/slok/goresilience/circuitbreaker"
	"github.com/globocom/httpclient"
)
func main() {
    cbConfig := circuitbreaker.Config {
        ErrorPercentThresholdToOpen:        5, 
        MinimumRequestToOpen:               50, 
        SuccessfulRequiredOnHalfOpen:       50, 
        WaitDurationInOpenState:            30 * time.Second, 
        MetricsSlidingWindowBucketQuantity: 5, 
        MetricsBucketDuration:              5 * time.Second,
    }
		
    timeout := 200 * time.Millisecond
    retries := 1
    backoff := 5 * time.Millisecond
    maxBackoff := 10 * time.Millisecond
    
    client := httpclient.NewHTTPClient(log.Writer(),
        httpclient.WithDefaultTransport(timeout),
        httpclient.WithTimeout(timeout),
        httpclient.WithCircuitBreaker(cbConfig),
        httpclient.WithRetries(retries, backoff, maxBackoff),
    )
    
    resp, err := client.NewRequest().
        Get("/example")

    log.Printf("resp: %#v", resp)
    log.Printf("err: %s", err)
}
```

### 3. Callback Chain
```go
package example

import (
	"log"

	"github.com/globocom/httpclient"
)

func main() {
    client := httpclient.NewHTTPClient(
        log.Writer(),
        httpclient.WithChainCallback(loggerCallback),
    )
    resp, err := client.NewRequest().Get("example.com")
    log.Printf("resp: %#v", resp)
    log.Printf("err: %s", err)
}

func loggerCallback(fn func() (*httpclient.Response, error)) (*httpclient.Response, error) {
    resp, err := fn()
    
if resp != nil {
    restyRequest := resp.Request().RestyRequest()
    requestURL := restyRequest.RawRequest.URL
    host := requestURL.Host
    // If the client is initialized without WithHostURL, request.HostURL() is going to be nil
    if requestHostURL := resp.Request().HostURL(); requestHostURL != nil {
        host = requestHostURL.Host
    }
	
    responseTime := resp.ResponseTime().Microseconds()
    log.Printf("%s [%s] %d -- %s (%dμs)", host, restyRequest.Method,
        resp.StatusCode(), requestURL.String(), responseTime)
}
	
    return resp, err
}
```