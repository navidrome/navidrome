package external_test

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - RefreshInfo", func() {
	var (
		ctx            context.Context
		p              external.Provider
		ds             *tests.MockDataStore
		ag             *mockAgents
		mockArtistRepo *tests.MockArtistRepo
		mockAlbumRepo  *tests.MockAlbumRepo
	)

	expectArtistAgents := func() {
		ag.On("GetArtistMBID", mock.Anything, mock.Anything, mock.Anything).Return("mbid-1", nil)
		ag.On("GetArtistImages", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]agents.ExternalImage{}, nil)
		ag.On("GetArtistBiography", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("Fresh Bio", nil)
		ag.On("GetArtistURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("http://artist.url", nil)
		ag.On("GetSimilarArtists", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]agents.Artist{}, nil)
	}

	expectAlbumAgents := func() {
		ag.On("GetAlbumInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&agents.AlbumInfo{URL: "http://album.url", Description: "Fresh Notes"}, nil)
		ag.On("GetAlbumImages", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]agents.ExternalImage{}, nil)
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = new(tests.MockDataStore)
		ag = new(mockAgents)
		p = external.NewProvider(ds, ag, matcher.New(ds))
		mockArtistRepo = ds.Artist(ctx).(*tests.MockArtistRepo)
		mockAlbumRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
	})

	It("repopulates an artist even when its info is fresh", func() {
		fresh := time.Now()
		mockArtistRepo.SetData(model.Artists{{
			ID: "ar-1", Name: "Test Artist", Biography: "stale", ExternalInfoUpdatedAt: &fresh,
		}})
		expectArtistAgents()

		Expect(p.RefreshInfo(ctx, model.KindArtistArtwork, "ar-1")).To(Succeed())

		saved, err := mockArtistRepo.Get("ar-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(saved.Biography).To(Equal("Fresh Bio"))
	})

	It("repopulates an album even when its info is fresh", func() {
		fresh := time.Now()
		mockAlbumRepo.SetData(model.Albums{{
			ID: "al-1", Name: "Test Album", AlbumArtist: "Test Artist",
			Description: "stale", ExternalInfoUpdatedAt: &fresh,
		}})
		expectAlbumAgents()

		Expect(p.RefreshInfo(ctx, model.KindAlbumArtwork, "al-1")).To(Succeed())

		saved, err := mockAlbumRepo.Get("al-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(saved.Description).To(Equal("Fresh Notes"))
	})

	It("returns ErrNotFound for an unknown id", func() {
		Expect(p.RefreshInfo(ctx, model.KindArtistArtwork, "nope")).To(MatchError(model.ErrNotFound))
	})

	It("returns ErrNotFound for a kind with no external info", func() {
		Expect(p.RefreshInfo(ctx, model.KindPlaylistArtwork, "pl-1")).To(MatchError(model.ErrNotFound))
	})
})
