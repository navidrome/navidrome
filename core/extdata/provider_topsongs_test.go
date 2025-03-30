package extdata

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/listenbrainz"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
	"github.com/navidrome/navidrome/model"

	// Use helper mocks, not tests.MockDataStore for this file due to filtering needs
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - TopSongs", func() {
	var (
		// ds model.DataStore // Keep using helper mocks below
		p                    Provider
		artistRepo           *mockArtistRepo    // From provider_helper_test.go
		mediaFileRepo        *mockMediaFileRepo // From provider_helper_test.go
		ag                   *MockAgents        // Consolidated mock from export_test.go
		ctx                  context.Context
		originalAgentsConfig string
	)

	BeforeEach(func() {
		ctx = context.Background()
		originalAgentsConfig = conf.Server.Agents

		artistRepo = newMockArtistRepo()       // Keep using helper mock
		mediaFileRepo = newMockMediaFileRepo() // Keep using helper mock

		// Mock DataStore interface implementation using helper mocks
		ds := &mockDataStoreForTopSongs{
			mockedArtist:    artistRepo,
			mockedMediaFile: mediaFileRepo,
		}

		ag = new(MockAgents) // Use consolidated mock

		p = NewProvider(ds, ag)
	})

	AfterEach(func() {
		conf.Server.Agents = originalAgentsConfig
	})

	Describe("TopSongs", func() {
		BeforeEach(func() {
			// Setup data directly in testify mocks if needed, or setup expectations
		})

		It("returns top songs for a known artist", func() {
			// Mock finding the artist
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{artist1}, nil).Once()

			// Mock agent response
			agentSongs := []agents.Song{
				{Name: "Song One", MBID: "mbid-song-1"},
				{Name: "Song Two", MBID: "mbid-song-2"},
			}
			ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

			// Mock finding matching tracks
			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
			song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-song-2"}
			mediaFileRepo.On("GetAll", mock.MatchedBy(matchMediaFileByMBIDFilter("mbid-song-1"))).Return(model.MediaFiles{song1}, nil).Once()
			mediaFileRepo.On("GetAll", mock.MatchedBy(matchMediaFileByMBIDFilter("mbid-song-2"))).Return(model.MediaFiles{song2}, nil).Once()

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
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{}, nil).Once()

			songs, err := p.TopSongs(ctx, "Unknown Artist", 5)

			Expect(err).ToNot(HaveOccurred()) // TopSongs returns nil error if artist not found
			Expect(songs).To(BeNil())
			artistRepo.AssertExpectations(GinkgoT())
			ag.AssertNotCalled(GinkgoT(), "GetArtistTopSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})

		It("returns error when the agent returns an error", func() {
			// Mock finding the artist
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{artist1}, nil).Once()

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
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{artist1}, nil).Once()

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
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{artist1}, nil).Once()

			// Mock agent response (only need 1 for the test)
			agentSongs := []agents.Song{{Name: "Song One", MBID: "mbid-song-1"}}
			ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 1).Return(agentSongs, nil).Once()

			// Mock finding matching track
			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
			mediaFileRepo.On("GetAll", mock.MatchedBy(matchMediaFileByMBIDFilter("mbid-song-1"))).Return(model.MediaFiles{song1}, nil).Once()

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
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{artist1}, nil).Once()

			// Mock agent response
			agentSongs := []agents.Song{
				{Name: "Song One", MBID: "mbid-song-1"},
				{Name: "Song Two", MBID: "mbid-song-2"}, // Agent knows this song
			}
			ag.On("GetArtistTopSongs", ctx, "artist-1", "Artist One", "mbid-artist-1", 2).Return(agentSongs, nil).Once()

			// Mock finding matching tracks (only find song 1)
			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-song-1"}
			mediaFileRepo.On("GetAll", mock.MatchedBy(matchMediaFileByMBIDFilter("mbid-song-1"))).Return(model.MediaFiles{song1}, nil).Once()
			mediaFileRepo.On("GetAll", mock.MatchedBy(matchMediaFileByMBIDFilter("mbid-song-2"))).Return(model.MediaFiles{}, nil).Once() // MBID lookup fails
			// Mock fallback lookup by title (also fails)
			mediaFileRepo.On("GetAll", mock.MatchedBy(matchMediaFileByTitleFilter("artist-1", "Song Two"))).Return(model.MediaFiles{}, nil).Once()

			songs, err := p.TopSongs(ctx, "Artist One", 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].ID).To(Equal("song-1"))
			artistRepo.AssertExpectations(GinkgoT())
			ag.AssertExpectations(GinkgoT())
			mediaFileRepo.AssertExpectations(GinkgoT())
		})

		// Context canceled test needs separate handling if async ops were involved, but TopSongs is synchronous.
		// We can test cancellation behavior if the agent call respects context cancellation.
		It("returns error when context is canceled during agent call", func() {
			// Mock finding the artist
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One", MbzArtistID: "mbid-artist-1"}
			artistRepo.On("GetAll", mock.MatchedBy(matchArtistByNameFilter)).Return(model.Artists{artist1}, nil).Once()

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
	})
})

