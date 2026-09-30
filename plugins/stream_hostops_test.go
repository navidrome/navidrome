package plugins

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeGuestMemory models Extism blocks: an offset is valid only at the start of a block.
type fakeGuestMemory struct {
	blocks      map[uint64][]byte
	next        uint64
	failWrites  bool
	zeroOffsets bool
}

func newFakeGuestMemory() *fakeGuestMemory {
	return &fakeGuestMemory{blocks: map[uint64][]byte{}, next: 64}
}

func (m *fakeGuestMemory) alloc(size int) uint64 {
	off := m.next
	m.blocks[off] = make([]byte, size)
	m.next += uint64(size) + 8
	return off
}

func (m *fakeGuestMemory) blockLength(offset uint64) uint64 { return uint64(len(m.blocks[offset])) }

func (m *fakeGuestMemory) view(offset uint64, n uint32) ([]byte, bool) {
	b, ok := m.blocks[offset]
	if !ok || int(n) > len(b) {
		return nil, false
	}
	return b[:n], true
}

func (m *fakeGuestMemory) writeBlock(data []byte) (uint64, error) {
	if m.failWrites {
		return 0, errors.New("out of memory")
	}
	if m.zeroOffsets {
		return 0, nil
	}
	off := m.alloc(len(data))
	copy(m.blocks[off], data)
	return off, nil
}

