package plugins

import (
	"context"
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

	It("stops a call when its time limit passes", func() {
		start := time.Now()
		_, err := callPluginFunctionWithTimeout[struct{}, struct{}](ctx, p, "spin", 200*time.Millisecond, struct{}{})
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(time.Since(start)).To(BeNumerically("<", 5*time.Second))
	})

	It("records a call that runs out of time as a failure", func() {
		_, err := callPluginFunctionWithTimeout[struct{}, struct{}](ctx, p, "spin", 200*time.Millisecond, struct{}{})
		Expect(err).To(MatchError(context.DeadlineExceeded))

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
