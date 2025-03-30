package extdata_test

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/extdata"
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

var _ = Describe("Provider UpdateAlbumInfo", func() {
	var (
		ctx context.Context
		p   extdata.Provider
		ds  *tests.MockDataStore
		ag  *extdata.MockAgents
	)

	BeforeEach(func() {
		ctx = context.Background()
		ds = new(tests.MockDataStore)
		ag = new(extdata.MockAgents)
		p = extdata.NewProvider(ds, ag)

		// Default config
		conf.Server.DevAlbumInfoTimeToLive = 1 * time.Hour
	})

	It("returns error when album is not found", func() {
		// MockDataStore will return a MockAlbumRepo with an empty Data map by default
		_ = ds.Album(ctx).(*tests.MockAlbumRepo) // Ensure MockAlbumRepo is initialized

		album, err := p.UpdateAlbumInfo(ctx, "al-not-found")

		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(album).To(BeNil())
		// No AssertExpectations needed for custom mock
		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("populates info when album exists but has no external info", func() {
		// Setup: Album exists in DS, ExternalInfoUpdatedAt is nil
		originalAlbum := &model.Album{
			ID:          "al-existing",
			Name:        "Test Album",
			AlbumArtist: "Test Artist",
			MbzAlbumID:  "mbid-album",
		}
		mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		// Mock agent response
		expectedInfo := &agents.AlbumInfo{
			URL:         "http://example.com/album",
			Description: "Album Description",
			Images: []agents.ExternalImage{
				{URL: "http://example.com/large.jpg", Size: 300},
				{URL: "http://example.com/medium.jpg", Size: 200},
				{URL: "http://example.com/small.jpg", Size: 100},
			},
		}
		ag.On("GetAlbumInfo", ctx, "Test Album", "Test Artist", "mbid-album").Return(expectedInfo, nil)

		// Call the method under test
		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-existing")

		// Assertions
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(updatedAlbum.ID).To(Equal("al-existing"))
		Expect(updatedAlbum.ExternalUrl).To(Equal("http://example.com/album"))
		Expect(updatedAlbum.Description).To(Equal("Album Description"))
		Expect(updatedAlbum.LargeImageUrl).To(Equal("http://example.com/large.jpg"))
		Expect(updatedAlbum.MediumImageUrl).To(Equal("http://example.com/medium.jpg"))
		Expect(updatedAlbum.SmallImageUrl).To(Equal("http://example.com/small.jpg"))
		Expect(updatedAlbum.ExternalInfoUpdatedAt).NotTo(BeNil())
		Expect(*updatedAlbum.ExternalInfoUpdatedAt).To(BeTemporally("~", time.Now(), time.Second)) // Check timestamp was set

		ag.AssertExpectations(GinkgoT())
	})

	It("returns cached info when album exists and info is not expired", func() {
		// Setup: Album exists in DS, ExternalInfoUpdatedAt is recent
		now := time.Now()
		originalAlbum := &model.Album{
			ID:                    "al-cached",
			Name:                  "Cached Album",
			AlbumArtist:           "Cached Artist",
			ExternalUrl:           "http://cached.com/album",
			Description:           "Cached Desc",
			LargeImageUrl:         "http://cached.com/large.jpg",
			ExternalInfoUpdatedAt: gg.P(now.Add(-conf.Server.DevAlbumInfoTimeToLive / 2)), // Not expired
		}
		mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		// Call the method under test
		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-cached")

		// Assertions
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum)) // Should return the exact cached data

		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns cached info and triggers background refresh when info is expired", func() {
		// Setup: Album exists in DS, ExternalInfoUpdatedAt is old
		now := time.Now()
		expiredTime := now.Add(-conf.Server.DevAlbumInfoTimeToLive * 2) // Definitely expired
		originalAlbum := &model.Album{
			ID:                    "al-expired",
			Name:                  "Expired Album",
			AlbumArtist:           "Expired Artist",
			ExternalUrl:           "http://expired.com/album",
			Description:           "Expired Desc",
			LargeImageUrl:         "http://expired.com/large.jpg",
			ExternalInfoUpdatedAt: gg.P(expiredTime),
		}
		mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		// Call the method under test
		// Note: We are not testing the background refresh directly here, only the immediate return
		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-expired")

		// Assertions: Should return the stale data immediately
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum)) // Should return the exact cached (stale) data

		// Assertions: Agent should NOT be called synchronously
		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns error when agent fails to get album info", func() {
		// Setup: Album exists in DS, ExternalInfoUpdatedAt is nil
		originalAlbum := &model.Album{
			ID:          "al-agent-error",
			Name:        "Agent Error Album",
			AlbumArtist: "Agent Error Artist",
			MbzAlbumID:  "mbid-agent-error",
		}
		mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		// Mock agent response with an error
		expectedErr := errors.New("agent communication failed")
		ag.On("GetAlbumInfo", ctx, "Agent Error Album", "Agent Error Artist", "mbid-agent-error").Return(nil, expectedErr)

		// Call the method under test
		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-agent-error")

		// Assertions
		Expect(err).To(MatchError(expectedErr))
		Expect(updatedAlbum).To(BeNil())
		ag.AssertExpectations(GinkgoT())
	})

	It("returns original album when agent returns ErrNotFound", func() {
		// Setup: Album exists in DS, ExternalInfoUpdatedAt is nil
		originalAlbum := &model.Album{
			ID:          "al-agent-notfound",
			Name:        "Agent NotFound Album",
			AlbumArtist: "Agent NotFound Artist",
			MbzAlbumID:  "mbid-agent-notfound",
			// ExternalInfoUpdatedAt is nil
		}
		mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		// Mock agent response with ErrNotFound
		ag.On("GetAlbumInfo", ctx, "Agent NotFound Album", "Agent NotFound Artist", "mbid-agent-notfound").Return(nil, agents.ErrNotFound)

		// Call the method under test
		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-agent-notfound")

		// Assertions
		Expect(err).NotTo(HaveOccurred()) // No error should be returned
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum))        // Should return original data
		Expect(updatedAlbum.ExternalInfoUpdatedAt).To(BeNil()) // Timestamp should not be set

		// Agent was called, but UpdateExternalInfo should not have been
		ag.AssertExpectations(GinkgoT())
		// We can't assert mockAlbumRepo.AssertNotCalled for UpdateExternalInfo because it's not a testify mock method
	})

	// Test cases will go here

})
