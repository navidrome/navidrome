package external_test

import (
	"context"
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
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
	var playlistRepo *tests.MockPlaylistRepo
	var playlistTrackRepo *mockPlaylistTrackRepo
	var genreRepo *tests.MockedGenreRepo
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		artistRepo = newMockArtistRepo()
		mediaFileRepo = newMockMediaFileRepo()
		albumRepo = newMockAlbumRepo()
		playlistTrackRepo = &mockPlaylistTrackRepo{}
		playlistRepo = tests.CreateMockPlaylistRepo()
		playlistRepo.TracksRepo = playlistTrackRepo
		genreRepo = &tests.MockedGenreRepo{}

		ds = &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
			MockedAlbum:     albumRepo,
			MockedPlaylist:  playlistRepo,
			MockedGenre:     genreRepo,
		}

		mockAgent = &mockSimilarArtistAgent{}
		mockTopAgent = mockAgent
		mockSimilarAgent = mockAgent

		agentsCombined = &mockAgents{
			topSongsAgent: mockTopAgent,
			similarAgent:  mockSimilarAgent,
		}

		provider = NewProvider(ds, agentsCombined, matcher.New(ds))
	})

	Describe("dispatch by entity type", func() {
		Context("when ID is a MediaFile (track)", func() {
			It("calls GetSimilarSongsByTrack and returns matched songs", func() {
				track := model.MediaFile{ID: "track-1", Title: "Just Can't Get Enough", Artist: "Depeche Mode", MbzRecordingID: "track-mbid"}

				// Depeche Mode artist row used by matcher artist resolution and track-fetch back-mapping.
				dmArtist := model.Artist{ID: "dm-1", Name: "Depeche Mode", OrderArtistName: "depeche mode", MbzArtistID: "artist-mbid"}
				dmParticipant := model.Participant{Artist: dmArtist}
				matchedSong := model.MediaFile{
					ID: "matched-1", Title: "Dreaming of Me", Artist: "Depeche Mode",
					Participants: model.Participants{model.RoleArtist: model.ParticipantList{dmParticipant}},
				}

				// GetEntityByID tries Artist, Album, Playlist, then MediaFile
				artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Once()
				mediaFileRepo.On("Get", "track-1").Return(&track, nil).Once()

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Just Can't Get Enough", "Depeche Mode", "track-mbid", 5).
					Return([]agents.Song{
						{Name: "Dreaming of Me", MBID: "", Artists: []agents.Artist{{Name: "Depeche Mode", MBID: "artist-mbid"}}},
					}, nil).Once()

				// Matcher artist resolution: resolve Depeche Mode in the artist table.
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{dmArtist}, nil).Maybe()

				// ID phase: no IDs → squirrel.And with media_file.id; won't be called but guard it.
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					_, ok := opt.Filters.(squirrel.Eq)
					return ok
				})).Return(model.MediaFiles{}, nil).Maybe()

				// MBID phase: won't fire (empty MBID).
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

				// Matcher track-fetch: subquery returns the matched song with participants.
				mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
					and, ok := opt.Filters.(squirrel.And)
					if !ok {
						return false
					}
					for _, f := range and {
						sql, _, err := f.ToSql()
						if err == nil && strings.Contains(sql, "media_file_artists") {
							return true
						}
					}
					return false
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
						{Name: "New Life", MBID: "song-mbid", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
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

			It("falls back to sampled album tracks when GetSimilarSongsByAlbum returns ErrNotFound", func() {
				album := model.Album{ID: "album-1", Name: "Album", AlbumArtist: "Artist", AlbumArtistID: "artist-1"}
				seed := model.MediaFile{ID: "seed-1", Title: "Seed", Artist: "Artist"}

				artistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "album-1").Return(&album, nil).Once()

				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "album-1", "Album", "Artist", "", mock.Anything).
					Return(nil, agents.ErrNotFound).Once()

				mediaFileRepo.On("GetRandom", mock.MatchedBy(func(opt model.QueryOptions) bool {
					eq, ok := opt.Filters.(squirrel.Eq)
					return ok && eq["album_id"] == "album-1"
				})).Return(model.MediaFiles{seed}, nil).Once()

				// seedMix falls back to the seed itself when the agent finds nothing.
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "seed-1", "Seed", "Artist", "", mock.Anything).
					Return([]agents.Song{}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "album-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("seed-1"))
			})
		})

		Context("when ID is an Album and the album agent returns nothing (AudioMuse-only)", func() {
			It("samples the album's tracks and returns their track-similars", func() {
				album := model.Album{ID: "al-1", Name: "The Album", AlbumArtist: "A"}
				artistRepo.On("Get", "al-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "al-1").Return(&album, nil).Once()

				// AudioMuse doesn't implement album similarity -> empty.
				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "al-1", "The Album", "A", "", 5).
					Return([]agents.Song{}, nil).Once()

				// sampleAlbumTracks -> GetRandom(album_id) -> one seed track
				mediaFileRepo.On("GetRandom", mock.MatchedBy(func(opt model.QueryOptions) bool {
					eq, ok := opt.Filters.(squirrel.Eq)
					return ok && eq["album_id"] == "al-1"
				})).Return(model.MediaFiles{{ID: "s1", Title: "Seed", Artist: "A"}}, nil).Once()

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "A", "", 5).
					Return([]agents.Song{{Name: "AudioMuseResult", Artists: []agents.Artist{{Name: "A"}}}}, nil).Once()

				// Matcher resolves "AudioMuseResult" -> a real MediaFile credited to artist "A".
				aArtist := model.Artist{ID: "a1", Name: "A", OrderArtistName: "a"}
				matchedTrack := model.MediaFile{
					ID: "m1", Title: "AudioMuseResult", Artist: "A",
					Participants: model.Participants{model.RoleArtist: model.ParticipantList{{Artist: aArtist}}},
				}
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{aArtist}, nil).Maybe()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{matchedTrack}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "al-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).ToNot(BeEmpty())
				Expect(songs[0].ID).To(Equal("m1"))
			})
		})

		Context("when ID is an Artist", func() {
			It("calls GetSimilarSongsByArtist and returns matched songs", func() {
				artist := model.Artist{ID: "artist-1", Name: "Depeche Mode", MbzArtistID: "artist-mbid"}
				matchedSong := model.MediaFile{ID: "matched-1", Title: "Enjoy the Silence", Artist: "Depeche Mode", MbzRecordingID: "song-mbid"}

				artistRepo.On("Get", "artist-1").Return(&artist, nil).Once()
				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Depeche Mode", "artist-mbid", 5).
					Return([]agents.Song{
						{Name: "Enjoy the Silence", MBID: "song-mbid", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
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

		Context("when ID is an Artist and both the artist agent and the similar-artists fallback are empty", func() {
			It("samples the artist's tracks and returns their track-similars", func() {
				artist := model.Artist{ID: "ar-1", Name: "The Artist"}
				// Get is called twice: once to resolve the entity, once inside similarSongsFallback.
				artistRepo.On("Get", "ar-1").Return(&artist, nil).Maybe()

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "ar-1", "The Artist", "", 5).
					Return([]agents.Song{}, nil).Once()
				// similarSongsFallback: no similar artists, no top songs -> empty (allow its lookups).
				mockAgent.On("GetSimilarArtists", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Artist{}, nil).Maybe()
				mockAgent.On("GetArtistTopSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{}, nil).Maybe()

				mediaFileRepo.On("GetRandom", mock.MatchedBy(func(opt model.QueryOptions) bool {
					eq, ok := opt.Filters.(squirrel.Eq)
					return ok && eq["artist_id"] == "ar-1"
				})).Return(model.MediaFiles{{ID: "s1", Title: "Seed"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "", "", 5).
					Return([]agents.Song{{Name: "Result"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{{ID: "m1", Title: "Result"}}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "ar-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).ToNot(BeEmpty())
			})
		})

		Context("when ID is a Playlist", func() {
			It("samples playlist tracks and returns their track-similars", func() {
				pls := model.Playlist{ID: "pl-1", Name: "My List"}
				seedTrack := model.MediaFile{ID: "s1", Title: "Seed One", Artist: "A"}

				// GetEntityByID order: Artist, Album, Playlist(hit)
				artistRepo.On("Get", "pl-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-1").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})

				// samplePlaylistTracks -> Tracks(...).GetAll -> one seed track, bounded+randomized in SQL
				playlistTrackRepo.On("GetAll", model.QueryOptions{Sort: "random", Max: 5}).Return(model.PlaylistTracks{
					{MediaFile: seedTrack},
				}, nil).Once()

				// seedMix -> GetSimilarSongsByTrack for the seed
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "A", "", 5).
					Return([]agents.Song{{Name: "Similar", Artists: []agents.Artist{{Name: "A"}}}}, nil).Once()

				// Matcher resolves "Similar" -> a real MediaFile (allow the matcher's lookups).
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{{ID: "a1", Name: "A"}}, nil).Maybe()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{{ID: "m1", Title: "Similar"}}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "pl-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).ToNot(BeEmpty())
			})

			It("falls back to the seed tracks themselves when no similar songs are found", func() {
				pls := model.Playlist{ID: "pl-2", Name: "Fallback List"}
				seed1 := model.MediaFile{ID: "s1", Title: "Seed One", Artist: "A"}
				seed2 := model.MediaFile{ID: "s2", Title: "Seed Two", Artist: "B"}

				artistRepo.On("Get", "pl-2").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-2").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})

				playlistTrackRepo.On("GetAll", model.QueryOptions{Sort: "random", Max: 5}).Return(model.PlaylistTracks{
					{MediaFile: seed1},
					{MediaFile: seed2},
				}, nil).Once()

				// Both seeds come back empty, so the mix must fall back to the seeds themselves.
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "A", "", 5).
					Return([]agents.Song{}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "B", "", 5).
					Return([]agents.Song{}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-2", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(2))
				Expect([]string{songs[0].ID, songs[1].ID}).To(ConsistOf("s1", "s2"))
			})

			It("caps agent calls at maxSeeds even if the repo returns more than requested", func() {
				pls := model.Playlist{ID: "pl-3", Name: "Big List"}
				tracks := model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "seed-1", Title: "Seed 1", Artist: "A"}},
					{MediaFile: model.MediaFile{ID: "seed-2", Title: "Seed 2", Artist: "A"}},
					{MediaFile: model.MediaFile{ID: "seed-3", Title: "Seed 3", Artist: "A"}},
					{MediaFile: model.MediaFile{ID: "seed-4", Title: "Seed 4", Artist: "A"}},
					{MediaFile: model.MediaFile{ID: "seed-5", Title: "Seed 5", Artist: "A"}},
					{MediaFile: model.MediaFile{ID: "seed-6", Title: "Seed 6", Artist: "A"}},
				}

				artistRepo.On("Get", "pl-3").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-3").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})

				// The bound lives in the query; over-returning here proves seedMix still caps agent calls.
				playlistTrackRepo.On("GetAll", model.QueryOptions{Sort: "random", Max: 5}).Return(tracks, nil).Once()

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, 5).
					Return([]agents.Song{}, nil)

				songs, err := provider.SimilarSongs(ctx, "pl-3", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(5))
				agentsCombined.AssertNumberOfCalls(GinkgoT(), "GetSimilarSongsByTrack", 5)
			})
		})

		Context("when ID is a Genre (not resolved by GetEntityByID)", func() {
			It("samples genre songs and returns their track-similars", func() {
				// GetEntityByID misses everywhere; the empty/auto-created mocks need no setup.
				artistRepo.On("Get", "g-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "g-1").Return(nil, model.ErrNotFound).Once()
				mediaFileRepo.On("Get", "g-1").Return(nil, model.ErrNotFound).Once()
				genreRepo.Data = map[string]model.Genre{"g-1": {ID: "g-1", Name: "Jazz"}}

				// sampleGenreTracks -> GetRandom with the indexed media_file_tags semi-join (not a json_tree scan)
				mediaFileRepo.On("GetRandom", mock.MatchedBy(func(opt model.QueryOptions) bool {
					if opt.Filters == nil {
						return false
					}
					sql, _, err := opt.Filters.ToSql()
					return err == nil && strings.Contains(sql, "media_file_tags") && !strings.Contains(sql, "json_tree")
				})).Return(model.MediaFiles{{ID: "s1", Title: "Seed"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "", "", 5).
					Return([]agents.Song{{Name: "Similar"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{{ID: "m1", Title: "Similar"}}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "g-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).ToNot(BeEmpty())
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

		// Fallback yields nothing, so the sampling path is tried and also finds no tracks.
		mediaFileRepo.On("GetRandom", mock.MatchedBy(func(opt model.QueryOptions) bool {
			eq, ok := opt.Filters.(squirrel.Eq)
			return ok && eq["artist_id"] == "artist-1"
		})).Return(model.MediaFiles{}, nil).Once()

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
