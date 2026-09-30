package plugins

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type streamsInput struct {
	Handle uint32 `json:"handle,omitempty"`
	Dst    uint32 `json:"dst,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Chunk  int    `json:"chunk,omitempty"`
}

type streamsOutput struct {
	Count int64  `json:"count"`
	SHA   string `json:"sha"`
	Data  string `json:"data"`
	Error string `json:"error"`
}

// patternReader produces the same bytes as the test plugin's write_pattern: byte(offset % 251).
type patternReader struct{ pos, size int64 }

func (r *patternReader) Read(p []byte) (int, error) {
	if r.pos >= r.size {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), r.size-r.pos))
	for i := range n {
		p[i] = byte((r.pos + int64(i)) % 251)
	}
	r.pos += int64(n)
	return n, nil
}

func patternSHA(size int64) string {
	h := sha256.New()
	_, _ = io.Copy(h, &patternReader{size: size})
	return hex.EncodeToString(h.Sum(nil))
}

type hashWriter struct {
	mu     sync.Mutex
	h      hash.Hash
	n      int64
	closes atomic.Int32
}

func newHashWriter() *hashWriter { return &hashWriter{h: sha256.New()} }

func (w *hashWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.n += int64(len(p))
	return w.h.Write(p)
}

func (w *hashWriter) Close() error { w.closes.Add(1); return nil }

func (w *hashWriter) sum() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return hex.EncodeToString(w.h.Sum(nil))
}

func readInput(r io.ReadCloser, chunk int) func(*callScope) (streamsInput, error) {
	return func(s *callScope) (streamsInput, error) {
		h, err := s.register(newReadStream(r))
		return streamsInput{Handle: h, Chunk: chunk}, err
	}
}

var _ = Describe("Plugin streams", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		manager, _ := createTestManagerWithPlugins(nil, "test-streams"+PackageExtension)
		p = loadedTestPlugin(manager, "test-streams")
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	call := func(fn string, prepare func(*callScope) (streamsInput, error)) streamsOutput {
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, fn, defaultTimeout, prepare, nil)
		Expect(err).ToNot(HaveOccurred())
		return out
	}

	It("reads a host stream in small chunks", func() {
		text := "the quick brown fox jumps over the lazy dog"
		out := call("read_all", readInput(io.NopCloser(strings.NewReader(text)), 7))
		Expect(out.Error).To(BeEmpty())
		Expect(out.Data).To(Equal(text))
		Expect(out.Count).To(Equal(int64(len(text))))
	})

	It("reads a stream larger than one buffer", func() {
		data := make([]byte, 5<<20)
		_, _ = rand.Read(data)
		sum := sha256.Sum256(data)
		out := call("read_all", readInput(io.NopCloser(strings.NewReader(string(data))), 64<<10))
		Expect(out.Error).To(BeEmpty())
		Expect(out.Count).To(Equal(int64(len(data))))
		Expect(out.SHA).To(Equal(hex.EncodeToString(sum[:])))
	})

	It("writes to a host stream and closes it", func() {
		w := newHashWriter()
		const size = 3 << 20
		out := call("write_pattern", func(s *callScope) (streamsInput, error) {
			h, err := s.register(newWriteStream(w))
			return streamsInput{Handle: h, Size: size, Chunk: 100_000}, err
		})
		Expect(out.Error).To(BeEmpty())
		Expect(out.Count).To(Equal(int64(size)))
		Expect(w.sum()).To(Equal(patternSHA(size)))
		Expect(w.closes.Load()).To(Equal(int32(1)))
	})

	It("copies between host streams", func() {
		w := &trackingWriter{}
		out := call("copy", func(s *callScope) (streamsInput, error) {
			src, err := s.register(newReadStream(io.NopCloser(strings.NewReader("copied inside the host"))))
			if err != nil {
				return streamsInput{}, err
			}
			dst, err := s.register(newWriteStream(w))
			return streamsInput{Handle: src, Dst: dst}, err
		})
		Expect(out.Error).To(BeEmpty())
		Expect(out.Count).To(Equal(int64(22)))
		Expect(string(w.bytes())).To(Equal("copied inside the host"))
	})

	It("delivers the data before the error that came with it", func() {
		r := io.NopCloser(&dataErrReader{data: []byte("abc"), err: io.ErrUnexpectedEOF})
		out := call("read_all", readInput(r, 16))
		Expect(out.Data).To(Equal("abc"))
		Expect(out.Error).To(Equal(io.ErrUnexpectedEOF.Error()))
	})

	It("refuses to read a write-only stream", func() {
		out := call("read_all", func(s *callScope) (streamsInput, error) {
			h, err := s.register(newWriteStream(&trackingWriter{}))
			return streamsInput{Handle: h, Chunk: 8}, err
		})
		Expect(out.Error).To(Equal("stream is not readable"))
	})

	It("forgets a handle once the plugin closed it", func() {
		out := call("close_twice", readInput(io.NopCloser(strings.NewReader("x")), 1))
		Expect(out.Error).To(Equal("|unknown stream handle"))
	})

	It("fails cleanly when a stream function runs outside a plugin call", func() {
		instance, err := p.instance(ctx)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { instance.Close(ctx) })

		exit, raw, err := instance.Call("read_all", []byte(`{"handle":1,"chunk":8}`))
		Expect(err).ToNot(HaveOccurred())
		Expect(exit).To(Equal(uint32(0)))
		var out streamsOutput
		Expect(json.Unmarshal(raw, &out)).To(Succeed())
		Expect(out.Error).To(Equal("stream operation failed"))
	})
})

// enteredReader signals when its first Read starts, so tests act only once the host is blocked in I/O.
type enteredReader struct {
	*io.PipeReader
	entered chan struct{}
	once    sync.Once
}

func (r *enteredReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.entered) })
	return r.PipeReader.Read(p)
}

var _ = Describe("Plugin stream lifecycle", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		manager, _ := createTestManagerWithPlugins(nil, "test-streams"+PackageExtension)
		p = loadedTestPlugin(manager, "test-streams")
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("closes a handle the plugin never closed", func() {
		r := &trackingReader{Reader: strings.NewReader("xyz")}
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "leak", defaultTimeout, readInput(r, 1), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Count).To(Equal(int64(1)))
		Expect(r.closed()).To(BeTrue())
	})

	type callResult struct {
		out streamsOutput
		err error
	}

	// startRead runs read_all on r in the background; every wait on it below is bounded.
	startRead := func(callCtx context.Context, timeout time.Duration, prepare func(*callScope) (streamsInput, error)) chan callResult {
		results := make(chan callResult, 1)
		go func() {
			out, err := callPluginFunctionScoped[streamsInput, streamsOutput](callCtx, p, "read_all", timeout, prepare, nil)
			results <- callResult{out: out, err: err}
		}()
		return results
	}

	newEnteredPipe := func() (*enteredReader, *io.PipeWriter) {
		pr, pw := io.Pipe()
		DeferCleanup(func() {
			_ = pw.Close()
			_ = pr.Close()
		})
		return &enteredReader{PipeReader: pr, entered: make(chan struct{})}, pw
	}

	It("does not let one call use another call's handle", func() {
		r, pw := newEnteredPipe()
		handles := make(chan uint32, 1)
		results := startRead(ctx, defaultTimeout, func(s *callScope) (streamsInput, error) {
			h, err := s.register(newReadStream(r))
			handles <- h
			return streamsInput{Handle: h, Chunk: 16}, err
		})
		var h uint32
		Eventually(handles).WithTimeout(10 * time.Second).Should(Receive(&h))
		Eventually(r.entered).WithTimeout(10 * time.Second).Should(BeClosed())

		other, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "read_all", defaultTimeout,
			func(*callScope) (streamsInput, error) { return streamsInput{Handle: h, Chunk: 16}, nil }, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(other.Error).To(Equal("unknown stream handle"))

		_, err = pw.Write([]byte("mine"))
		Expect(err).ToNot(HaveOccurred())
		Expect(pw.Close()).To(Succeed())
		var res callResult
		Eventually(results).WithTimeout(10 * time.Second).Should(Receive(&res))
		Expect(res.err).ToNot(HaveOccurred())
		Expect(res.out.Data).To(Equal("mine"))
	})

	It("unblocks a read when the call is cancelled", func() {
		r, pw := newEnteredPipe()
		callCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		results := startRead(callCtx, defaultTimeout, readInput(r, 16))
		Eventually(r.entered).WithTimeout(10 * time.Second).Should(BeClosed())

		cancel()
		var res callResult
		Eventually(results).WithTimeout(5 * time.Second).Should(Receive(&res))
		// The plugin may finish its export before wazero sees the cancel, so a clean return is also valid.
		Expect(res.err).To(Or(Not(HaveOccurred()), MatchError(context.Canceled)))
		_, err := pw.Write([]byte("x"))
		Expect(err).To(MatchError(io.ErrClosedPipe))
	})

	It("unblocks a read when the time limit passes", func() {
		r, pw := newEnteredPipe()
		results := startRead(ctx, 2*time.Second, readInput(r, 16))
		Eventually(r.entered).WithTimeout(10 * time.Second).Should(BeClosed())

		var res callResult
		Eventually(results).WithTimeout(5 * time.Second).Should(Receive(&res))
		Expect(res.err).To(Or(Not(HaveOccurred()), MatchError(context.DeadlineExceeded)))
		_, err := pw.Write([]byte("x"))
		Expect(err).To(MatchError(io.ErrClosedPipe))
	})
})

var _ = Describe("Plugin streams under a memory cap", Ordered, func() {
	// Each memory (the plugin's and Extism's block memory) is capped below the bytes moved, so any per-chunk
	// allocation that is not freed runs out of memory.
	const (
		memoryPages  = 1024     // 64 MiB
		transferSize = 96 << 20 // more than the cap
	)

	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		extismMaxMemoryPages = memoryPages
		DeferCleanup(func() { extismMaxMemoryPages = 0 })
		manager, _ := createTestManagerWithPlugins(nil, "test-streams"+PackageExtension)
		p = loadedTestPlugin(manager, "test-streams")
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("reads more than the cap through stream_read", func() {
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "read_all", 2*time.Minute,
			readInput(io.NopCloser(&patternReader{size: transferSize}), 64<<10), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Count).To(Equal(int64(transferSize)))
		Expect(out.SHA).To(Equal(patternSHA(transferSize)))
	})

	It("writes more than the cap through stream_write", func() {
		w := newHashWriter()
		out, err := callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, "write_pattern", 2*time.Minute,
			func(s *callScope) (streamsInput, error) {
				h, err := s.register(newWriteStream(w))
				return streamsInput{Handle: h, Size: transferSize, Chunk: 64 << 10}, err
			}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(w.sum()).To(Equal(patternSHA(transferSize)))
	})
})
