package external_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/slice"
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
	var playlistTrackRepo *tests.MockPlaylistTrackRepo
	var genreRepo *tests.MockedGenreRepo
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		artistRepo = newMockArtistRepo()
		mediaFileRepo = newMockMediaFileRepo()
		albumRepo = newMockAlbumRepo()
		playlistTrackRepo = &tests.MockPlaylistTrackRepo{}
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

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Just Can't Get Enough", "Depeche Mode", "track-mbid", 1).
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

				songs, err := provider.SimilarSongs(ctx, "track-1", 1)

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

			It("tops the mix up with the fallback when the agent's picks alone are too few", func() {
				track := model.MediaFile{ID: "track-1", Title: "Track", Artist: "Artist", ArtistID: "artist-1"}
				artist := model.Artist{ID: "artist-1", Name: "Artist"}

				artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Twice()
				albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Twice()
				mediaFileRepo.On("Get", "track-1").Return(&track, nil).Twice()
				artistRepo.On("Get", "artist-1").Return(&artist, nil).Maybe()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{artist}, nil).Maybe()

				// The agent knows one track of the three asked for.
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", 3).
					Return([]agents.Song{{Name: "Agent Pick", MBID: "mbid-agent"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).
					Return(model.MediaFiles{{ID: "agent-1", Title: "Agent Pick", MbzRecordingID: "mbid-agent"}}, nil).Once()

				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song One", MBID: "mbid-1"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).
					Return(model.MediaFiles{{ID: "song-1", Title: "Song One", MbzRecordingID: "mbid-1"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(songs, func(mf model.MediaFile) string { return mf.ID })).
					To(ConsistOf("agent-1", "song-1"))
			})

			It("keeps topping up when the agent's picks repeat a track", func() {
				track := model.MediaFile{ID: "track-1", Title: "Track", Artist: "Artist", ArtistID: "artist-1"}
				artist := model.Artist{ID: "artist-1", Name: "Artist"}

				artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Twice()
				albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Twice()
				mediaFileRepo.On("Get", "track-1").Return(&track, nil).Twice()
				artistRepo.On("Get", "artist-1").Return(&artist, nil).Maybe()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{artist}, nil).Maybe()

				// The matcher re-emits a track when the same input song repeats, so these three
				// picks resolve to only two distinct library tracks.
				repeated := agents.Song{Name: "Song A", MBID: "mbid-a"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", 3).
					Return([]agents.Song{repeated, repeated, {Name: "Song B", MBID: "mbid-b"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{
					{ID: "t-a", Title: "Song A", MbzRecordingID: "mbid-a"},
					{ID: "t-b", Title: "Song B", MbzRecordingID: "mbid-b"},
				}, nil).Once()

				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song C", MBID: "mbid-c"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).
					Return(model.MediaFiles{{ID: "t-c", Title: "Song C", MbzRecordingID: "mbid-c"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(songs, func(mf model.MediaFile) string { return mf.ID })).
					To(ConsistOf("t-a", "t-b", "t-c"))
			})
		})

		Context("when ID is an Album", func() {
			It("calls GetSimilarSongsByAlbum and returns matched songs", func() {
				album := model.Album{ID: "album-1", Name: "Speak & Spell", AlbumArtist: "Depeche Mode", MbzAlbumID: "album-mbid"}
				matchedSong := model.MediaFile{ID: "matched-1", Title: "New Life", Artist: "Depeche Mode", MbzRecordingID: "song-mbid"}

				// GetEntityByID tries Artist, Album, Playlist, then MediaFile
				artistRepo.On("Get", "album-1").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "album-1").Return(&album, nil).Once()

				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "album-1", "Speak & Spell", "Depeche Mode", "album-mbid", 1).
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

				songs, err := provider.SimilarSongs(ctx, "album-1", 1)

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
					sql, args, err := opt.Filters.ToSql()
					return err == nil && strings.Contains(sql, "album_id") &&
						strings.Contains(sql, "missing") && slices.Contains(args, any(false)) && slices.Contains(args, any("album-1"))
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
					sql, args, err := opt.Filters.ToSql()
					return err == nil && strings.Contains(sql, "album_id") &&
						strings.Contains(sql, "missing") && slices.Contains(args, any(false)) && slices.Contains(args, any("al-1"))
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
				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Depeche Mode", "artist-mbid", 1).
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

				songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

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

				// Seeds come from the participant join covering both roles, so an artist credited
				// only on the album (compilations, classical) still yields seeds.
				mediaFileRepo.On("GetRandom", mock.MatchedBy(func(opt model.QueryOptions) bool {
					sql, args, err := opt.Filters.ToSql()
					return err == nil && strings.Contains(sql, "media_file_artists") &&
						strings.Contains(sql, "missing") && slices.Contains(args, any(false)) && slices.Contains(args, any("ar-1")) &&
						slices.Contains(args, any(model.RoleAlbumArtist.String())) && slices.Contains(args, any(model.RoleArtist.String()))
				})).Return(model.MediaFiles{{ID: "s1", Title: "Seed"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "", "", 5).
					Return([]agents.Song{{Name: "Result"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{{ID: "m1", Title: "Result"}}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "ar-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).ToNot(BeEmpty())
			})
		})

		Context("when ID is an Artist and the similar-artists fallback can't fill the mix", func() {
			It("tops the mix up with the artist's own track-similars", func() {
				artist := model.Artist{ID: "ar-1", Name: "Thin Artist"}
				topSong := model.MediaFile{ID: "top-1", Title: "Top Song", ArtistID: "ar-1", MbzRecordingID: "mbid-top"}

				artistRepo.On("Get", "ar-1").Return(&artist, nil).Maybe()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{artist}, nil).Maybe()

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "ar-1", "Thin Artist", "", 5).
					Return([]agents.Song{}, nil).Once()

				// No similar artist is in the library, so the fallback yields only the seed artist's
				// own matching top song: one track for a mix of five.
				mockAgent.On("GetSimilarArtists", mock.Anything, "ar-1", "Thin Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "ar-1", "Thin Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Top Song", MBID: "mbid-top"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{topSong}, nil).Once()

				mediaFileRepo.On("GetRandom", mock.Anything).
					Return(model.MediaFiles{{ID: "s1", Title: "Seed", Artist: "Thin Artist"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "Thin Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Mix Song", MBID: "mbid-mix"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).
					Return(model.MediaFiles{{ID: "mix-1", Title: "Mix Song", MbzRecordingID: "mbid-mix"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "ar-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(songs, func(mf model.MediaFile) string { return mf.ID })).
					To(ConsistOf("top-1", "mix-1"))
			})
		})

		Context("when ID is an Artist and the agent plus the similar-artists fallback already fill the mix", func() {
			It("does not pay for seed-track sampling", func() {
				artist := model.Artist{ID: "ar-1", Name: "The Artist"}
				artistRepo.On("Get", "ar-1").Return(&artist, nil).Maybe()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{artist}, nil).Maybe()

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "ar-1", "The Artist", "", 3).
					Return([]agents.Song{{Name: "A", MBID: "mbid-a"}, {Name: "B", MBID: "mbid-b"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{
					{ID: "t-a", Title: "A", MbzRecordingID: "mbid-a"},
					{ID: "t-b", Title: "B", MbzRecordingID: "mbid-b"},
				}, nil).Once()

				// The similar-artists fallback supplies the third track, so the mix is full.
				mockAgent.On("GetSimilarArtists", mock.Anything, "ar-1", "The Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "ar-1", "The Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "C", MBID: "mbid-c"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).
					Return(model.MediaFiles{{ID: "t-c", Title: "C", MbzRecordingID: "mbid-c"}}, nil).Once()

				mediaFileRepo.On("GetRandom", mock.Anything).Return(model.MediaFiles{{ID: "seed"}}, nil).Maybe()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "ar-1", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(songs, func(mf model.MediaFile) string { return mf.ID })).
					To(ConsistOf("t-a", "t-b", "t-c"))
				mediaFileRepo.AssertNotCalled(GinkgoT(), "GetRandom", mock.Anything)
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
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: seedTrack},
				})

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

			It("asks for a smart-playlist refresh so an unevaluated one still yields seeds", func() {
				// A smart playlist materializes no playlist_tracks until it is evaluated, so sampling
				// without the refresh would mix an empty seed set.
				pls := model.Playlist{ID: "pl-smart", Name: "Smart"}
				artistRepo.On("Get", "pl-smart").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-smart").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "s1", Title: "Seed One"}},
				})
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "pl-smart", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).ToNot(BeEmpty())
				Expect(playlistRepo.TracksRefreshed).To(BeTrue())
			})

			It("returns an error instead of panicking when the track repository is unavailable", func() {
				// Tracks() logs and returns a nil repository when its own lookup fails.
				pls := model.Playlist{ID: "pl-nil", Name: "Gone"}
				artistRepo.On("Get", "pl-nil").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-nil").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistRepo.TracksRepo = nil

				_, err := provider.SimilarSongs(ctx, "pl-nil", 5)

				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("does not seed a mix with missing tracks", func() {
				pls := model.Playlist{ID: "pl-missing", Name: "Missing"}
				artistRepo.On("Get", "pl-missing").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-missing").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "s1", Title: "Seed One"}},
				})
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{}, nil).Maybe()

				_, err := provider.SimilarSongs(ctx, "pl-missing", 5)

				Expect(err).ToNot(HaveOccurred())
				sql, args, sqlErr := playlistTrackRepo.Options.Filters.ToSql()
				Expect(sqlErr).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("missing"))
				Expect(args).To(ContainElement(false), "must exclude missing files, not select them")
			})

			It("keeps a later seed's picks when an earlier seed overlaps it", func() {
				// The matcher stops once it has count matches, and it re-emits the shared track, so
				// matching only count of the merged set would spend slots on the duplicate.
				pls := model.Playlist{ID: "pl-overlap2", Name: "Overlap2"}
				artistRepo.On("Get", "pl-overlap2").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-overlap2").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "s1", Title: "Seed One"}},
					{MediaFile: model.MediaFile{ID: "s2", Title: "Seed Two"}},
				})
				shared := agents.Song{ID: "x", Name: "X"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 3).
					Return([]agents.Song{shared}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "", "", 3).
					Return([]agents.Song{shared, {ID: "y", Name: "Y"}, {ID: "z", Name: "Z"}}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{
					{ID: "x", Title: "X"}, {ID: "y", Title: "Y"}, {ID: "z", Title: "Z"},
				}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "pl-overlap2", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(3), "the duplicate must not cost a slot")
			})

			It("returns a track once when two seeds recommend it", func() {
				// The matcher re-emits a track when two inputs are identical, so overlapping
				// recommendations would otherwise take two slots in the mix.
				pls := model.Playlist{ID: "pl-overlap", Name: "Overlap"}
				artistRepo.On("Get", "pl-overlap").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-overlap").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "s1", Title: "Seed One"}},
					{MediaFile: model.MediaFile{ID: "s2", Title: "Seed Two"}},
				})
				shared := agents.Song{ID: "m1", Name: "Shared"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 5).
					Return([]agents.Song{shared}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "", "", 5).
					Return([]agents.Song{shared}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{{ID: "m1", Title: "Shared"}}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "pl-overlap", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1), "the shared recommendation must appear once")
			})

			It("does not seed a mix twice with a track the playlist repeats", func() {
				pls := model.Playlist{ID: "pl-dup", Name: "Dupes"}
				artistRepo.On("Get", "pl-dup").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-dup").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				// The same file at two positions, which playlists allow.
				dup := model.MediaFile{ID: "s1", Title: "Seed One"}
				playlistTrackRepo.SetData(model.PlaylistTracks{{MediaFile: dup}, {MediaFile: dup}})
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 5).
					Return([]agents.Song{}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-dup", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1), "the repeated track must appear once")
				agentsCombined.AssertNumberOfCalls(GinkgoT(), "GetSimilarSongsByTrack", 1)
			})

			It("clamps an enormous count before it reaches the queries", func() {
				// count+1 in the local agent overflows on MaxInt64, and GetRandom omits the SQL
				// limit unless Max is positive, so the query would hydrate the whole library.
				pls := model.Playlist{ID: "pl-huge", Name: "Huge"}
				artistRepo.On("Get", "pl-huge").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-huge").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "s1", Title: "Seed One"}},
				})
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 500).
					Return([]agents.Song{}, nil).Once()

				_, err := provider.SimilarSongs(ctx, "pl-huge", math.MaxInt64)

				Expect(err).ToNot(HaveOccurred())
				agentsCombined.AssertExpectations(GinkgoT())
			})

			It("does not panic when the caller asks for a non-positive count", func() {
				// Subsonic passes count straight through, so a negative one reaches the provider.
				// The guard returns before any lookup, so no repository setup is needed.
				songs, err := provider.SimilarSongs(ctx, "pl-neg", -1)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(BeEmpty())
			})

			It("blends results from every seed, not just the first", func() {
				pls := model.Playlist{ID: "pl-blend", Name: "Blend"}
				artistRepo.On("Get", "pl-blend").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-blend").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: model.MediaFile{ID: "s1", Title: "Seed One"}},
					{MediaFile: model.MediaFile{ID: "s2", Title: "Seed Two"}},
				})

				// Each seed returns a full count's worth, as a real similarity agent does. Asking for
				// one more than seed one can supply makes this independent of the final shuffle:
				// three of the four matches always include a seed-two track.
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 3).
					Return([]agents.Song{{ID: "a1", Name: "A1"}, {ID: "a2", Name: "A2"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "", "", 3).
					Return([]agents.Song{{ID: "b1", Name: "B1"}, {ID: "b2", Name: "B2"}}, nil).Once()

				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{
					{ID: "a1", Title: "A1"}, {ID: "a2", Title: "A2"},
					{ID: "b1", Title: "B1"}, {ID: "b2", Title: "B2"},
				}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "pl-blend", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(3))
				ids := slice.Map(songs, func(mf model.MediaFile) string { return mf.ID })
				Expect(ids).To(ContainElement(BeElementOf("b1", "b2")), "seed two must be represented in the mix")
			})

			It("falls back to the seed tracks themselves when no similar songs are found", func() {
				pls := model.Playlist{ID: "pl-2", Name: "Fallback List"}
				seed1 := model.MediaFile{ID: "s1", Title: "Seed One", Artist: "A"}
				seed2 := model.MediaFile{ID: "s2", Title: "Seed Two", Artist: "B"}

				artistRepo.On("Get", "pl-2").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-2").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})

				playlistTrackRepo.SetData(model.PlaylistTracks{
					{MediaFile: seed1},
					{MediaFile: seed2},
				})

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

			It("samples the album when the agent's picks are not in this library", func() {
				// Last.fm answers from its own catalogue, so a small library can match none of it.
				// An unmatched non-empty answer must not shortcut the sampling fallback.
				album := model.Album{ID: "al-nm", Name: "NoMatch", AlbumArtist: "A"}
				artistRepo.On("Get", "al-nm").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "al-nm").Return(&album, nil).Once()
				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "al-nm", "NoMatch", "A", "", 5).
					Return([]agents.Song{{Name: "Not In Library"}}, nil).Once()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{}, nil).Maybe()

				// The matcher resolves nothing; the sampled seed is what reaches the mix.
				mediaFileRepo.On("GetRandom", mock.Anything).
					Return(model.MediaFiles{{ID: "s1", Title: "Album Track"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Album Track", "", "", 5).
					Return([]agents.Song{}, nil).Once()
				mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "al-nm", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(1))
				Expect(songs[0].ID).To(Equal("s1"))
			})

			It("caps agent calls at maxSeeds when the repository ignores the bound", func() {
				// Isolates seedMix's own cap: the album sampler bounds the query with Max, so this
				// exercises the guard by having the repo hand back more rows than were asked for.
				album := model.Album{ID: "al-cap", Name: "Cap", AlbumArtist: "A"}
				artistRepo.On("Get", "al-cap").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "al-cap").Return(&album, nil).Once()
				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "al-cap", "Cap", "A", "", 5).
					Return([]agents.Song{}, nil).Once()

				var overflow model.MediaFiles
				for _, id := range []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8"} {
					overflow = append(overflow, model.MediaFile{ID: id, Title: id})
				}
				mediaFileRepo.On("GetRandom", mock.Anything).Return(overflow, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, 5).
					Return([]agents.Song{}, nil)

				_, err := provider.SimilarSongs(ctx, "al-cap", 5)

				Expect(err).ToNot(HaveOccurred())
				agentsCombined.AssertNumberOfCalls(GinkgoT(), "GetSimilarSongsByTrack", 5)
			})

			It("still finds maxSeeds distinct seeds when the leading positions repeat", func() {
				// The sampler over-fetches for exactly this case: a page bounded at maxSeeds could
				// be entirely one repeated file and collapse to a single seed.
				pls := model.Playlist{ID: "pl-3", Name: "Big List"}
				dup := model.MediaFile{ID: "dup", Title: "Dup", Artist: "A"}
				tracks := model.PlaylistTracks{
					{MediaFile: dup}, {MediaFile: dup}, {MediaFile: dup}, {MediaFile: dup}, {MediaFile: dup},
				}
				for _, id := range []string{"seed-1", "seed-2", "seed-3", "seed-4", "seed-5"} {
					tracks = append(tracks, model.PlaylistTrack{
						MediaFile: model.MediaFile{ID: id, Title: id, Artist: "A"},
					})
				}

				artistRepo.On("Get", "pl-3").Return(nil, model.ErrNotFound).Once()
				albumRepo.On("Get", "pl-3").Return(nil, model.ErrNotFound).Once()
				playlistRepo.SetData(model.Playlists{pls})
				playlistTrackRepo.SetData(tracks)

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, 5).
					Return([]agents.Song{}, nil)

				songs, err := provider.SimilarSongs(ctx, "pl-3", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(5))
				Expect(playlistTrackRepo.Options.Sort).To(Equal("random"))
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
					sql, args, err := opt.Filters.ToSql()
					return err == nil && strings.Contains(sql, "media_file_tags") &&
						!strings.Contains(sql, "json_tree") && strings.Contains(sql, "missing") && slices.Contains(args, any(false))
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

	It("keeps picking until the fallback holds count distinct tracks", func() {
		// A collaboration in two artists' top songs lands in the chooser twice. Picking a fixed
		// count of entries lets those duplicates eat the budget and strand unique candidates.
		track := model.MediaFile{ID: "track-1", Title: "Track", Artist: "Artist One", ArtistID: "artist-1"}
		artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
		similarArtist := model.Artist{ID: "artist-3", Name: "Similar Artist"}

		artistRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Twice()
		albumRepo.On("Get", "track-1").Return(nil, model.ErrNotFound).Twice()
		mediaFileRepo.On("Get", "track-1").Return(&track, nil).Twice()
		artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
		artistRepo.On("Get", "artist-3").Return(&similarArtist, nil).Maybe()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			return opt.Max == 1 && opt.Filters != nil
		})).Return(model.Artists{artist1}, nil).Maybe()

		agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist One", "", 3).
			Return([]agents.Song{}, nil).Once()

		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{{Name: "Similar Artist"}}, nil).Once()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			_, ok := opt.Filters.(squirrel.Eq)
			return opt.Max == 0 && ok
		})).Return(model.Artists{}, nil).Once()
		artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			_, ok := opt.Filters.(squirrel.Or)
			return opt.Max == 0 && ok
		})).Return(model.Artists{similarArtist}, nil).Once()

		shared := model.MediaFile{ID: "t-a", Title: "Shared", MbzRecordingID: "mbid-a"}
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{{Name: "Shared", MBID: "mbid-a"}}, nil).Once()
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).
			Return(model.MediaFiles{shared}, nil).Once()

		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-3", "Similar Artist", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Shared", MBID: "mbid-a"},
				{Name: "B", MBID: "mbid-b"},
				{Name: "C", MBID: "mbid-c"},
			}, nil).Once()
		mediaFileRepo.On("GetAll", mock.AnythingOfType("model.QueryOptions")).Return(model.MediaFiles{
			shared,
			{ID: "t-b", Title: "B", MbzRecordingID: "mbid-b"},
			{ID: "t-c", Title: "C", MbzRecordingID: "mbid-c"},
		}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "track-1", 3)

		Expect(err).ToNot(HaveOccurred())
		Expect(slice.Map(songs, func(mf model.MediaFile) string { return mf.ID })).
			To(ConsistOf("t-a", "t-b", "t-c"))
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

		songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

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
			sql, args, err := opt.Filters.ToSql()
			return err == nil && strings.Contains(sql, "artist_id") &&
				strings.Contains(sql, "missing") && slices.Contains(args, any(false)) && slices.Contains(args, any("artist-1"))
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
