package external_test

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
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
		p = external.NewProvider(ds, ag, matcher.New(ds))
		DeferCleanup(p.Close)
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
		ag.On("GetAlbumInfo", mock.Anything, "Test Album", "Test Artist", "mbid-album").Return(expectedInfo, nil)
		ag.On("GetAlbumImages", mock.Anything, "Test Album", "Test Artist", "mbid-album").Return([]agents.ExternalImage{
			{URL: "http://example.com/large.jpg", Size: 300},
			{URL: "http://example.com/medium.jpg", Size: 200},
			{URL: "http://example.com/small.jpg", Size: 100},
		}, nil)

		first, err := p.UpdateAlbumInfo(ctx, "al-existing")
		Expect(err).NotTo(HaveOccurred())
		Expect(first).NotTo(BeNil())
		Expect(first.Description).To(BeEmpty())

		updatedAlbum := waitAlbumInfo(ctx, p, "al-existing", func(a *model.Album) bool {
			return a.Description == "Album Description"
		})
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
			ExternalInfoUpdatedAt: new(now.Add(-conf.Server.DevAlbumInfoTimeToLive / 2)),
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
			ExternalInfoUpdatedAt: new(expiredTime),
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})
		ag.On("GetAlbumInfo", mock.Anything, "Expired Album", "Expired Artist", "").Return(&agents.AlbumInfo{
			URL:         "http://refreshed.com/album",
			Description: "Refreshed Desc",
		}, nil).Maybe()
		ag.On("GetAlbumImages", mock.Anything, "Expired Album", "Expired Artist", "").Return([]agents.ExternalImage{}, nil).Maybe()

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-expired")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(*updatedAlbum).To(Equal(*originalAlbum))
		Eventually(func() bool { return ag.albumInfoHit.Load() }).WithTimeout(2 * time.Second).Should(BeTrue())
	})

	It("returns the album immediately when the agent fails, then refreshes in the background", func() {
		originalAlbum := &model.Album{
			ID:          "al-agent-error",
			Name:        "Agent Error Album",
			AlbumArtist: "Agent Error Artist",
			MbzAlbumID:  "mbid-agent-error",
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		expectedErr := errors.New("agent communication failed")
		// Wait for the mock call itself (not just albumInfoHit): albumInfoHit is set
		// before testify records the call, so AssertExpectations can race the worker.
		done := make(chan struct{})
		ag.On("GetAlbumInfo", mock.Anything, "Agent Error Album", "Agent Error Artist", "mbid-agent-error").
			Return(nil, expectedErr).
			Run(func(mock.Arguments) { close(done) }).
			Once()

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-agent-error")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		Expect(updatedAlbum.ID).To(Equal("al-agent-error"))
		Expect(updatedAlbum.Description).To(BeEmpty())
		Eventually(done).WithTimeout(2 * time.Second).Should(BeClosed())
		ag.AssertExpectations(GinkgoT())
	})

	It("returns original album immediately and stamps a refresh timestamp when the agent returns ErrNotFound", func() {
		originalAlbum := &model.Album{
			ID:          "al-agent-notfound",
			Name:        "Agent NotFound Album",
			AlbumArtist: "Agent NotFound Artist",
			MbzAlbumID:  "mbid-agent-notfound",
		}
		mockAlbumRepo.SetData(model.Albums{*originalAlbum})

		ag.On("GetAlbumInfo", mock.Anything, "Agent NotFound Album", "Agent NotFound Artist", "mbid-agent-notfound").Return(nil, agents.ErrNotFound)
		ag.On("GetAlbumImages", mock.Anything, "Agent NotFound Album", "Agent NotFound Artist", "mbid-agent-notfound").Return(nil, agents.ErrNotFound)

		first, err := p.UpdateAlbumInfo(ctx, "al-agent-notfound")
		Expect(err).NotTo(HaveOccurred())
		Expect(first).NotTo(BeNil())
		Expect(first.ExternalInfoUpdatedAt).To(BeNil())

		updatedAlbum := waitAlbumInfo(ctx, p, "al-agent-notfound", func(a *model.Album) bool {
			return a.ExternalInfoUpdatedAt != nil && !a.ExternalInfoUpdatedAt.IsZero()
		})
		Expect(updatedAlbum.ID).To(Equal("al-agent-notfound"))

		ag.AssertExpectations(GinkgoT())
	})
})
