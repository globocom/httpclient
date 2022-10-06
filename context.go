package httpclient

import (
	"context"
)

const contextRequestIDKey = "request.id"

// requestID returns a request present on context.
func requestID(ctx context.Context) string {
	value := ctx.Value(contextRequestIDKey)
	if value == nil {
		return ""
	}

	return value.(string)
}
