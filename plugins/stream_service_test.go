package plugins

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/plugins/internal/streamtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testStreamService struct {
	created *trackingWriter
	blocked *io.PipeWriter
}

func (s *testStreamService) Open(_ context.Context, text string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(text)), nil
}

func (s *testStreamService) Create(context.Context, string) (io.WriteCloser, error) {
	return s.created, nil
}

func (s *testStreamService) Drain(_ context.Context, in io.ReadCloser) (string, error) {
	if in == nil {
		return "<nil>", nil
	}
	defer in.Close()
	data, err := io.ReadAll(in)
	return string(data), err
}

func (s *testStreamService) Wrap(_ context.Context, wrap streamtest.WrapRequest) (*streamtest.WrapResponse, error) {
	defer wrap.Body.Close()
	data, err := io.ReadAll(wrap.Body)
	if err != nil {
		return nil, err
	}
	out := wrap.Prefix + string(data)
	return &streamtest.WrapResponse{Size: int64(len(out)), Body: io.NopCloser(strings.NewReader(out))}, nil
}

func (s *testStreamService) Block(context.Context) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	s.blocked = pw
	return pr, nil
}

func (s *testStreamService) Echo(_ context.Context, text string) (string, error) { return text, nil }

// streamTestSvc serves the StreamTest host functions, registered for every plugin because test-streams imports them.
var streamTestSvc streamtest.StreamTestService = &testStreamService{created: &trackingWriter{}}

func init() {
	hostServices = append(hostServices, hostServiceEntry{
		name:          "StreamTest",
		hasPermission: func(*Permissions) bool { return true },
		create: func(*serviceContext) ([]extism.HostFunction, io.Closer, error) {
			return streamtest.RegisterStreamTestHostFunctions(streamTestSvc), nil, nil
		},
	})
}

func loadStreamTestPlugin(svc *testStreamService, pkg string, memPages uint32) *plugin {
	if memPages > 0 {
		saved := extismMaxMemoryPages
		extismMaxMemoryPages = memPages
		DeferCleanup(func() { extismMaxMemoryPages = saved })
	}
	saved := streamTestSvc
	streamTestSvc = svc
	DeferCleanup(func() { streamTestSvc = saved })
	manager, _ := createTestManagerWithPlugins(nil, pkg)
	return loadedTestPlugin(manager, strings.TrimSuffix(pkg, PackageExtension))
}

func callText(ctx context.Context, p *plugin, fn, text string) serviceOutput {
	GinkgoHelper()
	out, err := callPluginFunction[map[string]string, serviceOutput](ctx, p, fn, map[string]string{"text": text})
	Expect(err).ToNot(HaveOccurred())
	return out
}

// generatedClientSpecs are the specs the Go and the Rust generated clients share.
func generatedClientSpecs(call func(fn, text string) serviceOutput) {
	It("returns a host reader to the plugin", func() {
		Expect(call("open_read", "hello")).To(Equal(serviceOutput{Data: "hello"}))
	})

	It("converts stream fields of request and response structs", func() {
		Expect(call("wrap", "abc")).To(Equal(serviceOutput{Data: "p:abc", Size: 5}))
	})
}

type serviceOutput struct {
	Data  string `json:"data"`
	Size  int64  `json:"size"`
	Error string `json:"error"`
}

var _ = Describe("Generated stream host functions", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
		svc *testStreamService
	)

	BeforeAll(func() {
		svc = &testStreamService{created: &trackingWriter{}}
		p = loadStreamTestPlugin(svc, "test-streams"+PackageExtension, 0)
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	call := func(fn, text string) serviceOutput { return callText(ctx, p, fn, text) }

	generatedClientSpecs(call)

	It("returns a host writer to the plugin", func() {
		Expect(call("create_write", "written by plugin").Error).To(BeEmpty())
		Expect(string(svc.created.bytes())).To(Equal("written by plugin"))
		Expect(svc.created.closed()).To(BeTrue())
	})

	It("hands a plugin's stream to the service and invalidates its handle", func() {
		out := call("drain_own", "abc")
		Expect(out.Data).To(Equal("abc"))
		Expect(out.Error).To(Equal("unknown stream handle"))
	})

	It("rejects a reader the plugin made itself", func() {
		Expect(call("drain_plugin_reader", "x").Error).To(Equal("only streams from Navidrome can be passed to a host function"))
	})

	It("passes no stream as nil", func() {
		Expect(call("drain_nil", "").Data).To(Equal("<nil>"))
	})

	// Runs after "returns a host writer" (Ordered): its scope closes the shared svc.created writer.
	It("checks the stream direction", func() {
		Expect(call("drain_writer", "").Error).To(Equal("stream is not readable"))
	})

	It("treats a JSON null handle as no stream", func() {
		Expect(call("drain_null_raw", "").Data).To(MatchJSON(`{"result":"<nil>"}`))
	})

	It("closes a handed-over stream when the call times out, unblocking the service", func() {
		done := make(chan error, 1)
		go func() {
			_, err := callPluginFunctionWithTimeout[map[string]string, serviceOutput](ctx, p, "drain_blocked", 250*time.Millisecond, map[string]string{})
			done <- err
		}()
		var err error
		Eventually(done).WithTimeout(5 * time.Second).Should(Receive(&err))
		Expect(err).To(Or(Not(HaveOccurred()), MatchError(context.DeadlineExceeded)))
		_, werr := svc.blocked.Write([]byte("x"))
		Expect(werr).To(MatchError(io.ErrClosedPipe))
	})
})

var _ = Describe("Generated stream host functions under a memory cap", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		// 24 MiB; drain_loop would keep 800 x 64 KiB buffers if hand-over leaked them
		p = loadStreamTestPlugin(&testStreamService{created: &trackingWriter{}}, "test-streams"+PackageExtension, 384)
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("frees a stream's buffer when the plugin hands it over", func() {
		out, err := callPluginFunctionWithTimeout[map[string]string, serviceOutput](ctx, p, "drain_loop", time.Minute, map[string]string{"text": "abc"})
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Size).To(Equal(int64(800)))
	})
})

var _ = Describe("Generated stream host functions from Rust", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		pkg := buildRustTestPlugin(filepath.Join(testdataDir, "test-streams-rs"))
		p = loadStreamTestPlugin(&testStreamService{created: &trackingWriter{}}, pkg, 0)
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	generatedClientSpecs(func(fn, text string) serviceOutput { return callText(ctx, p, fn, text) })
})

var _ = Describe("Generated Rust host calls under a memory cap", Ordered, func() {
	var (
		ctx context.Context
		p   *plugin
	)

	BeforeAll(func() {
		pkg := buildRustTestPlugin(filepath.Join(testdataDir, "test-streams-rs"))
		// 16 MiB; 500 leaked 64 KiB responses would not fit
		p = loadStreamTestPlugin(&testStreamService{created: &trackingWriter{}}, pkg, 256)
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("frees every host response", func() {
		out, err := callPluginFunctionWithTimeout[map[string]string, serviceOutput](ctx, p, "echo_loop", time.Minute, map[string]string{})
		Expect(err).ToNot(HaveOccurred())
		Expect(out.Error).To(BeEmpty())
		Expect(out.Size).To(Equal(int64(500)))
	})
})
