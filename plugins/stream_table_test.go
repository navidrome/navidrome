package plugins

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/plugins/streams"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// blockingCloseReader signals when Close starts, then blocks until released.
type blockingCloseReader struct {
	started, release chan struct{}
}

func newBlockingCloseReader() *blockingCloseReader {
	return &blockingCloseReader{started: make(chan struct{}), release: make(chan struct{})}
}

func (r *blockingCloseReader) Read([]byte) (int, error) { return 0, io.EOF }

func (r *blockingCloseReader) Close() error {
	close(r.started)
	<-r.release
	return nil
}

var _ = Describe("streamTable", func() {
	var (
		ctx   context.Context
		table *streamTable
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		table = newStreamTable()
	})

	newScope := func() *callScope {
		s, err := table.newScope(ctx)
		Expect(err).ToNot(HaveOccurred())
		return s
	}
	reader := func() (*trackingReader, *streamBackend) {
		r := &trackingReader{Reader: strings.NewReader("data")}
		return r, newReadStream(r)
	}
	mustRegister := func(s *callScope, b *streamBackend) uint32 {
		h, err := s.register(b)
		Expect(err).ToNot(HaveOccurred())
		return h
	}

	It("numbers handles from 1 up", func() {
		s := newScope()
		_, b1 := reader()
		_, b2 := reader()
		Expect(mustRegister(s, b1)).To(Equal(uint32(1)))
		Expect(mustRegister(s, b2)).To(Equal(uint32(2)))
	})

	It("finds a handle only in its own scope", func() {
		a, b := newScope(), newScope()
		_, backend := reader()
		h := mustRegister(a, backend)

		_, err := b.lookup(h, streamRead)
		Expect(err).To(MatchError(errUnknownStream))

		found, err := a.lookup(h, streamRead)
		Expect(err).ToNot(HaveOccurred())
		Expect(found).To(BeIdenticalTo(backend))
	})

	It("checks the rights stored at registration", func() {
		s := newScope()
		h := mustRegister(s, newWriteStream(&trackingWriter{}))
		_, err := s.lookup(h, streamRead)
		Expect(err).To(MatchError(errStreamNotReadable))

		h = mustRegister(s, newReadStream(&trackingReader{Reader: strings.NewReader("")}))
		_, err = s.lookup(h, streamWrite)
		Expect(err).To(MatchError(errStreamNotWritable))
	})

	It("caps open handles across scopes and closes the one it refuses", func() {
		a, b := newScope(), newScope()
		for i := range maxOpenStreams {
			s := a
			if i%2 == 1 {
				s = b
			}
			_, backend := reader()
			mustRegister(s, backend)
		}
		r, backend := reader()
		_, err := a.register(backend)
		Expect(err).To(MatchError(errTooManyStreams))
		Expect(r.closed()).To(BeTrue())
	})

	It("frees a slot when a handle is transferred", func() {
		s := newScope()
		var last uint32
		for range maxOpenStreams {
			_, backend := reader()
			last = mustRegister(s, backend)
		}
		_, err := s.transfer(last)
		Expect(err).ToNot(HaveOccurred())
		_, backend := reader()
		mustRegister(s, backend)
	})

	It("detaches a handle without closing it", func() {
		s := newScope()
		r, backend := reader()
		h := mustRegister(s, backend)
		got, err := s.detach(h)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeIdenticalTo(backend))
		Expect(r.closed()).To(BeFalse())
		_, err = s.lookup(h, streamRead)
		Expect(err).To(MatchError(errUnknownStream))
	})

	It("closes every handle left when the scope closes, once", func() {
		s := newScope()
		r1, b1 := reader()
		r2, b2 := reader()
		mustRegister(s, b1)
		mustRegister(s, b2)
		s.close()
		s.close()
		Expect(r1.closes.Load()).To(Equal(int32(1)))
		Expect(r2.closes.Load()).To(Equal(int32(1)))
	})

	DescribeTable("reports why the call ended when a handle is used after it",
		func(stop func(*callScope, context.CancelFunc), want error) {
			callCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			s, err := table.newScope(callCtx)
			Expect(err).ToNot(HaveOccurred())
			_, backend := reader()
			h := mustRegister(s, backend)
			stop(s, cancel)
			_, err = s.lookup(h, streamRead)
			Expect(err).To(MatchError(want))
			_, err = s.detach(h)
			Expect(err).To(MatchError(want))
		},
		Entry("cancelled", func(_ *callScope, cancel context.CancelFunc) { cancel() }, context.Canceled),
		Entry("cancelled and closed", func(s *callScope, cancel context.CancelFunc) { cancel(); s.close() }, context.Canceled),
		Entry("ended", func(s *callScope, _ context.CancelFunc) { s.close() }, errCallEnded),
		Entry("unloaded", func(*callScope, context.CancelFunc) { _ = table.Close() }, errPluginUnloaded),
	)

	It("closes a stream registered after the scope closed", func() {
		s := newScope()
		s.close()
		r, backend := reader()
		_, err := s.register(backend)
		Expect(err).To(MatchError(errCallEnded))
		Expect(r.closed()).To(BeTrue())
	})

	It("refuses to register once the call ctx is cancelled, even before the scope closes", func() {
		callCtx, cancel := context.WithCancel(ctx)
		s, err := table.newScope(callCtx)
		Expect(err).ToNot(HaveOccurred())
		cancel()
		r, backend := reader()
		_, err = s.register(backend)
		Expect(err).To(MatchError(context.Canceled))
		Expect(r.closed()).To(BeTrue())
	})

	It("still reports the ctx error after a cancel closed the scope", func() {
		callCtx, cancel := context.WithCancel(ctx)
		s, err := table.newScope(callCtx)
		Expect(err).ToNot(HaveOccurred())
		cancel()
		s.close()
		_, backend := reader()
		_, err = s.register(backend)
		Expect(err).To(MatchError(context.Canceled))
	})

	It("makes a second close wait until the first has closed every stream", func() {
		s := newScope()
		blocker := newBlockingCloseReader()
		mustRegister(s, newReadStream(blocker))
		go s.close()
		Eventually(blocker.started).Should(BeClosed())

		second := make(chan struct{})
		go func() {
			s.close()
			close(second)
		}()
		Consistently(second, 100*time.Millisecond).ShouldNot(BeClosed())
		close(blocker.release)
		Eventually(second).Should(BeClosed())
	})

	It("refuses registration and transfer in every scope once the table starts closing", func() {
		slow, other := newScope(), newScope()
		blocker := newBlockingCloseReader()
		mustRegister(slow, newReadStream(blocker))
		_, backend := reader()
		h := mustRegister(other, backend)

		closed := make(chan struct{})
		go func() {
			_ = table.Close()
			close(closed)
		}()
		Eventually(blocker.started).Should(BeClosed())

		r, late := reader()
		_, err := other.register(late)
		Expect(err).To(MatchError(errPluginUnloaded))
		Expect(r.closed()).To(BeTrue())
		_, err = other.transfer(h)
		Expect(err).To(MatchError(errPluginUnloaded))

		close(blocker.release)
		Eventually(closed).Should(BeClosed())
	})

	Describe("transfer", func() {
		It("moves the handles out and returns the registered values", func() {
			s := newScope()
			r, backend := reader()
			h := mustRegister(s, backend)

			values, err := s.transfer(h)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(HaveLen(1))
			Expect(values[0]).To(BeIdenticalTo(r))

			_, err = s.lookup(h, streamRead)
			Expect(err).To(MatchError(errUnknownStream))
			s.close()
			Expect(r.closed()).To(BeFalse())
		})

		It("moves nothing when a handle appears twice", func() {
			s := newScope()
			_, b1 := reader()
			_, b2 := reader()
			h1, h2 := mustRegister(s, b1), mustRegister(s, b2)
			_, err := s.transfer(h1, h2, h1)
			Expect(err).To(MatchError(errDuplicateStream))
			_, err = s.lookup(h1, streamRead)
			Expect(err).ToNot(HaveOccurred())
			_, err = s.lookup(h2, streamRead)
			Expect(err).ToNot(HaveOccurred())
		})

		It("moves nothing when a handle belongs to another scope", func() {
			a, b := newScope(), newScope()
			_, b1 := reader()
			_, b2 := reader()
			h1, h2 := mustRegister(a, b1), mustRegister(b, b2)
			_, err := a.transfer(h1, h2)
			Expect(err).To(MatchError(errUnknownStream))
			_, err = a.lookup(h1, streamRead)
			Expect(err).ToNot(HaveOccurred())
		})

		It("fails once the call ctx is cancelled", func() {
			callCtx, cancel := context.WithCancel(ctx)
			s, err := table.newScope(callCtx)
			Expect(err).ToNot(HaveOccurred())
			_, backend := reader()
			h := mustRegister(s, backend)
			cancel()
			_, err = s.transfer(h)
			Expect(err).To(MatchError(context.Canceled))
		})

		It("fails after the scope closed", func() {
			s := newScope()
			_, backend := reader()
			h := mustRegister(s, backend)
			s.close()
			_, err := s.transfer(h)
			Expect(err).To(MatchError(errCallEnded))
		})
	})

	It("wraps handles around after 2^31-1 and skips the live ones", func() {
		a, b := newScope(), newScope()
		_, b1 := reader()
		_, b2 := reader()
		Expect(mustRegister(a, b1)).To(Equal(uint32(1)))
		Expect(mustRegister(b, b2)).To(Equal(uint32(2)))
		table.next = math.MaxInt32 - 1
		_, backend := reader()
		Expect(mustRegister(a, backend)).To(Equal(uint32(math.MaxInt32)))
		_, backend = reader()
		Expect(mustRegister(a, backend)).To(Equal(uint32(3)))
	})

	It("closes all live scopes when the table closes", func() {
		s := newScope()
		r, backend := reader()
		mustRegister(s, backend)
		Expect(table.Close()).To(Succeed())
		Expect(r.closed()).To(BeTrue())
		_, err := table.newScope(ctx)
		Expect(err).To(MatchError(errPluginUnloaded))
	})

	Describe("as streams.Streams", func() {
		It("registers and takes readers and writers with rights checked", func() {
			s := newScope()
			r := &trackingReader{Reader: strings.NewReader("data")}
			h, err := s.RegisterReader(r)
			Expect(err).ToNot(HaveOccurred())

			_, err = s.TakeWriter(h)
			Expect(err).To(MatchError(errStreamNotWritable))

			got, err := s.TakeReader(h)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).ToNot(BeNil())
			_, err = s.TakeReader(h)
			Expect(err).To(MatchError(errUnknownStream))
		})

		It("keeps a taken stream counted and closes it with the scope", func() {
			s := newScope()
			r := &trackingReader{Reader: strings.NewReader("data")}
			h, _ := s.RegisterReader(r)
			_, err := s.TakeReader(h)
			Expect(err).ToNot(HaveOccurred())
			for range maxOpenStreams - 1 {
				_, backend := reader()
				mustRegister(s, backend)
			}
			_, backend := reader()
			_, err = s.register(backend)
			Expect(err).To(MatchError(errTooManyStreams), "the taken stream still holds a slot")

			s.close()
			Expect(r.closes.Load()).To(Equal(int32(1)))
			Expect(table.open).To(BeZero())
		})

		It("frees the slot when the new owner closes a taken stream", func() {
			s := newScope()
			r := &trackingReader{Reader: strings.NewReader("data")}
			h, _ := s.RegisterReader(r)
			got, _ := s.TakeReader(h)
			for range maxOpenStreams - 1 {
				_, backend := reader()
				mustRegister(s, backend)
			}
			Expect(got.Close()).To(Succeed())
			_, backend := reader()
			mustRegister(s, backend)

			s.close()
			Expect(r.closes.Load()).To(Equal(int32(1)), "closed once, by its owner")
			Expect(table.open).To(BeZero())
		})

		It("frees the slot once when the owner closes a taken stream after the scope", func() {
			s := newScope()
			r := &trackingReader{Reader: strings.NewReader("data")}
			h, _ := s.RegisterReader(r)
			got, err := s.TakeReader(h)
			Expect(err).ToNot(HaveOccurred())

			s.close()
			Expect(table.open).To(BeZero())
			Expect(got.Close()).To(Succeed())
			Expect(table.open).To(BeZero())
			Expect(r.closes.Load()).To(Equal(int32(1)))
		})

		It("refuses to take from a closed scope", func() {
			s := newScope()
			h, _ := s.RegisterReader(&trackingReader{Reader: strings.NewReader("x")})
			s.close()
			_, err := s.TakeReader(h)
			Expect(err).To(MatchError(errCallEnded))
		})

		It("releases a registered stream whose handle never reached the plugin", func() {
			s := newScope()
			r := &trackingReader{Reader: strings.NewReader("x")}
			h, _ := s.RegisterReader(r)
			s.Release(h)
			Expect(r.closed()).To(BeTrue())
			_, err := s.lookup(h, streamRead)
			Expect(err).To(MatchError(errUnknownStream))
		})

		Describe("Hold", func() {
			It("counts a held closer toward the open-stream limit until released", func() {
				s := newScope()
				for range maxOpenStreams - 1 {
					_, backend := reader()
					mustRegister(s, backend)
				}
				release, err := s.Hold(&trackingReader{Reader: strings.NewReader("")})
				Expect(err).ToNot(HaveOccurred())
				_, backend := reader()
				_, err = s.register(backend)
				Expect(err).To(MatchError(errTooManyStreams))

				release()
				_, backend = reader()
				mustRegister(s, backend)
			})

			It("closes a held closer with the scope", func() {
				s := newScope()
				c := &trackingReader{Reader: strings.NewReader("")}
				_, err := s.Hold(c)
				Expect(err).ToNot(HaveOccurred())
				s.close()
				Expect(c.closes.Load()).To(Equal(int32(1)))
				Expect(table.open).To(BeZero())
			})

			It("does not close a released closer", func() {
				s := newScope()
				c := &trackingReader{Reader: strings.NewReader("")}
				release, err := s.Hold(c)
				Expect(err).ToNot(HaveOccurred())
				release()
				s.close()
				Expect(c.closed()).To(BeFalse())
				Expect(table.open).To(BeZero())
			})

			It("closes the closer at once on a closed scope", func() {
				s := newScope()
				s.close()
				c := &trackingReader{Reader: strings.NewReader("")}
				_, err := s.Hold(c)
				Expect(err).To(MatchError(errCallEnded))
				Expect(c.closed()).To(BeTrue())
			})
		})

		It("is found through the call ctx", func() {
			s := newScope()
			Expect(callScopeFrom(withCallScope(ctx, s))).To(BeIdenticalTo(s))
			Expect(streams.From(withCallScope(ctx, s))).To(BeIdenticalTo(s))
		})
	})
})

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

