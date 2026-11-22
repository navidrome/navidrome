package external_test

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func init() {
	log.SetLevel(log.LevelDebug)
}

var _ = Describe("Provider - UpdateAlbumInfo", func() {
	var (
		ctx context.Context
		p   external.Provider
		ds  model.DataStore
		ag  *mockAgents
	)

	expectCached := func(got, want *model.Album) {
		GinkgoHelper()
		Expect(got.ID).To(Equal(want.ID))
		Expect(got.Name).To(Equal(want.Name))
		Expect(got.ExternalUrl).To(Equal(want.ExternalUrl))
		Expect(got.Description).To(Equal(want.Description))
		Expect(got.LargeImageUrl).To(Equal(want.LargeImageUrl))
		if want.ExternalInfoUpdatedAt == nil {
			Expect(got.ExternalInfoUpdatedAt).To(BeNil())
		} else {
			Expect(*got.ExternalInfoUpdatedAt).To(BeTemporally("==", *want.ExternalInfoUpdatedAt))
		}
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		ag = new(mockAgents)
		p = external.NewProvider(ctx, ds, ag, matcher.New(ds), &fakeBroker{})
		conf.Update(func(c *conf.Options) { c.DevAlbumInfoTimeToLive = 1 * time.Hour })
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
		persistencetest.PutAlbums(ctx, ds, *originalAlbum)

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

		stored, err := ds.Album().Get(ctx, "al-existing")
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Description).To(Equal("Album Description"))
		Expect(stored.LargeImageUrl).To(Equal("http://example.com/large.jpg"))
		Expect(stored.SmallImageUrl).To(Equal("http://example.com/small.jpg"))
		Expect(external.QueuedAlbumRefreshes(p)).To(BeZero(), "freshly fetched info must not be refreshed again")

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
			ExternalInfoUpdatedAt: new(now.Add(-conf.Server().DevAlbumInfoTimeToLive / 2)),
		}
		persistencetest.PutAlbums(ctx, ds, *originalAlbum)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-cached")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		expectCached(updatedAlbum, originalAlbum)

		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns cached info and triggers background refresh when info is expired", func() {
		now := time.Now()
		expiredTime := now.Add(-conf.Server().DevAlbumInfoTimeToLive * 2)
		originalAlbum := &model.Album{
			ID:                    "al-expired",
			Name:                  "Expired Album",
			AlbumArtist:           "Expired Artist",
			ExternalUrl:           "http://expired.com/album",
			Description:           "Expired Desc",
			LargeImageUrl:         "http://expired.com/large.jpg",
			ExternalInfoUpdatedAt: new(expiredTime),
		}
		persistencetest.PutAlbums(ctx, ds, *originalAlbum)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-expired")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		expectCached(updatedAlbum, originalAlbum)
		Expect(external.QueuedAlbumRefreshes(p)).To(Equal(1))

		ag.AssertNotCalled(GinkgoT(), "GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns error when agent fails to get album info", func() {
		originalAlbum := &model.Album{
			ID:          "al-agent-error",
			Name:        "Agent Error Album",
			AlbumArtist: "Agent Error Artist",
			MbzAlbumID:  "mbid-agent-error",
		}
		persistencetest.PutAlbums(ctx, ds, *originalAlbum)

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
		persistencetest.PutAlbums(ctx, ds, *originalAlbum)

		ag.On("GetAlbumInfo", ctx, "Agent NotFound Album", "Agent NotFound Artist", "mbid-agent-notfound").Return(nil, agents.ErrNotFound)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-agent-notfound")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		expectCached(updatedAlbum, originalAlbum)
		Expect(updatedAlbum.ExternalInfoUpdatedAt).To(BeNil())

		ag.AssertExpectations(GinkgoT())
	})

	It("returns the original album, unstamped, when the agents are throttled", func() {
		originalAlbum := &model.Album{
			ID:          "al-throttled",
			Name:        "Throttled Album",
			AlbumArtist: "Throttled Artist",
			MbzAlbumID:  "mbid-throttled",
		}
		persistencetest.PutAlbums(ctx, ds, *originalAlbum)

		ag.On("GetAlbumInfo", ctx, "Throttled Album", "Throttled Artist", "mbid-throttled").
			Return(nil, agents.ErrRetryLater)

		updatedAlbum, err := p.UpdateAlbumInfo(ctx, "al-throttled")

		Expect(err).NotTo(HaveOccurred())
		Expect(updatedAlbum).NotTo(BeNil())
		expectCached(updatedAlbum, originalAlbum)
		Expect(updatedAlbum.ExternalInfoUpdatedAt).To(BeNil())

		ag.AssertExpectations(GinkgoT())
	})
})
