package external_test

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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
			DeferCleanup(configtest.SetupConfig())
			// Disable fuzzy matching for these tests to avoid unexpected GetAll calls
			conf.Server.SimilarSongsMatchThreshold = 100

			track = model.MediaFile{ID: "track-1", Title: "Test Track", Artist: "Test Artist", MbzRecordingID: ""}

			// Setup for GetEntityByID to return the track
			artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
			albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
			mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()
		})

		setupExpectations := func(returnedSongs []agents.Song, idMatches, mbidMatches, titleMatches model.MediaFiles) {
			agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(returnedSongs, nil).Once()

			// loadTracksByID
			mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				_, ok := opt.Filters.(squirrel.Eq)
				return ok
			})).Return(idMatches, nil).Once()

			// loadTracksByMBID
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
			})).Return(mbidMatches, nil).Once()

			// loadTracksByTitleAndArtist
			mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				and, ok := opt.Filters.(squirrel.And)
				if !ok || len(and) < 2 {
					return false
				}
				_, hasOr := and[0].(squirrel.Or)
				return hasOr
			})).Return(titleMatches, nil).Once()
		}

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
				returnedSongs := []agents.Song{
					{Name: "Similar Song", Artist: "Depeche Mode", ArtistMBID: "artist-mbid-123", Album: "Violator", AlbumMBID: "album-mbid-456"},
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{wrongMatch, correctMatch})

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

				returnedSongs := []agents.Song{
					{Name: "Similar Song", Artist: "Depeche Mode", Album: "Violator"}, // No MBIDs
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{wrongMatch, correctMatch})

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
				returnedSongs := []agents.Song{
					{Name: "Similar Song", Artist: "Depeche Mode"}, // No album info
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{wrongMatch, correctMatch})

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

				returnedSongs := []agents.Song{
					{Name: "Similar Song"}, // No artist/album info at all
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{titleMatch})

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

				returnedSongs := []agents.Song{
					{Name: "Yesterday", Artist: "The Beatles", Album: "Help!"},
					{Name: "Yesterday", Artist: "Ray Charles", Album: "Greatest Hits"},
					{Name: "Yesterday", Artist: "Frank Sinatra", Album: "My Way"},
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{cover1, cover2, cover3})

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

				returnedSongs := []agents.Song{
					{Name: "Song A", Artist: "Artist One", ArtistMBID: "mbid-1", Album: "Album One", AlbumMBID: "album-mbid-1"},
					{Name: "Song B"}, // Title only
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{lessAccurateMatch, preciseMatch, titleOnlyMatch})

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

	Describe("Fuzzy matching fallback", func() {
		var track model.MediaFile

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			track = model.MediaFile{ID: "track-1", Title: "Test Track", Artist: "Test Artist"}

			// Setup for GetEntityByID to return the track
			artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
			albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
			mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()
		})

		setupFuzzyExpectations := func(returnedSongs []agents.Song, titleMatches, artistTracks model.MediaFiles, expectFuzzyCall bool) {
			agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(returnedSongs, nil).Once()

			// loadTracksByTitleAndArtist - exact title match returns titleMatches
			// Note: loadTracksByID and loadTracksByMBID return early when no IDs/MBIDs
			mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(titleMatches, nil).Once()

			// fuzzyMatchUnmatched - query by artist returns artistTracks
			if expectFuzzyCall {
				mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(artistTracks, nil).Once()
			}
		}

		Context("with default threshold (85%)", func() {
			It("matches songs with remastered suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				// Agent returns "Paranoid Android" but library has "Paranoid Android - Remastered"
				returnedSongs := []agents.Song{
					{Name: "Paranoid Android", Artist: "Radiohead"},
				}
				// No exact title match
				titleMatches := model.MediaFiles{}
				// Artist catalog has the remastered version
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead"},
				}

				setupFuzzyExpectations(returnedSongs, titleMatches, artistTracks, true)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("remastered"))
			})

			It("matches songs with live suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				returnedSongs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artist: "Queen"},
				}
				titleMatches := model.MediaFiles{}
				artistTracks := model.MediaFiles{
					{ID: "live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen"},
				}

				setupFuzzyExpectations(returnedSongs, titleMatches, artistTracks, true)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("live"))
			})

			It("does not match completely different songs", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				returnedSongs := []agents.Song{
					{Name: "Yesterday", Artist: "The Beatles"},
				}
				titleMatches := model.MediaFiles{}
				// Artist catalog has completely different songs
				artistTracks := model.MediaFiles{
					{ID: "different", Title: "Tomorrow Never Knows", Artist: "The Beatles"},
					{ID: "different2", Title: "Here Comes The Sun", Artist: "The Beatles"},
				}

				setupFuzzyExpectations(returnedSongs, titleMatches, artistTracks, true)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(BeEmpty())
			})
		})

		Context("with threshold set to 100 (disabled)", func() {
			It("does not perform fuzzy matching", func() {
				conf.Server.SimilarSongsMatchThreshold = 100

				returnedSongs := []agents.Song{
					{Name: "Paranoid Android", Artist: "Radiohead"},
				}
				// No exact title match, and fuzzy matching is disabled
				titleMatches := model.MediaFiles{}

				setupFuzzyExpectations(returnedSongs, titleMatches, model.MediaFiles{}, false)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(BeEmpty())
			})
		})

		Context("with lower threshold (75%)", func() {
			It("matches more aggressively", func() {
				conf.Server.SimilarSongsMatchThreshold = 75

				returnedSongs := []agents.Song{
					{Name: "Song", Artist: "Artist"},
				}
				titleMatches := model.MediaFiles{}
				artistTracks := model.MediaFiles{
					{ID: "extended", Title: "Song (Extended Mix)", Artist: "Artist"},
				}

				setupFuzzyExpectations(returnedSongs, titleMatches, artistTracks, true)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("extended"))
			})
		})
	})
})
