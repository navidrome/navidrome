package artwork

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error { c.closed = true; return nil }

var _ = Describe("teeReader", func() {
	It("closes the underlying source exactly once", func() {
		src := &closeSpy{Reader: bytes.NewReader([]byte("hello"))}
		tr := newTeeReader(src, 1024, func([]byte) {})
		_, err := io.ReadAll(tr)
		Expect(err).ToNot(HaveOccurred())
		Expect(tr.Close()).To(Succeed())
		Expect(src.closed).To(BeTrue(), "the source stream must be closed, or its fd leaks")
	})

	It("calls onComplete with the full bytes after a complete read+close", func() {
		var got []byte
		src := io.NopCloser(bytes.NewReader([]byte("hello world")))
		tr := newTeeReader(src, 1024, func(data []byte) { got = data })
		out, err := io.ReadAll(tr)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(Equal("hello world"))
		Expect(tr.Close()).To(Succeed())
		Expect(string(got)).To(Equal("hello world"))
	})

	It("does not call onComplete when the stream is not fully read", func() {
		called := false
		src := io.NopCloser(bytes.NewReader([]byte("hello world")))
		tr := newTeeReader(src, 1024, func(data []byte) { called = true })
		buf := make([]byte, 3)
		_, err := tr.Read(buf) // partial read, then close without EOF
		Expect(err).ToNot(HaveOccurred())
		Expect(tr.Close()).To(Succeed())
		Expect(called).To(BeFalse())
	})

	It("does not call onComplete when the data exceeds maxBytes", func() {
		called := false
		src := io.NopCloser(bytes.NewReader([]byte("hello world")))
		tr := newTeeReader(src, 4, func(data []byte) { called = true })
		_, err := io.ReadAll(tr)
		Expect(err).ToNot(HaveOccurred())
		Expect(tr.Close()).To(Succeed())
		Expect(called).To(BeFalse())
	})
})
