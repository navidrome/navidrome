package external_test

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - GetSimilarArtistImageURL", func() {
	var (
		ctx context.Context
		p   external.Provider
		ds  *tests.MockDataStore
		ag  *mockAgents
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = GinkgoT().Context()
		ds = new(tests.MockDataStore)
		ag = new(mockAgents)
		p = external.NewProvider(ds, ag, matcher.New(ds), &fakeBroker{})
	})

	When("the feature is disabled (the default)", func() {
		It("returns empty without calling any agent", func() {
			conf.Server.EnableSimilarArtistImages = false

			url := p.GetSimilarArtistImageURL(ctx, "Some Artist", "mbid-1")

			Expect(url).To(BeEmpty())
			ag.AssertNotCalled(GinkgoT(), "GetArtistImages")
		})
	})

	When("the feature is enabled", func() {
		BeforeEach(func() {
			conf.Server.EnableSimilarArtistImages = true
		})

		It("returns empty for an unresolved artist without blocking on an agent call", func() {
			ag.On("GetArtistImages", mock.Anything, "", "Some Artist", "mbid-1").
				Return([]agents.ExternalImage{{URL: "http://example.com/big.jpg", Size: 600}}, nil)

			url := p.GetSimilarArtistImageURL(ctx, "Some Artist", "mbid-1")

			// Resolution always happens on a background worker, never synchronously on the
			// request path - a miss must return immediately, with nothing found yet.
			Expect(url).To(BeEmpty())
		})

		It("returns empty for an artist with no name, without calling any agent", func() {
			url := p.GetSimilarArtistImageURL(ctx, "", "mbid-1")

			Expect(url).To(BeEmpty())
			ag.AssertNotCalled(GinkgoT(), "GetArtistImages")
		})
	})
})
