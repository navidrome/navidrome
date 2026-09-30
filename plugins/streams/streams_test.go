package streams

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type closeCounter struct {
	io.Reader
	io.Writer
	closes atomic.Int32
}

func (c *closeCounter) Close() error { c.closes.Add(1); return nil }

// fakeStreams is a scope with a fixed set of handles; failAfter makes registrations fail once that many succeeded.
type fakeStreams struct {
	readers    map[uint32]io.ReadCloser
	writers    map[uint32]io.WriteCloser
	registered map[uint32]io.Closer
	next       uint32
	failAfter  int
}

func newFakeStreams() *fakeStreams {
	return &fakeStreams{readers: map[uint32]io.ReadCloser{}, writers: map[uint32]io.WriteCloser{},
		registered: map[uint32]io.Closer{}, failAfter: -1}
}

func (f *fakeStreams) register(c io.Closer) (uint32, error) {
	if f.failAfter >= 0 && len(f.registered) >= f.failAfter {
		_ = c.Close()
		return 0, errors.New("too many open streams")
	}
	f.next++
	f.registered[f.next] = c
	return f.next, nil
}

func (f *fakeStreams) RegisterReader(rc io.ReadCloser) (uint32, error) { return f.register(rc) }

func (f *fakeStreams) RegisterWriter(wc io.WriteCloser) (uint32, error) { return f.register(wc) }

func (f *fakeStreams) TakeReader(h uint32) (io.ReadCloser, error) {
	rc, ok := f.readers[h]
	if !ok {
		return nil, errors.New("unknown stream handle")
	}
	delete(f.readers, h)
	return rc, nil
}

func (f *fakeStreams) TakeWriter(h uint32) (io.WriteCloser, error) {
	wc, ok := f.writers[h]
	if !ok {
		return nil, errors.New("unknown stream handle")
	}
	delete(f.writers, h)
	return wc, nil
}

func (f *fakeStreams) Release(h uint32) {
	if c, ok := f.registered[h]; ok {
		delete(f.registered, h)
		_ = c.Close()
	}
}

var _ = Describe("Conv", func() {
	var (
		ctx context.Context
		fs  *fakeStreams
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		fs = newFakeStreams()
		ctx = WithStreams(ctx, fs)
	})

	It("carries the call's streams in ctx", func() {
		got, ok := From(ctx)
		Expect(ok).To(BeTrue())
		Expect(got).To(BeIdenticalTo(fs))
		_, ok = From(GinkgoT().Context())
		Expect(ok).To(BeFalse())
	})

	It("maps handle 0 to nil and nil to handle 0", func() {
		conv := NewConv(ctx)
		Expect(conv.Reader(0)).To(BeNil())
		Expect(conv.Writer(0)).To(BeNil())
		Expect(conv.ReaderHandle(nil)).To(BeZero())
		Expect(conv.WriterHandle(nil)).To(BeZero())
		Expect(conv.Err()).To(Succeed())
	})

	It("takes input streams and registers output streams", func() {
		in := &closeCounter{Reader: strings.NewReader("in")}
		fs.readers[7] = in
		conv := NewConv(ctx)
		Expect(conv.Reader(7)).To(BeIdenticalTo(in))
		Expect(conv.Err()).To(Succeed())

		out := &closeCounter{Reader: strings.NewReader("out")}
		h := conv.ReaderHandle(out)
		Expect(conv.Err()).To(Succeed())
		Expect(fs.registered).To(HaveKeyWithValue(h, BeIdenticalTo(out)))
		Expect(in.closes.Load()).To(BeZero())
		Expect(out.closes.Load()).To(BeZero())
	})

	It("closes streams it already took when a later handle fails", func() {
		in := &closeCounter{Reader: strings.NewReader("x")}
		fs.readers[1] = in
		conv := NewConv(ctx)
		Expect(conv.Reader(1)).ToNot(BeNil())
		Expect(conv.Reader(1)).To(BeNil()) // same handle twice: already taken
		Expect(conv.Err()).To(MatchError("unknown stream handle"))
		Expect(in.closes.Load()).To(Equal(int32(1)))
	})

	It("hands taken streams to the service once Err returns nil", func() {
		in := &closeCounter{Reader: strings.NewReader("x")}
		fs.readers[1] = in
		conv := NewConv(ctx)
		conv.Reader(1)
		Expect(conv.Err()).To(Succeed())
		conv.Fail(errors.New("later failure"))
		Expect(conv.Err()).To(MatchError("later failure"))
		Expect(in.closes.Load()).To(BeZero())
	})

	It("rolls back outputs already registered when a later one fails", func() {
		fs.failAfter = 1
		first := &closeCounter{Reader: strings.NewReader("a")}
		second := &closeCounter{Writer: io.Discard}
		third := &closeCounter{Reader: strings.NewReader("c")}
		conv := NewConv(ctx)
		Expect(conv.ReaderHandle(first)).ToNot(BeZero())
		Expect(conv.WriterHandle(second)).To(BeZero())
		Expect(conv.ReaderHandle(third)).To(BeZero())
		Expect(conv.Err()).To(MatchError("too many open streams"))
		Expect(first.closes.Load()).To(Equal(int32(1)), "registered, then released")
		Expect(second.closes.Load()).To(Equal(int32(1)), "closed by the failed registration")
		Expect(third.closes.Load()).To(Equal(int32(1)), "closed without registering")
		Expect(fs.registered).To(BeEmpty())
	})

	It("closes streams a service returned together with an error", func() {
		out := &closeCounter{Reader: strings.NewReader("a")}
		conv := NewConv(ctx)
		conv.Fail(errors.New("service failed"))
		Expect(conv.ReaderHandle(out)).To(BeZero())
		Expect(conv.Err()).To(MatchError("service failed"))
		Expect(out.closes.Load()).To(Equal(int32(1)))
		Expect(fs.registered).To(BeEmpty())
	})

	It("fails outside a plugin call and closes what it is given", func() {
		conv := NewConv(GinkgoT().Context())
		out := &closeCounter{Reader: strings.NewReader("a")}
		Expect(conv.Reader(3)).To(BeNil())
		Expect(conv.ReaderHandle(out)).To(BeZero())
		Expect(conv.Err()).To(MatchError("no plugin call in progress"))
		Expect(out.closes.Load()).To(Equal(int32(1)))
	})
})
