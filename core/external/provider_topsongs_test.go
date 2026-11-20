package external_test

import (
	"context"
	"errors"

	_ "github.com/navidrome/navidrome/adapters/lastfm"
	_ "github.com/navidrome/navidrome/adapters/listenbrainz"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - TopSongs", func() {
	var (
		p   Provider
		ds  model.DataStore
		ag  *mockAgents
		ctx context.Context
	)

	artist1 := model.Artist{ID: "artist-1", Name: "Artist One", OrderArtistName: "artist one", MbzArtistID: "mbid-artist-1"}

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), adminUser)
		conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
		ds = persistence.New(db.Db())
		ag = new(mockAgents)
		p = NewProvider(ds, ag, matcher.New(ds), &fakeBroker{})
	})

	withFailingArtists := func(err error) {
		failing := failingArtists(ds, err)
		p = NewProvider(failing, ag, matcher.New(failing), &fakeBroker{})
	}

	It("returns top songs for a known artist", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"},
			model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-song-2"},
		)
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: "mbid-song-2"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1", "song-2"}))
		ag.AssertExpectations(GinkgoT())
	})

	Context("artist id and/or name", func() {
		BeforeEach(func() {
			persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
			persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"})
			agentSongs := []agents.Song{{Name: "Song One", MBID: "mbid-song-1"}}
			ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()
		})

		It("returns top songs for a known artist by id only", func() {
			songs, err := p.TopSongs(ctx, "", "artist-1", 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(ids(songs)).To(Equal([]string{"song-1"}))
			ag.AssertExpectations(GinkgoT())
		})

		It("returns top songs for a known artist by id, falling back to name", func() {
			songs, err := p.TopSongs(ctx, "Artist One", "fake-id", 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(ids(songs)).To(Equal([]string{"song-1"}))
			ag.AssertExpectations(GinkgoT())
		})
	})

	It("bails if lookup by id returns a fatal error", func() {
		withFailingArtists(model.ErrInvalidAuth)

		songs, err := p.TopSongs(ctx, "Artist One", "artist-1", 2)

		Expect(songs).To(BeEmpty())
		Expect(err).To(MatchError(model.ErrInvalidAuth))
	})

	It("bails if lookup by name returns a fatal error", func() {
		withFailingArtists(model.ErrInvalidAuth)

		songs, err := p.TopSongs(ctx, "Artist One", "", 2)

		Expect(songs).To(BeEmpty())
		Expect(err).To(MatchError(model.ErrInvalidAuth))
	})

	It("backfills name and MBID onto an unnamed primary credit (the queried artist) and matches", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", Participants: credited(artist1)})

		// The blank first credit IS the queried artist, so enrichment fills both name and MBID and
		// the song matches the queried artist's track via the backfilled identity.
		agentSongs := []agents.Song{
			{Name: "Song One", Artists: []agents.Artist{{}}},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1"}))
	})

	It("does not stamp the queried MBID onto an already-named different first credit", func() {
		// The first credit is a different artist (Two) without an MBID. Stamping One's MBID onto it
		// would resolve it to One, matching One's same-titled track.
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "one-track", Title: "Collab Song", ArtistID: "artist-1", Participants: credited(artist1)})

		agentSongs := []agents.Song{
			{Name: "Collab Song", Artists: []agents.Artist{{Name: "Artist Two"}}},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(BeEmpty())
	})

	It("returns nil for an unknown artist", func() {
		songs, err := p.TopSongs(ctx, "Unknown Artist", "", 5)

		Expect(err).ToNot(HaveOccurred()) // TopSongs returns nil error if artist not found
		Expect(songs).To(BeNil())
		ag.AssertNotCalled(GinkgoT(), "GetArtistTopSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns error when the agent returns an error", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		agentErr := errors.New("agent error")
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 5).Return(nil, agentErr).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 5)

		Expect(err).To(MatchError(agentErr))
		Expect(songs).To(BeNil())
		ag.AssertExpectations(GinkgoT())
	})

	It("returns ErrNotFound when the agent returns ErrNotFound", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 5).Return(nil, agents.ErrNotFound).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 5)

		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(songs).To(BeNil())
		ag.AssertExpectations(GinkgoT())
	})

	// This endpoint answered with an empty list before retry-later existed; it must keep doing so.
	It("returns an empty list, not a client error, when the agents are throttled", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 5).
			Return(nil, agents.ErrRetryLater).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 5)

		Expect(songs).To(BeEmpty())
		Expect(err).To(MatchError(model.ErrNotFound), "the handler renders this as an empty 200")
		Expect(err).ToNot(MatchError(agents.ErrRetryLater))
		ag.AssertExpectations(GinkgoT())
	})

	It("returns fewer songs if count is less than available top songs", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"})
		agentSongs := []agents.Song{{Name: "Song One", MBID: "mbid-song-1"}}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1"}))
		ag.AssertExpectations(GinkgoT())
	})

	It("returns fewer songs if fewer matching tracks are found", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"})
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: "mbid-song-2"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1"}))
		ag.AssertExpectations(GinkgoT())
	})

	It("returns error when context is canceled during agent call", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		canceledCtx, cancel := context.WithCancel(ctx)
		ag.On("GetArtistTopSongs", canceledCtx, "artist-1", "Artist One", "mbid-artist-1", 5).
			Run(func(mock.Arguments) { cancel() }).
			Return(nil, context.Canceled).Once()

		songs, err := p.TopSongs(canceledCtx, "Artist One", "", 5)

		Expect(err).To(MatchError(context.Canceled))
		Expect(songs).To(BeNil())
		ag.AssertExpectations(GinkgoT())
	})

	It("falls back to title matching when MbzRecordingID is missing", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", Artist: "Artist One", ArtistID: "artist-1", Participants: credited(artist1)},
			model.MediaFile{ID: "song-2", Title: "Song Two", Artist: "Artist One", ArtistID: "artist-1", Participants: credited(artist1)},
		)
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: ""},
			{Name: "Song Two", MBID: ""},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1", "song-2"}))
		ag.AssertExpectations(GinkgoT())
	})

	It("combines MBID and title matching when some songs have missing MbzRecordingID", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"},
			model.MediaFile{ID: "song-2", Title: "Song Two", Artist: "Artist One", ArtistID: "artist-1", Participants: credited(artist1)},
		)
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: ""},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1", "song-2"}))
		ag.AssertExpectations(GinkgoT())
	})

	It("only returns requested count when provider returns additional items", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"},
			model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-song-2"},
		)
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: "mbid-song-2"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1"}))
		ag.AssertExpectations(GinkgoT())
	})

	It("matches songs by ID first when agent provides IDs", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1"},
			model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1"},
		)
		agentSongs := []agents.Song{
			{ID: "song-1", Name: "Song One"},
			{ID: "song-2", Name: "Song Two"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1", "song-2"}))
		ag.AssertExpectations(GinkgoT())
	})

	It("falls back to MBID when ID is not found", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist1)
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"})
		agentSongs := []agents.Song{
			{ID: "non-existent-id", Name: "Song One", MBID: "mbid-song-1"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", "", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1"}))
		ag.AssertExpectations(GinkgoT())
	})
})
