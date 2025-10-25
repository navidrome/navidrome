package external_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/listenbrainz"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - TopSongs", func() {
	var (
		p             Provider
		artistRepo    *mockArtistRepo    // From provider_helper_test.go
		mediaFileRepo *mockMediaFileRepo // From provider_helper_test.go
		ag            *mockAgents        // Consolidated mock from export_test.go
		ctx           context.Context
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		artistRepo = newMockArtistRepo()       // Use helper mock
		mediaFileRepo = newMockMediaFileRepo() // Use helper mock

		// Configure tests.MockDataStore to use the testify/mock-based repos
		ds := &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
		}

		ag = new(mockAgents)

		p = NewProvider(ds, ag)
	})

	It("returns top songs for a known artist", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent response
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: "mbid-song-2"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		// Mock finding matching tracks (both returned in a single query)
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-song-2"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(2))
		Expect(songs[0].ID).To(Equal("song-1"))
		Expect(songs[1].ID).To(Equal("song-2"))
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
		mediaFileRepo.AssertExpectations(GinkgoT())
	})

	It("returns nil for an unknown artist", func() {
		// Mock artist not found
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{}, nil).Once()

		songs, err := p.TopSongs(ctx, "Unknown Artist", 5)

		Expect(err).ToNot(HaveOccurred()) // TopSongs returns nil error if artist not found
		Expect(songs).To(BeNil())
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertNotCalled(GinkgoT(), "GetArtistTopSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	It("returns error when the agent returns an error", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent error
		agentErr := errors.New("agent error")
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 5).Return(nil, agentErr).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 5)

		Expect(err).To(MatchError(agentErr))
		Expect(songs).To(BeNil())
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
	})

	It("returns ErrNotFound when the agent returns ErrNotFound", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent ErrNotFound
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 5).Return(nil, agents.ErrNotFound).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 5)

		Expect(err).To(MatchError(model.ErrNotFound))
		Expect(songs).To(BeNil())
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
	})

	It("returns fewer songs if count is less than available top songs", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent response (only need 1 for the test)
		agentSongs := []agents.Song{{Name: "Song One", MBID: "mbid-song-1"}}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		// Mock finding matching track
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1}, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(Equal("song-1"))
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
		mediaFileRepo.AssertExpectations(GinkgoT())
	})

	It("returns fewer songs if fewer matching tracks are found", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent response
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: "mbid-song-2"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		// Mock finding matching tracks (only find song 1 on bulk query)
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1}, nil).Once() // bulk MBID query
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{}, nil).Once()      // title fallback for song2

		songs, err := p.TopSongs(ctx, "Artist One", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(Equal("song-1"))
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
		mediaFileRepo.AssertExpectations(GinkgoT())
	})

	It("returns error when context is canceled during agent call", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Setup context that will be canceled
		canceledCtx, cancel := context.WithCancel(ctx)

		// Mock agent call to return context canceled error
		ag.On("GetArtistTopSongs", canceledCtx, "artist-1", "Artist One", "mbid-artist-1", 5).Return(nil, context.Canceled).Once()

		cancel() // Cancel the context before calling
		songs, err := p.TopSongs(canceledCtx, "Artist One", 5)

		Expect(err).To(MatchError(context.Canceled))
		Expect(songs).To(BeNil())
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
	})

	It("falls back to title matching when MbzRecordingID is missing", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent response with songs that have NO MBID (empty string)
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: ""}, // No MBID, should fall back to title matching
			{Name: "Song Two", MBID: ""}, // No MBID, should fall back to title matching
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		// Since there are no MBIDs, loadTracksByMBID should not make any database call
		// loadTracksByTitle should make a database call for title matching
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "", OrderTitle: "song one"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "", OrderTitle: "song two"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(2))
		Expect(songs[0].ID).To(Equal("song-1"))
		Expect(songs[1].ID).To(Equal("song-2"))
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
		mediaFileRepo.AssertExpectations(GinkgoT())
	})

	It("combines MBID and title matching when some songs have missing MbzRecordingID", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent response with mixed MBID availability
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"}, // Has MBID, should match by MBID
			{Name: "Song Two", MBID: ""},            // No MBID, should fall back to title matching
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

		// Mock the MBID query (finds song1 by MBID)
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1", OrderTitle: "song one"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1}, nil).Once()

		// Mock the title fallback query (finds song2 by title)
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "", OrderTitle: "song two"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song2}, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 2)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(2))
		Expect(songs[0].ID).To(Equal("song-1")) // Found by MBID
		Expect(songs[1].ID).To(Equal("song-2")) // Found by title
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
		mediaFileRepo.AssertExpectations(GinkgoT())
	})

	It("only returns requested count when provider returns additional items", func() {
		// Mock finding the artist
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
		artistRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.Artists{artist1}, nil).Once()

		// Mock agent response
		agentSongs := []agents.Song{
			{Name: "Song One", MBID: "mbid-song-1"},
			{Name: "Song Two", MBID: "mbid-song-2"},
		}
		ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

		// Mock finding matching tracks (both returned in a single query)
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-song-2"}
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()

		songs, err := p.TopSongs(ctx, "Artist One", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(Equal("song-1"))
		artistRepo.AssertExpectations(GinkgoT())
		ag.AssertExpectations(GinkgoT())
		mediaFileRepo.AssertExpectations(GinkgoT())
	})
})
