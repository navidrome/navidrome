package plugins

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// dataErrReader returns its data together with err on the first Read, then err alone.
type dataErrReader struct {
	data []byte
	err  error
}

func (r *dataErrReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
}

type noProgressReader struct{}

func (noProgressReader) Read([]byte) (int, error) { return 0, nil }

type trackingReader struct {
	io.Reader
	closes atomic.Int32
}

func (r *trackingReader) Close() error { r.closes.Add(1); return nil }
func (r *trackingReader) closed() bool { return r.closes.Load() > 0 }

type failingCloseReader struct{ io.Reader }

func (failingCloseReader) Close() error { return errors.New("close failed") }

// trackingWriter keeps what it receives; limit > 0 makes it accept at most limit bytes per Write.
type trackingWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	limit  int
	closes atomic.Int32
}

func (w *trackingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.limit > 0 && len(p) > w.limit {
		p = p[:w.limit]
	}
	return w.buf.Write(p)
}

func (w *trackingWriter) Close() error { w.closes.Add(1); return nil }
func (w *trackingWriter) closed() bool { return w.closes.Load() > 0 }

func (w *trackingWriter) bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return bytes.Clone(w.buf.Bytes())
}

var _ = Describe("streamBackend", func() {
	Describe("Read", func() {
		It("returns data from the wrapped reader", func() {
			b := newReadStream(io.NopCloser(strings.NewReader("hello")))
			data, err := io.ReadAll(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("hello"))
		})

		It("reports data first and keeps the error that came with it", func() {
			boom := errors.New("disk gone")
			b := newReadStream(io.NopCloser(&dataErrReader{data: []byte("abc"), err: boom}))
			buf := make([]byte, 10)

			n, err := b.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("abc"))

			_, err = b.Read(buf)
			Expect(err).To(MatchError(boom))
			_, err = b.Read(buf)
			Expect(err).To(MatchError(boom))
		})

		It("keeps returning EOF after the end", func() {
			b := newReadStream(io.NopCloser(strings.NewReader("")))
			_, err := b.Read(make([]byte, 1))
			Expect(err).To(MatchError(io.EOF))
			_, err = b.Read(make([]byte, 1))
			Expect(err).To(MatchError(io.EOF))
		})

		It("refuses to read a write-only stream", func() {
			b := newWriteStream(&trackingWriter{})
			_, err := b.Read(make([]byte, 1))
			Expect(err).To(MatchError(errStreamNotReadable))
		})

		It("gives up on a reader that never makes progress, for every consumer", func() {
			b := newReadStream(io.NopCloser(noProgressReader{}))
			_, err := b.Read(make([]byte, 8))
			Expect(err).To(MatchError(io.ErrNoProgress))

			n, err := io.Copy(io.Discard, newReadStream(io.NopCloser(noProgressReader{})))
			Expect(n).To(BeZero())
			Expect(err).To(MatchError(io.ErrNoProgress))
		})
	})

	Describe("Write", func() {
		It("passes bytes to the wrapped writer", func() {
			w := &trackingWriter{}
			n, err := newWriteStream(w).Write([]byte("hi"))
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(2))
			Expect(string(w.bytes())).To(Equal("hi"))
		})

		It("turns a short write into io.ErrShortWrite and keeps it", func() {
			b := newWriteStream(&trackingWriter{limit: 2})
			n, err := b.Write([]byte("abcd"))
			Expect(n).To(Equal(2))
			Expect(err).To(MatchError(io.ErrShortWrite))

			_, err = b.Write([]byte("x"))
			Expect(err).To(MatchError(io.ErrShortWrite))
		})

		It("refuses to write a read-only stream", func() {
			b := newReadStream(io.NopCloser(strings.NewReader("")))
			_, err := b.Write([]byte("x"))
			Expect(err).To(MatchError(errStreamNotWritable))
		})
	})

	Describe("Close", func() {
		It("closes the wrapped value once and repeats its result", func() {
			r := &trackingReader{Reader: strings.NewReader("")}
			b := newReadStream(r)
			Expect(b.Close()).To(Succeed())
			Expect(b.Close()).To(Succeed())
			Expect(r.closes.Load()).To(Equal(int32(1)))
			Expect(r.closed()).To(BeTrue())

			w := &trackingWriter{}
			Expect(newWriteStream(w).Close()).To(Succeed())
			Expect(w.closed()).To(BeTrue())

			failing := newReadStream(failingCloseReader{strings.NewReader("")})
			Expect(failing.Close()).To(MatchError("close failed"))
			Expect(failing.Close()).To(MatchError("close failed"))
		})
	})

	Describe("transferValue", func() {
		It("hands back the registered value", func() {
			r := &trackingReader{Reader: strings.NewReader("x")}
			Expect(newReadStream(r).transferValue()).To(BeIdenticalTo(r))
		})

		It("hands back the backend when an error is retained", func() {
			b := newReadStream(io.NopCloser(&dataErrReader{data: []byte("a"), err: errors.New("boom")}))
			_, _ = b.Read(make([]byte, 4))
			Expect(b.transferValue()).To(BeIdenticalTo(b))
		})
	})
})
