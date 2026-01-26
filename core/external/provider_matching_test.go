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

		setupExpectations := func(returnedSongs []agents.Song, idMatches, mbidMatches, artistTracks model.MediaFiles) {
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

			// loadTracksByTitleAndArtist - now queries by artist name
			mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				and, ok := opt.Filters.(squirrel.And)
				if !ok || len(and) < 2 {
					return false
				}
				eq, hasEq := and[0].(squirrel.Eq)
				if !hasEq {
					return false
				}
				_, hasArtist := eq["order_artist_name"]
				return hasArtist
			})).Return(artistTracks, nil).Maybe()
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

			It("does not match songs without artist info", func() {
				// Songs without artist info cannot be matched since we query by artist
				returnedSongs := []agents.Song{
					{Name: "Similar Song"}, // No artist/album info at all
				}

				// No artist to query, so no GetAll calls for title matching
				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{})

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(BeEmpty())
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
				artistTwoMatch := model.MediaFile{
					ID: "artist-two", Title: "Song B", Artist: "Artist Two",
				}

				returnedSongs := []agents.Song{
					{Name: "Song A", Artist: "Artist One", ArtistMBID: "mbid-1", Album: "Album One", AlbumMBID: "album-mbid-1"},
					{Name: "Song B", Artist: "Artist Two"}, // Different artist
				}

				setupExpectations(returnedSongs, model.MediaFiles{}, model.MediaFiles{}, model.MediaFiles{lessAccurateMatch, preciseMatch, artistTwoMatch})

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(2))
				// First song should be the precise match (has all MBIDs)
				Expect(songs[0].ID).To(Equal("precise"))
				// Second song matches by title + artist
				Expect(songs[1].ID).To(Equal("artist-two"))
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

		setupFuzzyExpectations := func(returnedSongs []agents.Song, artistTracks model.MediaFiles) {
			agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(returnedSongs, nil).Once()

			// loadTracksByTitleAndArtist now queries by artist in a single pass
			// Note: loadTracksByID and loadTracksByMBID return early when no IDs/MBIDs
			mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				and, ok := opt.Filters.(squirrel.And)
				if !ok || len(and) < 2 {
					return false
				}
				eq, hasEq := and[0].(squirrel.Eq)
				if !hasEq {
					return false
				}
				_, hasArtist := eq["order_artist_name"]
				return hasArtist
			})).Return(artistTracks, nil).Maybe()
		}

		Context("with default threshold (85%)", func() {
			It("matches songs with remastered suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				// Agent returns "Paranoid Android" but library has "Paranoid Android - Remastered"
				returnedSongs := []agents.Song{
					{Name: "Paranoid Android", Artist: "Radiohead"},
				}
				// Artist catalog has the remastered version (fuzzy match will find it)
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead"},
				}

				setupFuzzyExpectations(returnedSongs, artistTracks)

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
				artistTracks := model.MediaFiles{
					{ID: "live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen"},
				}

				setupFuzzyExpectations(returnedSongs, artistTracks)

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
				// Artist catalog has completely different songs
				artistTracks := model.MediaFiles{
					{ID: "different", Title: "Tomorrow Never Knows", Artist: "The Beatles"},
					{ID: "different2", Title: "Here Comes The Sun", Artist: "The Beatles"},
				}

				setupFuzzyExpectations(returnedSongs, artistTracks)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(BeEmpty())
			})
		})

		Context("with threshold set to 100 (exact match only)", func() {
			It("only matches exact titles", func() {
				conf.Server.SimilarSongsMatchThreshold = 100

				returnedSongs := []agents.Song{
					{Name: "Paranoid Android", Artist: "Radiohead"},
				}
				// Artist catalog has only remastered version - no exact match
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead"},
				}

				setupFuzzyExpectations(returnedSongs, artistTracks)

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
				artistTracks := model.MediaFiles{
					{ID: "extended", Title: "Song (Extended Mix)", Artist: "Artist"},
				}

				setupFuzzyExpectations(returnedSongs, artistTracks)

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("extended"))
			})
		})

		Context("with fuzzy album matching", func() {
			It("matches album with (Remaster) suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				// Agent returns "A Night at the Opera" but library has remastered version
				returnedSongs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera"},
				}
				// Library has same album with remaster suffix
				correctMatch := model.MediaFile{
					ID: "correct", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera (2011 Remaster)",
				}
				wrongMatch := model.MediaFile{
					ID: "wrong", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "Greatest Hits",
				}

				setupFuzzyExpectations(returnedSongs, model.MediaFiles{wrongMatch, correctMatch})

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				// Should prefer the fuzzy album match (Level 3) over title+artist only (Level 1)
				Expect(songs[0].ID).To(Equal("correct"))
			})

			It("matches album with (Deluxe Edition) suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				returnedSongs := []agents.Song{
					{Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"},
				}
				correctMatch := model.MediaFile{
					ID: "correct", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator (Deluxe Edition)",
				}
				wrongMatch := model.MediaFile{
					ID: "wrong", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "101",
				}

				setupFuzzyExpectations(returnedSongs, model.MediaFiles{wrongMatch, correctMatch})

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("correct"))
			})

			It("prefers exact album match over fuzzy album match", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				returnedSongs := []agents.Song{
					{Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"},
				}
				exactMatch := model.MediaFile{
					ID: "exact", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				}
				fuzzyMatch := model.MediaFile{
					ID: "fuzzy", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator (Deluxe Edition)",
				}

				setupFuzzyExpectations(returnedSongs, model.MediaFiles{fuzzyMatch, exactMatch})

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				// Both have same title similarity (1.0), so should prefer exact album match (higher specificity via higher album similarity)
				Expect(songs[0].ID).To(Equal("exact"))
			})
		})
	})
})