var _ = Describe("callScope host operations", func() {
	var (
		ctx   context.Context
		scope *callScope
		mem   *fakeGuestMemory
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		var err error
		scope, err = newStreamTable().newScope(ctx, ctx)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(scope.close)
		mem = newFakeGuestMemory()
	})

	register := func(b *streamBackend) int64 {
		h, err := scope.register(b)
		Expect(err).ToNot(HaveOccurred())
		return int64(h)
	}
	lastError := func() string {
		off := scope.hostError(mem)
		if off <= 0 {
			return ""
		}
		return string(mem.blocks[uint64(off)])
	}
	readerOf := func(s string) *streamBackend { return newReadStream(io.NopCloser(strings.NewReader(s))) }

	Describe("hostRead", func() {
		It("places bytes at the start of the buffer", func() {
			h := register(readerOf("hello"))
			buf := mem.alloc(16)
			Expect(scope.hostRead(mem, h, buf, 16)).To(Equal(int64(5)))
			Expect(string(mem.blocks[buf][:5])).To(Equal("hello"))
			Expect(lastError()).To(BeEmpty())
		})

		It("reads at most len bytes", func() {
			h := register(readerOf("hello"))
			Expect(scope.hostRead(mem, h, mem.alloc(16), 2)).To(Equal(int64(2)))
		})

		It("reads at most 1 MiB per call", func() {
			h := register(newReadStream(io.NopCloser(bytes.NewReader(make([]byte, 3<<20)))))
			Expect(scope.hostRead(mem, h, mem.alloc(2<<20), 2<<20)).To(Equal(int64(maxStreamChunk)))
		})

		It("returns 0 at the end of the stream", func() {
			h := register(readerOf(""))
			Expect(scope.hostRead(mem, h, mem.alloc(8), 8)).To(Equal(int64(0)))
			Expect(lastError()).To(BeEmpty())
		})

		It("returns the bytes first, then the error that came with them", func() {
			h := register(newReadStream(io.NopCloser(&dataErrReader{data: []byte("abc"), err: errors.New("disk gone")})))
			buf := mem.alloc(16)
			Expect(scope.hostRead(mem, h, buf, 16)).To(Equal(int64(3)))
			Expect(scope.hostRead(mem, h, buf, 16)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal("disk gone"))
		})

		It("gives up on a reader that never makes progress", func() {
			h := register(newReadStream(io.NopCloser(noProgressReader{})))
			Expect(scope.hostRead(mem, h, mem.alloc(8), 8)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(io.ErrNoProgress.Error()))
		})

		It("rejects len 0, len larger than the block, and an offset inside a block", func() {
			h := register(readerOf("x"))
			buf := mem.alloc(8)
			Expect(scope.hostRead(mem, h, buf, 0)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(errBadBuffer.Error()))
			Expect(scope.hostRead(mem, h, buf, 9)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(errBadBuffer.Error()))
			Expect(scope.hostRead(mem, h, buf+1, 4)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(errBadBuffer.Error()))
		})

		It("rejects handles outside 1 to 2^31-1", func() {
			buf := mem.alloc(8)
			for _, h := range []int64{0, -1, math.MaxInt32 + 1} {
				Expect(scope.hostRead(mem, h, buf, 8)).To(Equal(int64(-1)))
				Expect(lastError()).To(Equal(errUnknownStream.Error()))
			}
		})

		It("refuses to read a write-only stream", func() {
			h := register(newWriteStream(&trackingWriter{}))
			Expect(scope.hostRead(mem, h, mem.alloc(8), 8)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(errStreamNotReadable.Error()))
		})
	})

	Describe("hostWrite", func() {
		It("writes only the first len bytes of the buffer", func() {
			w := &trackingWriter{}
			h := register(newWriteStream(w))
			buf := mem.alloc(16)
			copy(mem.blocks[buf], "hello, old bytes")
			Expect(scope.hostWrite(mem, h, buf, 5)).To(Equal(int64(5)))
			Expect(string(w.bytes())).To(Equal("hello"))
			Expect(lastError()).To(BeEmpty())
		})

		It("reports a short write as a count plus a retained error", func() {
			h := register(newWriteStream(&trackingWriter{limit: 2}))
			buf := mem.alloc(8)
			Expect(scope.hostWrite(mem, h, buf, 4)).To(Equal(int64(2)))
			Expect(lastError()).To(Equal(io.ErrShortWrite.Error()))
			Expect(scope.hostWrite(mem, h, buf, 4)).To(Equal(int64(-1)))
		})

		It("refuses to write a read-only stream", func() {
			h := register(readerOf("x"))
			Expect(scope.hostWrite(mem, h, mem.alloc(8), 8)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(errStreamNotWritable.Error()))
		})
	})

	Describe("hostCopy", func() {
		It("copies everything and clears the last error", func() {
			w := &trackingWriter{}
			dst := register(newWriteStream(w))
			src := register(readerOf("copy me"))
			Expect(scope.hostCopy(dst, src)).To(Equal(int64(7)))
			Expect(string(w.bytes())).To(Equal("copy me"))
			Expect(lastError()).To(BeEmpty())
		})

		It("returns the bytes copied before an error", func() {
			w := &trackingWriter{}
			dst := register(newWriteStream(w))
			src := register(newReadStream(io.NopCloser(&dataErrReader{data: []byte("abc"), err: errors.New("net down")})))
			Expect(scope.hostCopy(dst, src)).To(Equal(int64(3)))
			Expect(lastError()).To(Equal("net down"))
		})

		It("checks the rights of both handles", func() {
			a := register(readerOf("x"))
			b := register(readerOf("y"))
			Expect(scope.hostCopy(a, b)).To(Equal(int64(0)))
			Expect(lastError()).To(Equal(errStreamNotWritable.Error()))
		})

		It("stops copying from a reader that never makes progress", func() {
			dst := register(newWriteStream(&trackingWriter{}))
			src := register(newReadStream(io.NopCloser(noProgressReader{})))
			Expect(scope.hostCopy(dst, src)).To(Equal(int64(0)))
			Expect(lastError()).To(Equal(io.ErrNoProgress.Error()))
		})
	})

	Describe("hostError", func() {
		It("returns -1, not 0, when the message cannot be allocated", func() {
			Expect(scope.hostClose(99)).To(Equal(int64(-1)))
			mem.failWrites = true
			Expect(scope.hostError(mem)).To(Equal(int64(-1)))
		})

		It("returns -1 when the message lands at offset 0", func() {
			Expect(scope.hostClose(99)).To(Equal(int64(-1)))
			mem.zeroOffsets = true
			Expect(scope.hostError(mem)).To(Equal(int64(-1)))
		})

		It("never hands out an empty message", func() {
			h := register(newReadStream(io.NopCloser(&dataErrReader{err: errors.New("")})))
			Expect(scope.hostRead(mem, h, mem.alloc(4), 4)).To(Equal(int64(-1)))
			off := scope.hostError(mem)
			Expect(off).To(BeNumerically(">", 0))
			Expect(string(mem.blocks[uint64(off)])).To(Equal("stream operation failed"))
		})

		It("returns 0 after a successful operation", func() {
			h := register(readerOf("x"))
			Expect(scope.hostRead(mem, h, mem.alloc(4), 4)).To(Equal(int64(1)))
			Expect(scope.hostError(mem)).To(BeZero())
		})
	})

	Describe("hostClose", func() {
		It("closes the stream and forgets the handle", func() {
			r := &trackingReader{Reader: strings.NewReader("x")}
			h := register(newReadStream(r))
			Expect(scope.hostClose(h)).To(Equal(int64(0)))
			Expect(r.closed()).To(BeTrue())

			Expect(scope.hostRead(mem, h, mem.alloc(4), 4)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal(errUnknownStream.Error()))
			Expect(scope.hostClose(h)).To(Equal(int64(-1)))
		})

		It("reports the close error", func() {
			h := register(newReadStream(failingCloseReader{strings.NewReader("")}))
			Expect(scope.hostClose(h)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal("close failed"))
		})

		It("reports a write error that came with a full write", func() {
			h := register(newWriteStream(&errWriter{accept: math.MaxInt, err: errors.New("flush failed")}))
			buf := mem.alloc(4)
			Expect(scope.hostWrite(mem, h, buf, 4)).To(Equal(int64(4)))
			Expect(scope.hostClose(h)).To(Equal(int64(-1)))
			Expect(lastError()).To(Equal("flush failed"))
		})

		It("closes a read stream cleanly after an error", func() {
			h := register(newReadStream(io.NopCloser(&dataErrReader{err: errors.New("net down")})))
			Expect(scope.hostRead(mem, h, mem.alloc(4), 4)).To(Equal(int64(-1)))
			Expect(scope.hostClose(h)).To(Equal(int64(0)))
		})
	})
})
