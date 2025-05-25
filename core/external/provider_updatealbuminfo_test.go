package external_test

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
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

var _ = Describe("Provider - UpdateAlbumInfo", func() {
	var (
		ctx           context.Context
		p             external.Provider
		ds            *tests.MockDataStore
		ag            *mockAgents
		mockAlbumRepo *tests.MockAlbumRepo
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = new(tests.MockDataStore)
		ag = new(mockAgents)
		p = external.NewProvider(ds, ag)
		mockAlbumRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
		conf.Server.DevAlbumInfoTimeToLive = 1 * time.Hour
	})

	It("returns error when album is not found", func() {
		album, err := p.UpdateAlbumInfo(ctx, "al-not-found")

		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(album).To(BeNil())
		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("populates info when album exists but has no external info", func() {
		originalAlbum := &model.Album{
			ID:          "al-existing",
			Name:        "Test Album",
			AlbumArtist: "Test Artist",
			MbzAlbumID:  "mbid-album",
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		expectedInfo := &agents.AlbumInfo{
			URL:         "http://example.com/album",
			Description: "Album Description",
		}
		ag.On("GetAlbumInfo", ctx, "Test Album", "Test Artist", "mbid-album").Return(expectedInfo, nil)
		ag.On("GetAlbumImages", ctx, "Test Album", "Test Artist", "mbid-album").Return([]agents.ExternalImage{
			{URL: "http://example.com/large.jpg", Size: 300},
			{URL: "http://example.com/medium.jpg", Size: 200},
			{URL: "http://example.com/small.jpg", Size: 100},
		}, nil)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-existing")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(updatedAlbum.ID).To(Equal("al-existing"))
		Expect(updatedAlbum.ExternalUrl).To(Equal("http://example.com/album"))
		Expect(updatedAlbum.Description).To(Equal("Album Description"))
		Expect(updatedAlbum.ExternalInfoUpdatedAt).NotTo(BeNil())
		Expect(*updatedAlbum.ExternalInfoUpdatedAt).To(BeTemporally("~", time.Now(), time.Second))

		ag.AssertExpectations(GinkgoT())
	})

	It("returns cached info when album exists and info is not expired", func() {
		now := time.Now()
		originalAlbum := &model.Album{
			ID:                    "al-cached",
			Name:                  "Cached Album",
			AlbumArtist:           "Cached Artist",
			ExternalUrl:           "http://cached.com/album",
			Description:           "Cached Desc",
			LargeImageUrl:         "http://cached.com/large.jpg",
			ExternalInfoUpdatedAt: gg.P(now.Add(-conf.Server.DevAlbumInfoTimeToLive / 2)),
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-cached")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum))

		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns cached info and triggers background refresh when info is expired", func() {
		now := time.Now()
		expiredTime := now.Add(-conf.Server.DevAlbumInfoTimeToLive * 2)
		originalAlbum := &model.Album{
			ID:                    "al-expired",
			Name:                  "Expired Album",
			AlbumArtist:           "Expired Artist",
			ExternalUrl:           "http://expired.com/album",
			Description:           "Expired Desc",
			LargeImageUrl:         "http://expired.com/large.jpg",
			ExternalInfoUpdatedAt: gg.P(expiredTime),
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-expired")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum))

		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns error when agent fails to get album info", func() {
		originalAlbum := &model.Album{
			ID:          "al-agent-error",
			Name:        "Agent Error Album",
			AlbumArtist: "Agent Error Artist",
			MbzAlbumID:  "mbid-agent-error",
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		expectedErr := errors.New("agent communication failed")
		ag.On("GetAlbumInfo", ctx, "Agent Error Album", "Agent Error Artist", "mbid-agent-error").Return(nil, expectedErr)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-agent-error")

		Expect(err).To(MatchError(expectedErr))
		Expect(updatedAlbum).To(BeNil())
		ag.AssertExpectations(GinkgoT())
	})

	It("returns original album when agent returns ErrNotFound", func() {
		originalAlbum := &model.Album{
			ID:          "al-agent-notfound",
			Name:        "Agent NotFound Album",
			AlbumArtist: "Agent NotFound Artist",
			MbzAlbumID:  "mbid-agent-notfound",
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		ag.On("GetAlbumInfo", ctx, "Agent NotFound Album", "Agent NotFound Artist", "mbid-agent-notfound").Return(nil, agents.ErrNotFound)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-agent-notfound")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum))
		Expect(updatedAlbum.ExternalInfoUpdatedAt).To(BeNil())

		ag.AssertExpectations(GinkgoT())
	})
})
