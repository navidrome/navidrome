package artwork

import (
	"context"
	"errors"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resizedItem", func() {
	Describe("Reader", func() {
		newItem := func(open func() (io.ReadCloser, error)) *resizedItem {
			return &resizedItem{hash: "abc123", size: 300, open: open}
		}

		It("reports a nil reader as unavailable instead of panicking on it", func() {
			// Every caller is expected to report "no image" as an error, but a nil reader reaches
			// the deferred Close as a nil interface, which takes the whole request down.
			_, err := newItem(func() (io.ReadCloser, error) { return nil, nil }).Reader(context.Background())
			Expect(err).To(MatchError(ErrUnavailable))
		})

		It("propagates the open error", func() {
			boom := errors.New("boom")
			_, err := newItem(func() (io.ReadCloser, error) { return nil, boom }).Reader(context.Background())
			Expect(err).To(MatchError(boom))
		})

		It("serves the original bytes when they cannot be resized", func() {
			rc, err := newItem(func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("not an image")), nil
			}).Reader(context.Background())
			Expect(err).ToNot(HaveOccurred())
			defer rc.Close()
			Expect(io.ReadAll(rc)).To(Equal([]byte("not an image")))
		})
	})
})
