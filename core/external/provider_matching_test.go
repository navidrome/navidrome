package external_test

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - Song Matching", func() {
	var ds model.DataStore
	var provider Provider
	var agentsCombined *mockAgents
	var artistRepo *mockArtistRepo
	var mediaFileRepo *mockMediaFileRepo
	var albumRepo *mockAlbumRepo
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		artistRepo = newMockArtistRepo()
		mediaFileRepo = newMockMediaFileRepo()
		albumRepo = newMockAlbumRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
			MockedAlbum:     albumRepo,
		}

		agentsCombined = &mockAgents{}
		provider = NewProvider(ds, agentsCombined)
	})

	Describe("matchSongsToLibrary priority matching", func() {
		var track model.MediaFile

		BeforeEach(func() {
			track = model.MediaFile{ID: "track-1", Title: "Test Track", Artist: "Test Artist", MbzRecordingID: ""}

			// Setup for GetEntityByID to return the track
			artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
			albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
			mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()
		})

		Context("when agent returns artist and album metadata", func() {
			It("matches by title + artist MBID + album MBID (highest priority)", func() {
				// Song in library with all MBIDs
				correctMatch := model.MediaFile{
					ID: "correct-match", Title: "Similar Song", Artist: "Depeche Mode", Album: "Violator",
					MbzArtistID: "artist-mbid-123", MbzAlbumID: "album-mbid-456",
				}
				// Another song with same title but different MBIDs (should NOT match)
				wrongMatch := model.MediaFile{
					ID: "wrong-match", Title: "Similar Song", Artist: "Depeche Mode", Album: "Some Other Album",
					MbzArtistID: "artist-mbid-123", MbzAlbumID: "different-album-mbid",
				}

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{
						{Name: "Similar Song", Artist: "Depeche Mode", ArtistMBID: "artist-mbid-123", Album: "Violator", AlbumMBID: "album-mbid-456"},
					}, nil).Once()

				// loadTracksByID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByMBID returns empty (no song MBID)
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					eq, hasEq := and[0].(squirrel.Eq)
					if !hasEq {
						return false
					}
					_, hasMBID := eq["mbz_recording_id"]
					return hasMBID
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByTitleAndArtist returns both songs
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 2 {
						return false
					}
					_, hasOr := and[0].(squirrel.Or)
					return hasOr
				})).Return(model.MediaFiles{wrongMatch, correctMatch}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("correct-match"))
			})

			It("matches by title + artist name + album name when MBIDs unavailable", func() {
				// Song in library without MBIDs but with matching artist/album names
				correctMatch := model.MediaFile{
					ID: "correct-match", Title: "Similar Song", Artist: "depeche mode", Album: "violator",
				}
				// Another song with same title but different artist (should NOT match)
				wrongMatch := model.MediaFile{
					ID: "wrong-match", Title: "Similar Song", Artist: "Other Artist", Album: "Other Album",
				}

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{
						{Name: "Similar Song", Artist: "Depeche Mode", Album: "Violator"}, // No MBIDs
					}, nil).Once()

				// loadTracksByID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByMBID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					eq, hasEq := and[0].(squirrel.Eq)
					if !hasEq {
						return false
					}
					_, hasMBID := eq["mbz_recording_id"]
					return hasMBID
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByTitleAndArtist returns both songs
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 2 {
						return false
					}
					_, hasOr := and[0].(squirrel.Or)
					return hasOr
				})).Return(model.MediaFiles{wrongMatch, correctMatch}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("correct-match"))
			})

			It("matches by title + artist only when album info unavailable", func() {
				// Song in library with matching artist
				correctMatch := model.MediaFile{
					ID: "correct-match", Title: "Similar Song", Artist: "depeche mode", Album: "Some Album",
				}
				// Another song with same title but different artist
				wrongMatch := model.MediaFile{
					ID: "wrong-match", Title: "Similar Song", Artist: "Other Artist", Album: "Other Album",
				}

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{
						{Name: "Similar Song", Artist: "Depeche Mode"}, // No album info
					}, nil).Once()

				// loadTracksByID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByMBID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					eq, hasEq := and[0].(squirrel.Eq)
					if !hasEq {
						return false
					}
					_, hasMBID := eq["mbz_recording_id"]
					return hasMBID
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByTitleAndArtist returns both songs
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 2 {
						return false
					}
					_, hasOr := and[0].(squirrel.Or)
					return hasOr
				})).Return(model.MediaFiles{wrongMatch, correctMatch}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("correct-match"))
			})

			It("falls back to title-only match when no artist info available", func() {
				// Song in library
				titleMatch := model.MediaFile{
					ID: "title-match", Title: "Similar Song", Artist: "Random Artist",
				}

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{
						{Name: "Similar Song"}, // No artist/album info at all
					}, nil).Once()

				// loadTracksByID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByMBID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					eq, hasEq := and[0].(squirrel.Eq)
					if !hasEq {
						return false
					}
					_, hasMBID := eq["mbz_recording_id"]
					return hasMBID
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByTitleAndArtist returns the song
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 2 {
						return false
					}
					_, hasOr := and[0].(squirrel.Or)
					return hasOr
				})).Return(model.MediaFiles{titleMatch}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("title-match"))
			})
		})

		Context("when matching multiple songs with the same title but different artists", func() {
			It("returns distinct matches for each artist's version (covers scenario)", func() {
				// Multiple covers of the same song by different artists
				cover1 := model.MediaFile{
					ID: "cover-1", Title: "Yesterday", Artist: "The Beatles", Album: "Help!",
				}
				cover2 := model.MediaFile{
					ID: "cover-2", Title: "Yesterday", Artist: "Ray Charles", Album: "Greatest Hits",
				}
				cover3 := model.MediaFile{
					ID: "cover-3", Title: "Yesterday", Artist: "Frank Sinatra", Album: "My Way",
				}

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{
						{Name: "Yesterday", Artist: "The Beatles", Album: "Help!"},
						{Name: "Yesterday", Artist: "Ray Charles", Album: "Greatest Hits"},
						{Name: "Yesterday", Artist: "Frank Sinatra", Album: "My Way"},
					}, nil).Once()

				// loadTracksByID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByMBID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					eq, hasEq := and[0].(squirrel.Eq)
					if !hasEq {
						return false
					}
					_, hasMBID := eq["mbz_recording_id"]
					return hasMBID
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByTitleAndArtist returns all three covers
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 2 {
						return false
					}
					_, hasOr := and[0].(squirrel.Or)
					return hasOr
				})).Return(model.MediaFiles{cover1, cover2, cover3}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				// All three covers should be returned, not just the first one
				Expect(songs).To(HaveLen(3))
				// Verify all three different versions are included
				ids := []string{songs[0].ID, songs[1].ID, songs[2].ID}
				Expect(ids).To(ContainElements("cover-1", "cover-2", "cover-3"))
			})
		})

		Context("when matching multiple songs with different precision levels", func() {
			It("prefers more precise matches for each song", func() {
				// Library has multiple versions of same song
				preciseMatch := model.MediaFile{
					ID: "precise", Title: "Song A", Artist: "Artist One", Album: "Album One",
					MbzArtistID: "mbid-1", MbzAlbumID: "album-mbid-1",
				}
				lessAccurateMatch := model.MediaFile{
					ID: "less-accurate", Title: "Song A", Artist: "Artist One", Album: "Compilation",
					MbzArtistID: "mbid-1",
				}
				titleOnlyMatch := model.MediaFile{
					ID: "title-only", Title: "Song B", Artist: "Different Artist",
				}

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{
						{Name: "Song A", Artist: "Artist One", ArtistMBID: "mbid-1", Album: "Album One", AlbumMBID: "album-mbid-1"},
						{Name: "Song B"}, // Title only
					}, nil).Once()

				// loadTracksByID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByMBID returns empty
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					eq, hasEq := and[0].(squirrel.Eq)
					if !hasEq {
						return false
					}
					_, hasMBID := eq["mbz_recording_id"]
					return hasMBID
				})).Return(model.MediaFiles{}, nil).Once()

				// loadTracksByTitleAndArtist returns all candidates
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 2 {
						return false
					}
					_, hasOr := and[0].(squirrel.Or)
					return hasOr
				})).Return(model.MediaFiles{lessAccurateMatch, preciseMatch, titleOnlyMatch}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(2))
				// First song should be the precise match (has all MBIDs)
				Expect(songs[0].ID).To(Equal("precise"))
				// Second song should be title-only match
				Expect(songs[1].ID).To(Equal("title-only"))
			})
		})
	})
})
