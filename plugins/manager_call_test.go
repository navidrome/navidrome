package plugins

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockMetricsRecorder tracks calls to RecordPluginRequest for testing
type mockMetricsRecorder struct {
	mu    sync.Mutex
	calls []metricsCall
}

type metricsCall struct {
	plugin  string
	method  string
	ok      bool
	elapsed int64
}

func (m *mockMetricsRecorder) RecordPluginRequest(_ context.Context, plugin, method string, ok bool, elapsed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, metricsCall{plugin: plugin, method: method, ok: ok, elapsed: elapsed})
}

func (m *mockMetricsRecorder) getCalls() []metricsCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]metricsCall{}, m.calls...)
}

func (m *mockMetricsRecorder) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
}

var _ = Describe("callPluginFunction metrics", Ordered, func() {
	var (
		metricsManager  *Manager
		metricsRecorder *mockMetricsRecorder
		agent           agents.Interface
	)

	BeforeAll(func() {
		metricsRecorder = &mockMetricsRecorder{}

		// Create a manager with the metrics recorder
		metricsManager, _ = createTestManagerWithPluginsAndMetrics(
			nil,
			metricsRecorder,
			"test-metadata-agent"+PackageExtension,
		)

		var ok bool
		agent, ok = metricsManager.LoadMediaAgent("test-metadata-agent")
		Expect(ok).To(BeTrue())
	})

	BeforeEach(func() {
		metricsRecorder.reset()
	})

	It("records metrics for successful plugin calls", func() {
		retriever := agent.(agents.ArtistBiographyRetriever)
		_, err := retriever.GetArtistBiography(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
		Expect(err).ToNot(HaveOccurred())

		calls := metricsRecorder.getCalls()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].plugin).To(Equal("test-metadata-agent"))
		Expect(calls[0].method).To(Equal(FuncGetArtistBiography))
		Expect(calls[0].ok).To(BeTrue())
		Expect(calls[0].elapsed).To(BeNumerically(">=", 0))
	})

	Context("with error config", Ordered, func() {
		var (
			errorRecorder *mockMetricsRecorder
			errorAgent    agents.Interface
		)

		BeforeAll(func() {
			errorRecorder = &mockMetricsRecorder{}
			errorManager, _ := createTestManagerWithPluginsAndMetrics(
				map[string]map[string]string{
					"test-metadata-agent": {"error": "simulated error"},
				},
				errorRecorder,
				"test-metadata-agent"+PackageExtension,
			)

			var ok bool
			errorAgent, ok = errorManager.LoadMediaAgent("test-metadata-agent")
			Expect(ok).To(BeTrue())
		})

		It("records metrics for failed plugin calls (error returned)", func() {
			retriever := errorAgent.(agents.ArtistBiographyRetriever)
			_, err := retriever.GetArtistBiography(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
			Expect(err).To(HaveOccurred())

			calls := errorRecorder.getCalls()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].plugin).To(Equal("test-metadata-agent"))
			Expect(calls[0].method).To(Equal(FuncGetArtistBiography))
			Expect(calls[0].ok).To(BeFalse())
		})
	})

	Context("with partial metadata agent", Ordered, func() {
		var (
			partialRecorder *mockMetricsRecorder
			partialAgent    agents.Interface
		)

		BeforeAll(func() {
			partialRecorder = &mockMetricsRecorder{}
			partialManager, _ := createTestManagerWithPluginsAndMetrics(
				nil,
				partialRecorder,
				"partial-metadata-agent"+PackageExtension,
			)

			var ok bool
			partialAgent, ok = partialManager.LoadMediaAgent("partial-metadata-agent")
			Expect(ok).To(BeTrue())
		})

		It("does not record metrics for not-implemented functions", func() {
			retriever := partialAgent.(agents.ArtistMBIDRetriever)
			_, err := retriever.GetArtistMBID(GinkgoT().Context(), "artist-1", "Test Artist")
			Expect(err).To(MatchError(errNotImplemented))

			calls := partialRecorder.getCalls()
			Expect(calls).To(BeEmpty())
		})
	})
})

func loadedTestPlugin(m *Manager, id string) *plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p := m.plugins[id]
	Expect(p).ToNot(BeNil(), "plugin %s is not loaded", id)
	return p
}

var _ = Describe("callPluginFunction time limits", Ordered, func() {
	var (
		ctx      context.Context
		p        *plugin
		recorder *mockMetricsRecorder
	)

	BeforeAll(func() {
		recorder = &mockMetricsRecorder{}
		manager, _ := createTestManagerWithPluginsAndMetrics(nil, recorder, "test-streams"+PackageExtension)
		p = loadedTestPlugin(manager, "test-streams")
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		recorder.reset()
	})

	It("stops a call when its time limit passes and records it as a failure", func() {
		start := time.Now()
		_, err := callPluginFunctionWithTimeout[struct{}, struct{}](ctx, p, "spin", 200*time.Millisecond, struct{}{})
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))

		calls := recorder.getCalls()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].plugin).To(Equal("test-streams"))
		Expect(calls[0].method).To(Equal("spin"))
		Expect(calls[0].ok).To(BeFalse())
	})

	It("leaves the time limit out of the Extism manifest", func() {
		m := buildExtismManifest(&ndpPackage{Manifest: &Manifest{}}, nil)
		Expect(m.Timeout).To(BeZero())
	})
})

var _ = Describe("callPluginFunctionScoped", Ordered, func() {
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

	It("closes streams left in the scope when the call returns", func() {
		r := &trackingReader{Reader: strings.NewReader("x")}
		_, err := callPluginFunctionScoped[streamsInput, struct{}](ctx, p, "noop", defaultTimeout, readInput(r, 0), nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(r.closed()).To(BeTrue())
	})

	It("closes input streams when the export does not exist", func() {
		r := &trackingReader{Reader: strings.NewReader("x")}
		_, err := callPluginFunctionScoped[streamsInput, struct{}](ctx, p, "missing", defaultTimeout, readInput(r, 0), nil)
		Expect(err).To(MatchError(errFunctionNotFound))
		Expect(r.closed()).To(BeTrue())
	})

	It("runs onOutput before the scope closes", func() {
		r := &trackingReader{Reader: strings.NewReader("x")}
		var h uint32
		var got []any
		_, err := callPluginFunctionScoped[struct{}, struct{}](ctx, p, "noop", defaultTimeout,
			func(s *callScope) (struct{}, error) {
				var err error
				h, err = s.register(newReadStream(r))
				return struct{}{}, err
			},
			func(s *callScope, _ struct{}) error {
				var err error
				got, err = s.transfer(h)
				return err
			})
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(ConsistOf(BeIdenticalTo(r)))
		Expect(r.closed()).To(BeFalse())
	})

	It("closes the scope when the time limit passes", func() {
		r := &trackingReader{Reader: strings.NewReader("x")}
		_, err := callPluginFunctionScoped[streamsInput, struct{}](ctx, p, "spin", 200*time.Millisecond, readInput(r, 0), nil)
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(r.closed()).To(BeTrue())
	})
})

var _ = Describe("Plugin unload", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("closes the plugin's stream table", func() {
		manager, _ := createTestManagerWithPlugins(nil, "test-streams"+PackageExtension)
		p := loadedTestPlugin(manager, "test-streams")
		Expect(manager.unloadPlugin("test-streams")).To(Succeed())
		_, err := p.streams.newScope(ctx)
		Expect(err).To(MatchError(errPluginUnloaded))
	})
})
