package plugins

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

var patternCycle = func() []byte {
	b := make([]byte, 251)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}()

// patternReader produces the same bytes as the test plugin's write_pattern: byte(offset % 251).
type patternReader struct{ pos, size int64 }

func (r *patternReader) Read(p []byte) (int, error) {
	if r.pos >= r.size {
		return 0, io.EOF
	}
	p = p[:min(int64(len(p)), r.size-r.pos)]
	for i := 0; i < len(p); {
		i += copy(p[i:], patternCycle[(r.pos+int64(i))%251:])
	}
	r.pos += int64(len(p))
	return len(p), nil
}

func patternSHA(size int64) string {
	h := sha256.New()
	_, _ = io.Copy(h, &patternReader{size: size})
	return hex.EncodeToString(h.Sum(nil))
}

type hashWriter struct {
	mu     sync.Mutex
	h      hash.Hash
	closes atomic.Int32
}

func newHashWriter() *hashWriter { return &hashWriter{h: sha256.New()} }

func (w *hashWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
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

func writeInput(w io.WriteCloser, size int64, chunk int) func(*callScope) (streamsInput, error) {
	return func(s *callScope) (streamsInput, error) {
		h, err := s.register(newWriteStream(w))
		return streamsInput{Handle: h, Size: size, Chunk: chunk}, err
	}
}

func copyInput(r io.ReadCloser, w io.WriteCloser) func(*callScope) (streamsInput, error) {
	return func(s *callScope) (streamsInput, error) {
		src, err := s.register(newReadStream(r))
		if err != nil {
			return streamsInput{}, err
		}
		dst, err := s.register(newWriteStream(w))
		return streamsInput{Handle: src, Dst: dst}, err
	}
}

func callStreams(ctx context.Context, p *plugin, fn string, timeout time.Duration,
	prepare func(*callScope) (streamsInput, error)) (streamsOutput, error) {
	return callPluginFunctionScoped[streamsInput, streamsOutput](ctx, p, fn, timeout, prepare, nil)
}

// errOnlyCtx reports Canceled from Err once stopped, but never closes Done, so wazero does not see the stop.
type errOnlyCtx struct {
	context.Context
	stopped atomic.Bool
}

func (c *errOnlyCtx) Err() error {
	if c.stopped.Load() {
		return context.Canceled
	}
	return c.Context.Err()
}

type stopAtEOF struct {
	io.Reader
	ctx *errOnlyCtx
}

func (r stopAtEOF) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if errors.Is(err, io.EOF) {
		r.ctx.stopped.Store(true)
	}
	return n, err
}

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

	Describe("reading and writing", func() {
		call := func(fn string, prepare func(*callScope) (streamsInput, error)) streamsOutput {
			out, err := callStreams(ctx, p, fn, defaultTimeout, prepare)
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
			data := make([]byte, 256<<10)
			_, _ = rand.Read(data)
			sum := sha256.Sum256(data)
			out := call("read_all", readInput(io.NopCloser(bytes.NewReader(data)), 64<<10))
			Expect(out.Error).To(BeEmpty())
			Expect(out.Count).To(Equal(int64(len(data))))
			Expect(out.SHA).To(Equal(hex.EncodeToString(sum[:])))
		})

		It("writes to a host stream and closes it", func() {
			w := newHashWriter()
			const size = 512 << 10
			out := call("write_pattern", writeInput(w, size, 100_000))
			Expect(out.Error).To(BeEmpty())
			Expect(out.Count).To(Equal(int64(size)))
			Expect(w.sum()).To(Equal(patternSHA(size)))
			Expect(w.closes.Load()).To(Equal(int32(1)))
		})

		It("copies between host streams", func() {
			w := &trackingWriter{}
			out := call("copy", copyInput(io.NopCloser(strings.NewReader("copied inside the host")), w))
			Expect(out.Error).To(BeEmpty())
			Expect(out.Count).To(Equal(int64(22)))
			Expect(string(w.bytes())).To(Equal("copied inside the host"))
		})

		It("reports a short write as the count and the writer's error", func() {
			out := call("write_pattern", writeInput(&errWriter{accept: 10, err: errors.New("disk full")}, 1000, 100))
			Expect(out.Count).To(Equal(int64(10)))
			Expect(out.Error).To(Equal("disk full"))
		})

		It("delivers the data before the error that came with it", func() {
			r := io.NopCloser(&dataErrReader{data: []byte("abc"), err: io.ErrUnexpectedEOF})
			out := call("read_all", readInput(r, 16))
			Expect(out.Data).To(Equal("abc"))
			Expect(out.Error).To(Equal(io.ErrUnexpectedEOF.Error()))
		})

		It("refuses to read a write-only stream", func() {
			out := call("read_all", writeInput(&trackingWriter{}, 0, 8))
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

			exit, raw, err := instance.CallWithContext(ctx, "read_all", []byte(`{"handle":1,"chunk":8}`))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))
			var out streamsOutput
			Expect(json.Unmarshal(raw, &out)).To(Succeed())
			Expect(out.Error).To(Equal("stream operation failed"))
		})
	})

	Describe("lifecycle", func() {
		It("closes a handle the plugin never closed", func() {
			r := &trackingReader{Reader: strings.NewReader("xyz")}
			out, err := callStreams(ctx, p, "leak", defaultTimeout, readInput(r, 1))
			Expect(err).ToNot(HaveOccurred())
			Expect(out.Count).To(Equal(int64(1)))
			Expect(r.closed()).To(BeTrue())
		})

		type callResult struct {
			out streamsOutput
			err error
		}

		// startCall runs the export fn in the background; every wait on it below is bounded.
		startCall := func(fn string, callCtx context.Context, timeout time.Duration, prepare func(*callScope) (streamsInput, error)) chan callResult {
			results := make(chan callResult, 1)
			go func() {
				out, err := callStreams(callCtx, p, fn, timeout, prepare)
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

		expectUnblocked := func(r *enteredReader, pw *io.PipeWriter, results chan callResult, stop func(), wantErr error) {
			Eventually(r.entered).WithTimeout(10 * time.Second).Should(BeClosed())
			stop()
			var res callResult
			Eventually(results).WithTimeout(5 * time.Second).Should(Receive(&res))
			Expect(res.err).To(MatchError(wantErr))
			_, err := pw.Write([]byte("x"))
			Expect(err).To(MatchError(io.ErrClosedPipe))
		}

		It("does not let one call use another call's handle", func() {
			r, pw := newEnteredPipe()
			handles := make(chan uint32, 1)
			results := startCall("read_all", ctx, defaultTimeout, func(s *callScope) (streamsInput, error) {
				h, err := s.register(newReadStream(r))
				handles <- h
				return streamsInput{Handle: h, Chunk: 16}, err
			})
			var h uint32
			Eventually(handles).WithTimeout(10 * time.Second).Should(Receive(&h))
			Eventually(r.entered).WithTimeout(10 * time.Second).Should(BeClosed())

			otherResults := startCall("read_all", ctx, defaultTimeout,
				func(*callScope) (streamsInput, error) { return streamsInput{Handle: h, Chunk: 16}, nil })
			var other callResult
			Eventually(otherResults).WithTimeout(10 * time.Second).Should(Receive(&other))
			Expect(other.err).ToNot(HaveOccurred())
			Expect(other.out.Error).To(Equal("unknown stream handle"))

			_, err := pw.Write([]byte("mine"))
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
			results := startCall("read_all", callCtx, defaultTimeout, readInput(r, 16))
			expectUnblocked(r, pw, results, cancel, context.Canceled)
		})

		It("unblocks a read when the time limit passes", func() {
			r, pw := newEnteredPipe()
			results := startCall("read_all", ctx, 2*time.Second, readInput(r, 16))
			expectUnblocked(r, pw, results, func() {}, context.DeadlineExceeded)
		})

		It("reports a cancel even when the plugin returns cleanly", func() {
			callCtx := &errOnlyCtx{Context: ctx}
			r := io.NopCloser(stopAtEOF{Reader: strings.NewReader("cut off"), ctx: callCtx})
			_, err := callStreams(callCtx, p, "read_all", defaultTimeout, readInput(r, 16))
			Expect(err).To(MatchError(context.Canceled))
		})

		It("unblocks a stream_copy when the call is cancelled", func() {
			r, pw := newEnteredPipe()
			callCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			results := startCall("copy", callCtx, defaultTimeout, copyInput(r, &trackingWriter{}))
			expectUnblocked(r, pw, results, cancel, context.Canceled)
		})
	})
})

