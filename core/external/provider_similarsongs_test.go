package external_test

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - SimilarSongs", func() {
	var ds model.DataStore
	var provider Provider
	var mockAgent *mockSimilarArtistAgent
	var mockTopAgent agents.ArtistTopSongsRetriever
	var mockSimilarAgent agents.ArtistSimilarRetriever
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

		mockAgent = &mockSimilarArtistAgent{}
		mockTopAgent = mockAgent
		mockSimilarAgent = mockAgent

		agentsCombined = &mockAgents{
			topSongsAgent: mockTopAgent,
			similarAgent:  mockSimilarAgent,
		}

		provider = NewProvider(ds, agentsCombined)
	})

	Describe("dispatch by entity type", func() {
		Context("when ID is a MediaFile (track)", func() {
			It("calls GetSimilarSongsByTrack and returns matched songs", func() {
				track := model.MediaFile{ID: "track-1", Title: "Just Can't Get Enough", Artist: "Depeche Mode", MbzRecordingID: "track-mbid"}
				matchedSong := model.MediaFile{ID: "matched-1", Title: "Dreaming of Me", Artist: "Depeche Mode"}

				// GetEntityByID tries Artist, Album, Playlist, then MediaFile
				artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Just Can't Get Enough", "Depeche Mode", "track-mbid", 5).
					Return([]agents.Song{
						{Name: "Dreaming of Me", MBID: "", Artist: "Depeche Mode", ArtistMBID: "artist-mbid"},
					}, nil).Once()

				// Mock loadTracksByID - no ID matches
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// Mock loadTracksByMBID - no MBID matches (empty MBID means this won't be called)
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
				})).Return(model.MediaFiles{}, nil).Maybe()

				// Mock loadTracksByTitleAndArtist - queries by artist name
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
				})).Return(model.MediaFiles{matchedSong}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("matched-1"))
			})

			It("falls back to artist-based algorithm when GetSimilarSongsByTrack returns empty", func() {
				track := model.MediaFile{ID: "track-1", Title: "Track", Artist: "Artist", ArtistID: "artist-1"}
				artist := model.Artist{ID: "artist-1", Name: "Artist"}
				song := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}

				// GetEntityByID for the initial call tries Artist, Album, Playlist, then MediaFile
				artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", mock.Anything).
					Return([]agents.Song{}, nil).Once()

				// Fallback calls getArtist(id) which calls GetEntityByID again - this time it finds the mediafile
				// and recursively calls getArtist(v.ArtistID)
				artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()

				// Then it recurses with the artist-1 ID
				artistRepo.On("Get", "artist-1").Return(&artist, nil).Maybe()
				artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					return opt.Max == 1 && opt.Filters != nil
				})).Return(model.Artists{artist}, nil).Maybe()

				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()

				artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					return opt.Max == 0 && opt.Filters != nil
				})).Return(model.Artists{}, nil).Once()

				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song One", MBID: "mbid-1"}}, nil).Once()

				mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("song-1"))
			})
		})

		Context("when ID is an Album", func() {
			It("calls GetSimilarSongsByAlbum and returns matched songs", func() {
				album := model.Album{ID: "album-1", Name: "Speak & Spell", AlbumArtist: "Depeche Mode", MbzAlbumID: "album-mbid"}
				matchedSong := model.MediaFile{ID: "matched-1", Title: "New Life", Artist: "Depeche Mode", MbzRecordingID: "song-mbid"}

				// GetEntityByID tries Artist, Album, Playlist, then MediaFile
				artistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "album-1").Return(&album, nil).Once()

				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "album-1", "Speak & Spell", "Depeche Mode", "album-mbid", 5).
					Return([]agents.Song{
						{Name: "New Life", MBID: "song-mbid", Artist: "Depeche Mode"},
					}, nil).Once()

				// Mock loadTracksByID - no ID matches
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// Mock loadTracksByMBID - MBID match
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					_, hasEq := and[0].(squirrel.Eq)
					return hasEq
				})).Return(model.MediaFiles{matchedSong}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "album-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("matched-1"))
			})

			It("falls back when GetSimilarSongsByAlbum returns ErrNotFound", func() {
				album := model.Album{ID: "album-1", Name: "Album", AlbumArtist: "Artist", AlbumArtistID: "artist-1"}
				artist := model.Artist{ID: "artist-1", Name: "Artist"}
				song := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}

				// GetEntityByID for the initial call tries Artist, Album, Playlist, then MediaFile
				artistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "album-1").Return(&album, nil).Once()

				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "album-1", "Album", "Artist", "", mock.Anything).
					Return(nil, agents.ErrNotFound).Once()

				// Fallback calls getArtist(id) which calls GetEntityByID again - this time it finds the album
				// and recursively calls getArtist(v.AlbumArtistID)
				artistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "album-1").Return(&album, nil).Once()

				// Then it recurses with the artist-1 ID
				artistRepo.On("Get", "artist-1").Return(&artist, nil).Maybe()
				artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					return opt.Max == 1 && opt.Filters != nil
				})).Return(model.Artists{artist}, nil).Maybe()

				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()

				artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					return opt.Max == 0 && opt.Filters != nil
				})).Return(model.Artists{}, nil).Once()

				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song One", MBID: "mbid-1"}}, nil).Once()

				mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "album-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("song-1"))
			})
		})

		Context("when ID is an Artist", func() {
			It("calls GetSimilarSongsByArtist and returns matched songs", func() {
				artist := model.Artist{ID: "artist-1", Name: "Depeche Mode", MbzArtistID: "artist-mbid"}
				matchedSong := model.MediaFile{ID: "matched-1", Title: "Enjoy the Silence", Artist: "Depeche Mode", MbzRecordingID: "song-mbid"}

				artistRepo.On("Get", "artist-1").Return(&artist, nil).Once()
				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Depeche Mode", "artist-mbid", 5).
					Return([]agents.Song{
						{Name: "Enjoy the Silence", MBID: "song-mbid", Artist: "Depeche Mode"},
					}, nil).Once()

				// Mock loadTracksByID - no ID matches
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Once()

				// Mock loadTracksByMBID - MBID match
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok || len(and) < 1 {
						return false
					}
					_, hasEq := and[0].(squirrel.Eq)
					return hasEq
				})).Return(model.MediaFiles{matchedSong}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "artist-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("matched-1"))
			})
		})
	})

	It("returns similar songs from main artist and similar artists", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		similarArtist := model.Artist{ID: "artist-3", Name: "Similar Artist"}
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-2"}
		song3 := model.MediaFile{ID: "song-3", Title: "Song Three", ArtistID: "artist-3", MbzRecordingID: "mbid-3"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("Get", "artist-3").Return(&similarArtist, nil).Maybe()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Once()

		// New similar songs by artist returns ErrNotFound to trigger fallback
		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()

		similarAgentsResp := []agents.Artist{
			{Name: "Similar Artist", MBID: "similar-mbid"},
		}
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return(similarAgentsResp, nil).Once()

		// Mock the three-phase artist lookup: ID (skipped - no IDs), MBID, then Name
		// MBID lookup returns empty (no match)
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			_, ok := opt.Filters.(squirrel.Eq)
			return opt.Max == 0 && ok
		})).Return(model.Artists{}, nil).Once()
		// Name lookup returns the similar artist
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			_, ok := opt.Filters.(squirrel.Or)
			return opt.Max == 0 && ok
		})).Return(model.Artists{similarArtist}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-3", "Similar Artist", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song Three", MBID: "mbid-3"},
			}, nil).Once()

		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song3}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 3)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(3))
		for _, song := range songs {
			Expect(song.ID).To(BeElementOf("song-1", "song-2", "song-3"))
		}
	})

	It("returns ErrNotFound when artist is not found", func() {
		artistRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)
		mediaFileRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)
		albumRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Maybe()

		songs, err := provider.SimilarSongs(ctx, "artist-unknown-artist", 5)

		Expect(err).To(Equal(model.ErrNotFound))
		Expect(songs).To(BeNil())
	})

	It("returns songs from main artist when GetSimilarArtists returns error", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		// New similar songs by artist returns ErrNotFound to trigger fallback
		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return(nil, errors.New("error getting similar artists")).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
			}, nil).Once()

		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 5)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(Equal("song-1"))
	})

	It("returns empty list when GetArtistTopSongs returns error", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		// New similar songs by artist returns ErrNotFound to trigger fallback
		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{}, nil).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, errors.New("error getting top songs")).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 5)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(BeEmpty())
	})

	It("respects count parameter", func() {
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"}
		song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-2"}

		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		// New similar songs by artist returns ErrNotFound to trigger fallback
		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{}, nil).Once()

		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 0 && opt.Filters != nil
		})).Return(model.Artists{}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}, nil).Once()

		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{song1, song2}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(HaveLen(1))
		Expect(songs[0].ID).To(BeElementOf("song-1", "song-2"))
	})
})