// Helper mock implementing model.DataStore for this test file
// Allows using helper repo mocks instead of tests.MockDataStore
type mockDataStoreForTopSongs struct {
	model.DataStore // Embed for unimplemented methods
	mockedArtist    model.ArtistRepository
	mockedMediaFile model.MediaFileRepository
}

func (m *mockDataStoreForTopSongs) Artist(ctx context.Context) model.ArtistRepository {
	return m.mockedArtist
}

func (m *mockDataStoreForTopSongs) MediaFile(ctx context.Context) model.MediaFileRepository {
	return m.mockedMediaFile
}

// Helper functions to match squirrel filters for testify/mock
func matchArtistByNameFilter(opt model.QueryOptions) bool {
	if opt.Max != 1 || opt.Filters == nil {
		return false
	}
	_, ok := opt.Filters.(squirrel.Like)
	// Could add more specific checks on the Like clause if needed
	return ok
}

func matchMediaFileByMBIDFilter(expectedMBID string) func(opt model.QueryOptions) bool {
	return func(opt model.QueryOptions) bool {
		if opt.Filters == nil {
			return false
		}
		andClause, ok := opt.Filters.(squirrel.And)
		if !ok || len(andClause) < 2 {
			return false
		}
		foundMBID := false
		foundMissing := false
		for _, condition := range andClause {
			if eqClause, ok := condition.(squirrel.Eq); ok {
				if mbid, exists := eqClause["mbz_recording_id"]; exists && mbid == expectedMBID {
					foundMBID = true
				}
				if missing, exists := eqClause["missing"]; exists && missing == false {
					foundMissing = true
				}
			}
		}
		return foundMBID && foundMissing
	}
}

func matchMediaFileByTitleFilter(expectedArtistID, expectedTitle string) func(opt model.QueryOptions) bool {
	return func(opt model.QueryOptions) bool {
		if opt.Filters == nil || opt.Max != 1 {
			return false
		}
		andClause, ok := opt.Filters.(squirrel.And)
		if !ok || len(andClause) < 3 {
			return false
		}
		foundArtist := false
		foundTitle := false
		foundMissing := false
		for _, condition := range andClause {
			if orClause, ok := condition.(squirrel.Or); ok {
				for _, orCond := range orClause {
					if eq, ok := orCond.(squirrel.Eq); ok {
						if id, exists := eq["artist_id"]; exists && id == expectedArtistID {
							foundArtist = true
						}
						if id, exists := eq["album_artist_id"]; exists && id == expectedArtistID {
							foundArtist = true
						}
					}
				}
			} else if likeClause, ok := condition.(squirrel.Like); ok {
				if _, exists := likeClause["order_title"]; exists {
					// We assume the title matches here for simplicity; could compare sanitized title if needed
					foundTitle = true
				}
			} else if eqClause, ok := condition.(squirrel.Eq); ok {
				if missing, exists := eqClause["missing"]; exists && missing == false {
					foundMissing = true
				}
			}
		}
		return foundArtist && foundTitle && foundMissing
	}
}
