package external_test

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type fakeBroker struct {
	events.Broker
	mu     sync.Mutex
	events []events.Event
}

func (f *fakeBroker) SendBroadcastMessage(_ context.Context, e events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func (f *fakeBroker) sent() []events.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.events)
}

var _ = Describe("Provider - RefreshInfo", func() {
	var (
		ctx            context.Context
		p              external.Provider
		ds             *tests.MockDataStore
		ag             *mockAgents
		broker         *fakeBroker
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
		broker = &fakeBroker{}
		p = external.NewProvider(ds, ag, matcher.New(ds), broker)
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

	It("broadcasts a RefreshResource naming the artist", func() {
		mockArtistRepo.SetData(model.Artists{{ID: "ar-1", Name: "Test Artist"}})
		expectArtistAgents()

		Expect(p.RefreshInfo(ctx, model.KindArtistArtwork, "ar-1")).To(Succeed())

		sent := broker.sent()
		Expect(sent).To(HaveLen(1))
		rr, ok := sent[0].(*events.RefreshResource)
		Expect(ok).To(BeTrue())
		Expect(rr.Data(rr)).To(ContainSubstring("ar-1"))
		Expect(rr.Data(rr)).To(ContainSubstring("artist"))
	})

	It("broadcasts a RefreshResource naming the album", func() {
		mockAlbumRepo.SetData(model.Albums{{ID: "al-1", Name: "Test Album", AlbumArtist: "Test Artist"}})
		expectAlbumAgents()

		Expect(p.RefreshInfo(ctx, model.KindAlbumArtwork, "al-1")).To(Succeed())

		sent := broker.sent()
		Expect(sent).To(HaveLen(1))
		Expect(sent[0].Data(sent[0])).To(ContainSubstring("album"))
		Expect(sent[0].Data(sent[0])).To(ContainSubstring("al-1"))
	})

	It("does not broadcast when the artist cannot be loaded", func() {
		mockArtistRepo.SetData(model.Artists{{ID: "ar-1", Name: "Test Artist"}})
		expectArtistAgents()
		mockArtistRepo.SetError(true)

		_ = p.RefreshInfo(ctx, model.KindArtistArtwork, "ar-1")

		Expect(broker.sent()).To(BeEmpty())
	})

	It("reports which kinds have external info", func() {
		Expect(external.HasInfo(model.KindArtistArtwork)).To(BeTrue())
		Expect(external.HasInfo(model.KindAlbumArtwork)).To(BeTrue())
		Expect(external.HasInfo(model.KindPlaylistArtwork)).To(BeFalse())
	})
})
