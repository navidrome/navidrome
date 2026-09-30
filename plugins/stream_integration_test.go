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
