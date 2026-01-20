//go:build !windows

package plugins

import (
	"context"
	"sync"

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

	It("records metrics for failed plugin calls (error returned)", func() {
		// Create a manager with error config to force plugin errors
		errorRecorder := &mockMetricsRecorder{}
		errorManager, _ := createTestManagerWithPluginsAndMetrics(
			map[string]map[string]string{
				"test-metadata-agent": {"error": "simulated error"},
			},
			errorRecorder,
			"test-metadata-agent"+PackageExtension,
		)

		errorAgent, ok := errorManager.LoadMediaAgent("test-metadata-agent")
		Expect(ok).To(BeTrue())

		retriever := errorAgent.(agents.ArtistBiographyRetriever)
		_, err := retriever.GetArtistBiography(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
		Expect(err).To(HaveOccurred())

		calls := errorRecorder.getCalls()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].plugin).To(Equal("test-metadata-agent"))
		Expect(calls[0].method).To(Equal(FuncGetArtistBiography))
		Expect(calls[0].ok).To(BeFalse())
	})

	It("records metrics for not-implemented functions", func() {
		// Use partial metadata agent that doesn't implement GetArtistMBID
		partialRecorder := &mockMetricsRecorder{}
		partialManager, _ := createTestManagerWithPluginsAndMetrics(
			nil,
			partialRecorder,
			"partial-metadata-agent"+PackageExtension,
		)

		partialAgent, ok := partialManager.LoadMediaAgent("partial-metadata-agent")
		Expect(ok).To(BeTrue())

		retriever := partialAgent.(agents.ArtistMBIDRetriever)
		_, err := retriever.GetArtistMBID(GinkgoT().Context(), "artist-1", "Test Artist")
		Expect(err).To(MatchError(errNotImplemented))

		calls := partialRecorder.getCalls()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].plugin).To(Equal("partial-metadata-agent"))
		Expect(calls[0].method).To(Equal(FuncGetArtistMBID))
		Expect(calls[0].ok).To(BeFalse())
	})
})
