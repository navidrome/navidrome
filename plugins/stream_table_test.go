package plugins

import (
	"context"
	"io"
	"math"
	"strings"
	"time"

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

			s.close()
			Expect(r.closed()).To(BeFalse())
			_, err = s.lookup(h, streamRead)
			Expect(err).To(MatchError(errUnknownStream))
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

	It("stops issuing handles at 2^31-1", func() {
		s := newScope()
		table.next = math.MaxInt32 - 1
		_, backend := reader()
		Expect(mustRegister(s, backend)).To(Equal(uint32(math.MaxInt32)))
		_, backend = reader()
		_, err := s.register(backend)
		Expect(err).To(MatchError(errStreamHandlesExhausted))
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

	It("carries the scope in a ctx", func() {
		s := newScope()
		Expect(callScopeFrom(withCallScope(ctx, s))).To(BeIdenticalTo(s))
		Expect(callScopeFrom(ctx)).To(BeNil())
	})
})
