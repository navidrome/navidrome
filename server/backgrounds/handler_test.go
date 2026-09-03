package backgrounds

import (
	"context"
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type recordingBody struct {
	io.Reader
	closed *bool
}

func (b recordingBody) Close() error {
	*b.closed = true
	return nil
}

type stubTransport struct {
	statusCode int
	closed     *bool
}

func (t stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Header:     make(http.Header),
		Body:       recordingBody{Reader: strings.NewReader("image-bytes"), closed: t.closed},
	}, nil
}

var _ = Describe("serveImage", func() {
	var closed bool

	BeforeEach(func() {
		closed = false
	})

	stubStatus := func(statusCode int) {
		original := http.DefaultTransport
		http.DefaultTransport = stubTransport{statusCode: statusCode, closed: &closed}
		DeferCleanup(func() { http.DefaultTransport = original })
	}

	It("closes the response body when the hosting service returns an error", func() {
		stubStatus(http.StatusNotFound)

		_, err := (&Handler{}).serveImage(context.Background(), cacheKey("some-image.webp"))

		Expect(err).To(MatchError(ContainSubstring("unexpected status code")))
		Expect(closed).To(BeTrue(), "response body was left open")
	})

	It("hands the still-open body to the caller on success", func() {
		stubStatus(http.StatusOK)

		reader, err := (&Handler{}).serveImage(context.Background(), cacheKey("some-image.webp"))

		Expect(err).ToNot(HaveOccurred())
		Expect(closed).To(BeFalse(), "response body must stay open for the CachedStream wrapper")
		body, _ := io.ReadAll(reader)
		Expect(string(body)).To(Equal("image-bytes"))
	})
})
