package external_test

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func init() {
	log.SetLevel(log.LevelDebug)
}

var _ = Describe("Provider - UpdateArtistInfo", func() {
	var (
		ctx            context.Context
		p              external.Provider
		ds             *tests.MockDataStore
		ag             *mockAgents
		mockArtistRepo *tests.MockArtistRepo
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DevArtistInfoTimeToLive = 1 * time.Hour
		ctx = GinkgoT().Context()
		ds = new(tests.MockDataStore)
		ag = new(mockAgents)
		p = external.NewProvider(ds, ag)
		mockArtistRepo = ds.Artist(ctx).(*tests.MockArtistRepo)
	})

	It("returns error when artist is not found", func() {
		artist, err := p.UpdateArtistInfo(ctx, "ar-not-found", 10, false)

		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(artist).To(BeNil())
		ag.AssertNotCalled(GinkgoT(), "GetArtistMBID")
		ag.AssertNotCalled(GinkgoT(), "GetArtistImages")
		ag.AssertNotCalled(GinkgoT(), "GetArtistBiography")
		ag.AssertNotCalled(GinkgoT(), "GetArtistURL")
		ag.AssertNotCalled(GinkgoT(), "GetSimilarArtists")
	})

	It("populates info when artist exists but has no external info", func() {
		originalArtist := &model.Artist{
			ID:   "ar-existing",
			Name: "Test Artist",
		}
		mockArtistRepo.SetData(model.Artists{*originalArtist})

		expectedMBID := "mbid-artist-123"
		expectedBio := "Artist Bio"
		expectedURL := "http://artist.url"
		expectedImages := []agents.ExternalImage{
			{URL: "http://large.jpg", Size: 300},
			{URL: "http://medium.jpg", Size: 200},
			{URL: "http://small.jpg", Size: 100},
		}
		rawSimilar := []agents.Artist{
			{Name: "Similar Artist 1", MBID: "mbid-similar-1"},
			{Name: "Similar Artist 2", MBID: "mbid-similar-2"},
			{Name: "Similar Artist 3", MBID: "mbid-similar-3"},
		}
		similarInDS := model.Artist{ID: "ar-similar-2", Name: "Similar Artist 2"}

		ag.On("GetArtistMBID", ctx, "ar-existing", "Test Artist").Return(expectedMBID, nil).Once()
		ag.On("GetArtistImages", ctx, "ar-existing", "Test Artist", expectedMBID).Return(expectedImages, nil).Once()
		ag.On("GetArtistBiography", ctx, "ar-existing", "Test Artist", expectedMBID).Return(expectedBio, nil).Once()
		ag.On("GetArtistURL", ctx, "ar-existing", "Test Artist", expectedMBID).Return(expectedURL, nil).Once()
		ag.On("GetSimilarArtists", ctx, "ar-existing", "Test Artist", expectedMBID, 100).Return(rawSimilar, nil).Once()

		mockArtistRepo.SetData(model.Artists{*originalArtist, similarInDS})

		updatedArtist, err := p.UpdateArtistInfo(ctx, "ar-existing", 10, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedArtist).NotTo(BeNil())
		Expect(updatedArtist.ID).To(Equal("ar-existing"))
		Expect(updatedArtist.MbzArtistID).To(Equal(expectedMBID))
		Expect(updatedArtist.Biography).To(Equal("Artist Bio"))
		Expect(updatedArtist.ExternalUrl).To(Equal(expectedURL))
		Expect(updatedArtist.LargeImageUrl).To(Equal("http://large.jpg"))
		Expect(updatedArtist.MediumImageUrl).To(Equal("http://medium.jpg"))
		Expect(updatedArtist.SmallImageUrl).To(Equal("http://small.jpg"))
		Expect(updatedArtist.ExternalInfoUpdatedAt).NotTo(BeNil())
		Expect(*updatedArtist.ExternalInfoUpdatedAt).To(BeTemporally("~", time.Now(), time.Second))

		Expect(updatedArtist.SimilarArtists).To(HaveLen(1))
		Expect(updatedArtist.SimilarArtists[0].ID).To(Equal("ar-similar-2"))
		Expect(updatedArtist.SimilarArtists[0].Name).To(Equal("Similar Artist 2"))

		ag.AssertExpectations(GinkgoT())
	})

	It("returns cached info when artist exists and info is not expired", func() {
		now := time.Now()
		originalArtist := &model.Artist{
			ID:                    "ar-cached",
			Name:                  "Cached Artist",
			MbzArtistID:           "mbid-cached",
			ExternalUrl:           "http://cached.url",
			Biography:             "Cached Bio",
			LargeImageUrl:         "http://cached_large.jpg",
			ExternalInfoUpdatedAt: gg.P(now.Add(-conf.Server.DevArtistInfoTimeToLive / 2)),
			SimilarArtists: model.Artists{
				{ID: "ar-similar-present", Name: "Similar Present"},
				{ID: "ar-similar-absent", Name: "Similar Absent"},
			},
		}
		similarInDS := model.Artist{ID: "ar-similar-present", Name: "Similar Present Updated"}
		mockArtistRepo.SetData(model.Artists{*originalArtist, similarInDS})

		updatedArtist, err := p.UpdateArtistInfo(ctx, "ar-cached", 5, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedArtist).NotTo(BeNil())
		Expect(updatedArtist.ID).To(Equal(originalArtist.ID))
		Expect(updatedArtist.Name).To(Equal(originalArtist.Name))
		Expect(updatedArtist.MbzArtistID).To(Equal(originalArtist.MbzArtistID))
		Expect(updatedArtist.ExternalUrl).To(Equal(originalArtist.ExternalUrl))
		Expect(updatedArtist.Biography).To(Equal(originalArtist.Biography))
		Expect(updatedArtist.LargeImageUrl).To(Equal(originalArtist.LargeImageUrl))
		Expect(updatedArtist.ExternalInfoUpdatedAt).To(Equal(originalArtist.ExternalInfoUpdatedAt))

		Expect(updatedArtist.SimilarArtists).To(HaveLen(1))
		Expect(updatedArtist.SimilarArtists[0].ID).To(Equal(similarInDS.ID))
		Expect(updatedArtist.SimilarArtists[0].Name).To(Equal(similarInDS.Name))

		ag.AssertNotCalled(GinkgoT(), "GetArtistMBID")
		ag.AssertNotCalled(GinkgoT(), "GetArtistImages")
		ag.AssertNotCalled(GinkgoT(), "GetArtistBiography")
		ag.AssertNotCalled(GinkgoT(), "GetArtistURL")
	})

	It("returns cached info and triggers background refresh when info is expired", func() {
		now := time.Now()
		expiredTime := now.Add(-conf.Server.DevArtistInfoTimeToLive * 2)
		originalArtist := &model.Artist{
			ID:                    "ar-expired",
			Name:                  "Expired Artist",
			ExternalInfoUpdatedAt: gg.P(expiredTime),
			SimilarArtists: model.Artists{
				{ID: "ar-exp-similar", Name: "Expired Similar"},
			},
		}
		similarInDS := model.Artist{ID: "ar-exp-similar", Name: "Expired Similar Updated"}
		mockArtistRepo.SetData(model.Artists{*originalArtist, similarInDS})

		updatedArtist, err := p.UpdateArtistInfo(ctx, "ar-expired", 5, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedArtist).NotTo(BeNil())
		Expect(updatedArtist.ID).To(Equal(originalArtist.ID))
		Expect(updatedArtist.Name).To(Equal(originalArtist.Name))
		Expect(updatedArtist.ExternalInfoUpdatedAt).To(Equal(originalArtist.ExternalInfoUpdatedAt))

		Expect(updatedArtist.SimilarArtists).To(HaveLen(1))
		Expect(updatedArtist.SimilarArtists[0].ID).To(Equal(similarInDS.ID))
		Expect(updatedArtist.SimilarArtists[0].Name).To(Equal(similarInDS.Name))

		ag.AssertNotCalled(GinkgoT(), "GetArtistMBID")
		ag.AssertNotCalled(GinkgoT(), "GetArtistImages")
		ag.AssertNotCalled(GinkgoT(), "GetArtistBiography")
		ag.AssertNotCalled(GinkgoT(), "GetArtistURL")
	})

	It("includes non-present similar artists when includeNotPresent is true", func() {
		now := time.Now()
		originalArtist := &model.Artist{
			ID:                    "ar-similar-test",
			Name:                  "Similar Test Artist",
			ExternalInfoUpdatedAt: gg.P(now.Add(-conf.Server.DevArtistInfoTimeToLive / 2)),
			SimilarArtists: model.Artists{
				{ID: "ar-sim-present", Name: "Similar Present"},
				{ID: "", Name: "Similar Absent Raw"},
				{ID: "ar-sim-absent-lookup", Name: "Similar Absent Lookup"},
			},
		}
		similarInDS := model.Artist{ID: "ar-sim-present", Name: "Similar Present Updated"}
		mockArtistRepo.SetData(model.Artists{*originalArtist, similarInDS})

		updatedArtist, err := p.UpdateArtistInfo(ctx, "ar-similar-test", 5, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedArtist).NotTo(BeNil())

		Expect(updatedArtist.SimilarArtists).To(HaveLen(3))
		Expect(updatedArtist.SimilarArtists[0].ID).To(Equal(similarInDS.ID))
		Expect(updatedArtist.SimilarArtists[0].Name).To(Equal(similarInDS.Name))
		Expect(updatedArtist.SimilarArtists[1].ID).To(BeEmpty())
		Expect(updatedArtist.SimilarArtists[1].Name).To(Equal("Similar Absent Raw"))
		Expect(updatedArtist.SimilarArtists[2].ID).To(BeEmpty())
		Expect(updatedArtist.SimilarArtists[2].Name).To(Equal("Similar Absent Lookup"))
	})

	It("updates ArtistInfo even if an optional agent call fails", func() {
		originalArtist := &model.Artist{
			ID:   "ar-agent-fail",
			Name: "Agent Fail Artist",
		}
		mockArtistRepo.SetData(model.Artists{*originalArtist})

		expectedErr := errors.New("agent MBID failed")
		ag.On("GetArtistMBID", ctx, "ar-agent-fail", "Agent Fail Artist").Return("", expectedErr).Once()
		ag.On("GetArtistImages", ctx, "ar-agent-fail", "Agent Fail Artist", mock.Anything).Return(nil, nil).Maybe()
		ag.On("GetArtistBiography", ctx, "ar-agent-fail", "Agent Fail Artist", mock.Anything).Return("", nil).Maybe()
		ag.On("GetArtistURL", ctx, "ar-agent-fail", "Agent Fail Artist", mock.Anything).Return("", nil).Maybe()
		ag.On("GetSimilarArtists", ctx, "ar-agent-fail", "Agent Fail Artist", mock.Anything, 100).Return(nil, nil).Maybe()

		updatedArtist, err := p.UpdateArtistInfo(ctx, "ar-agent-fail", 10, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedArtist).NotTo(BeNil())
		Expect(updatedArtist.ID).To(Equal("ar-agent-fail"))
		ag.AssertExpectations(GinkgoT())
	})
})