var _ = Describe("Plugin streams under a memory cap", Ordered, func() {
	// Each memory (the plugin's and Extism's block memory) is capped below the bytes moved, so any per-chunk
	// allocation that is not freed runs out of memory.
	const (
		memoryPages  = 1024 // 64 MiB
		transferSize = 96 << 20
	)

	var (
		ctx     context.Context
		p       *plugin
		wantSHA string
	)

	BeforeAll(func() {
		extismMaxMemoryPages = memoryPages
		DeferCleanup(func() { extismMaxMemoryPages = 0 })
		manager, _ := createTestManagerWithPlugins(nil, "test-streams"+PackageExtension)
		p = loadedTestPlugin(manager, "test-streams")
		wantSHA = patternSHA(transferSize)
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("reads more than the cap through stream_read", func() {
		out, err := callStreams(ctx, p, "read_all", 2*time.Minute,
			readInput(io.NopCloser(&patternReader{size: transferSize}), 64<<10))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Count).To(Equal(int64(transferSize)))
		Expect(out.SHA).To(Equal(wantSHA))
	})

	It("writes more than the cap through stream_write", func() {
		w := newHashWriter()
		out, err := callStreams(ctx, p, "write_pattern", 2*time.Minute, writeInput(w, transferSize, 64<<10))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(w.sum()).To(Equal(wantSHA))
	})

	It("fails a write cleanly when the stream buffer cannot be allocated", func() {
		w := &trackingWriter{}
		out, err := callStreams(ctx, p, "write_when_out_of_memory", defaultTimeout, writeInput(w, 0, 0))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(Equal("stream buffer allocation failed"))
		Expect(w.bytes()).To(BeEmpty())
	})
})

// buildRustTestPlugin skips when the wasm32-wasip1 Rust target is missing; CI does not install it.
func buildRustTestPlugin(dir string) string {
	rustup, err := exec.LookPath("rustup")
	if err != nil {
		Skip("rustup is not installed")
	}
	targets, err := exec.CommandContext(GinkgoT().Context(), rustup, "target", "list", "--installed").Output()
	if err != nil || !strings.Contains(string(targets), "wasm32-wasip1") {
		Skip("the wasm32-wasip1 Rust target is not installed")
	}
	cargoOut, err := exec.CommandContext(GinkgoT().Context(), rustup, "which", "cargo").Output()
	Expect(err).ToNot(HaveOccurred())
	rustc, err := exec.CommandContext(GinkgoT().Context(), rustup, "which", "rustc").Output()
	Expect(err).ToNot(HaveOccurred())

	cargo := strings.TrimSpace(string(cargoOut))
	cmd := exec.CommandContext(GinkgoT().Context(), cargo, "build", "--release", "--target", "wasm32-wasip1")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CARGO_BUILD_RUSTC="+strings.TrimSpace(string(rustc)))
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred(), string(out))

	name := filepath.Base(dir)
	wasm := filepath.Join(dir, "target", "wasm32-wasip1", "release", strings.ReplaceAll(name, "-", "_")+".wasm")
	pkg := name + PackageExtension
	Expect(packageFiles(filepath.Join(testdataDir, pkg), filepath.Join(dir, manifestFileName), wasm)).To(Succeed())
	return pkg
}

