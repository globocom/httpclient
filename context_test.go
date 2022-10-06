package httpclient

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("requestID", func() {
	It("returns the request id included within the values", func() {
		ctx := context.WithValue(context.Background(), contextRequestIDKey, "42")

		id := requestID(ctx)

		Expect(id).To(Equal("42"))
	})

	It("returns blank string if request id is not present on the context", func() {
		ctx := context.Background()

		id := requestID(ctx)

		Expect(id).To(Equal(""))
	})
})
