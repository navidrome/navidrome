package server

import (
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetWriteTimeout", func() {
	It("sets the write deadline on a writer that supports SetWriteDeadline", func() {
		w := &mockDeadlineWriter{}
		err := SetWriteTimeout(w, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		Expect(w.deadline).To(BeTemporally("~", time.Now().Add(30*time.Second), time.Second))
	})

	It("unwraps wrapped writers to find SetWriteDeadline", func() {
		inner := &mockDeadlineResponseWriter{}
		wrapped := &mockUnwrappingWriter{inner: inner}
		err := SetWriteTimeout(wrapped, 15*time.Second)
		Expect(err).ToNot(HaveOccurred())
		Expect(inner.deadline).To(BeTemporally("~", time.Now().Add(15*time.Second), time.Second))
	})

	It("returns ErrNotSupported for writers without SetWriteDeadline", func() {
		w := httptest.NewRecorder()
		err := SetWriteTimeout(w, 10*time.Second)
		Expect(err).To(MatchError(http.ErrNotSupported))
	})
})

// mockDeadlineWriter implements io.Writer + SetWriteDeadline
type mockDeadlineWriter struct {
	deadline time.Time
}

func (w *mockDeadlineWriter) SetWriteDeadline(t time.Time) error {
	w.deadline = t
	return nil
}

func (w *mockDeadlineWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

// mockDeadlineResponseWriter implements http.ResponseWriter + SetWriteDeadline
type mockDeadlineResponseWriter struct {
	httptest.ResponseRecorder
	deadline time.Time
}

func (w *mockDeadlineResponseWriter) SetWriteDeadline(t time.Time) error {
	w.deadline = t
	return nil
}

// mockUnwrappingWriter wraps a ResponseWriter via Unwrap (no SetWriteDeadline itself)
type mockUnwrappingWriter struct {
	inner http.ResponseWriter
}

func (w *mockUnwrappingWriter) Write(p []byte) (int, error) {
	return w.inner.Write(p)
}

func (w *mockUnwrappingWriter) Unwrap() http.ResponseWriter {
	return w.inner
}