var _ = Describe("Rust plugin streams", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		pkg := buildRustTestPlugin(filepath.Join(testdataDir, "test-streams-rs"))
		extismMaxMemoryPages = 1024 // 64 MiB, so the out-of-memory spec can fill it
		DeferCleanup(func() { extismMaxMemoryPages = 0 })
		manager, _ := createTestManagerWithPlugins(nil, pkg)
		p = loadedTestPlugin(manager, "test-streams-rs")
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("reads a host stream with std::io::Read", func() {
		out, err := callStreams(ctx, p, "read_all", defaultTimeout,
			readInput(io.NopCloser(strings.NewReader("streamed to rust")), 0))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Data).To(Equal("streamed to rust"))
	})

	It("reads into a slice that starts at an odd address", func() {
		out, err := callStreams(ctx, p, "read_unaligned", defaultTimeout,
			readInput(io.NopCloser(strings.NewReader("misaligned bytes here")), 0))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Data).To(Equal("misaligned bytes here"))
	})

	It("writes from a slice that starts at an odd address", func() {
		w := &trackingWriter{}
		out, err := callStreams(ctx, p, "write_unaligned", defaultTimeout, writeInput(w, 0, 0))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(string(w.bytes())).To(Equal("unaligned write, 8+ bytes"))
	})

	It("fails a write cleanly when the stream buffer cannot be allocated", func() {
		w := &trackingWriter{}
		out, err := callStreams(ctx, p, "write_when_out_of_memory", defaultTimeout, writeInput(w, 0, 0))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(Equal("stream buffer allocation failed"))
		Expect(w.bytes()).To(BeEmpty())
	})

	It("writes a host stream with std::io::Write and closes it", func() {
		w := &trackingWriter{}
		out, err := callStreams(ctx, p, "write_greeting", defaultTimeout, writeInput(w, 0, 0))
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(string(w.bytes())).To(Equal("hello from rust"))
		Expect(w.closed()).To(BeTrue())
	})
})