// errWriter takes at most accept bytes of each Write and reports err with them.
type errWriter struct {
	accept int
	err    error
}

func (w *errWriter) Write(p []byte) (int, error) { return min(len(p), w.accept), w.err }
func (w *errWriter) Close() error                { return nil }

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

		It("Close reports a retained write error", func() {
			w := &errWriter{accept: math.MaxInt, err: errors.New("flush failed")}
			b := newWriteStream(w)
			n, err := b.Write([]byte("abcd"))
			Expect(n).To(Equal(4))
			Expect(err).To(MatchError("flush failed"))
			Expect(b.Close()).To(MatchError("flush failed"))
			Expect(b.Close()).To(MatchError("flush failed"))
		})
	})

	Describe("transferValue", func() {
		It("hands back the registered value", func() {
			r := &trackingReader{Reader: strings.NewReader("x")}
			Expect(newReadStream(r).transferValue()).To(BeIdenticalTo(r))
		})

		It("hands back the registered value after a read to the end", func() {
			r := &trackingReader{Reader: strings.NewReader("x")}
			b := newReadStream(r)
			_, err := io.ReadAll(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(b.transferValue()).To(BeIdenticalTo(r))
		})

		It("hands back the backend when an error is retained", func() {
			b := newReadStream(io.NopCloser(&dataErrReader{data: []byte("a"), err: errors.New("boom")}))
			_, _ = b.Read(make([]byte, 4))
			Expect(b.transferValue()).To(BeIdenticalTo(b))
		})
	})
})
