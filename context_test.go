package httpclient_test

import (
	"context"

	"github.com/globocom/httpclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("requestID", func() {
	It("returns the request id included within the values", func() {
		ctx := context.WithValue(context.Background(), httpclient.ContextRequestIDKey, "42")

		id := httpclient.RequestID(ctx)

		Expect(id).To(Equal("42"))
	})

	It("returns blank string if request id is not present on the context", func() {
		ctx := context.Background()

		id := httpclient.RequestID(ctx)

		Expect(id).To(Equal(""))
	})
})
