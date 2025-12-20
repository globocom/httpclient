package httpclient

import (
	"context"
)

const ContextRequestIDKey = "request.id"

// requestID returns a request present on context.
func RequestID(ctx context.Context) string {
	value := ctx.Value(ContextRequestIDKey)
	if value == nil {
		return ""
	}

	return value.(string)
}
