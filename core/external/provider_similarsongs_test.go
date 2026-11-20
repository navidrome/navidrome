package external_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/core/agents"
	. "github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// nilTracksRepo is a playlist repository whose Tracks lookup fails, as the real one does when it
// cannot load the playlist.
type nilTracksRepo struct{ model.PlaylistRepository }

func (nilTracksRepo) Tracks(context.Context, string, bool) model.PlaylistTrackRepository { return nil }

// unboundedRandomRepo is a media file repository whose GetRandom ignores Max.
type unboundedRandomRepo struct{ model.MediaFileRepository }

func (r unboundedRandomRepo) GetRandom(ctx context.Context, options ...model.QueryOptions) (model.MediaFiles, error) {
	for i := range options {
		options[i].Max = 0
	}
	return r.MediaFileRepository.GetRandom(ctx, options...)
}

var _ = Describe("Provider - SimilarSongs", func() {
	var ds model.DataStore
	var provider Provider
	var mockAgent *mockSimilarArtistAgent
	var agentsCombined *mockAgents
	var ctx context.Context

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), adminUser)
		ds = persistence.New(db.Db())
		admin := adminUser
		Expect(ds.User().Put(ctx, &admin)).To(Succeed())

		mockAgent = &mockSimilarArtistAgent{}
		agentsCombined = &mockAgents{
			topSongsAgent: mockAgent,
			similarAgent:  mockAgent,
		}

		provider = NewProvider(ds, agentsCombined, matcher.New(ds), &fakeBroker{})
	})

	useStore := func(store model.DataStore) {
		provider = NewProvider(store, agentsCombined, matcher.New(store), &fakeBroker{})
	}

	putPlaylist := func(pls model.Playlist, trackIDs ...string) {
		GinkgoHelper()
		pls.OwnerID = adminUser.ID
		Expect(ds.Playlist().Put(ctx, &pls)).To(Succeed())
		if len(trackIDs) > 0 {
			_, err := ds.Playlist().Tracks(ctx, pls.ID, false).Add(ctx, trackIDs)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	anySimilarByTrack := func() *mock.Call {
		return agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}

	putTrackEntity := func() {
		GinkgoHelper()
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "artist-1", Name: "Artist"})
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "track-1", Title: "Track", Artist: "Artist", ArtistID: "artist-1"})
	}

	Describe("dispatch by entity type", func() {
		Context("when ID is a MediaFile (track)", func() {
			It("calls GetSimilarSongsByTrack and returns matched songs", func() {
				dmArtist := model.Artist{ID: "dm-1", Name: "Depeche Mode", OrderArtistName: "depeche mode", MbzArtistID: "artist-mbid"}
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, dmArtist)
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "track-1", Title: "Just Can't Get Enough", Artist: "Depeche Mode", MbzRecordingID: "track-mbid"},
					model.MediaFile{ID: "matched-1", Title: "Dreaming of Me", Artist: "Depeche Mode", Participants: credited(dmArtist)},
				)

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Just Can't Get Enough", "Depeche Mode", "track-mbid", 1).
					Return([]agents.Song{
						{Name: "Dreaming of Me", MBID: "", Artists: []agents.Artist{{Name: "Depeche Mode", MBID: "artist-mbid"}}},
					}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 1)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"matched-1"}))
			})

			It("falls back to artist-based algorithm when GetSimilarSongsByTrack returns empty", func() {
				putTrackEntity()
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"})

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", mock.Anything).
					Return([]agents.Song{}, nil).Once()
				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song One", MBID: "mbid-1"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"song-1"}))
			})

			It("tops the mix up with the fallback when the agent's picks alone are too few", func() {
				putTrackEntity()
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "agent-1", Title: "Agent Pick", MbzRecordingID: "mbid-agent"},
					model.MediaFile{ID: "song-1", Title: "Song One", MbzRecordingID: "mbid-1"},
				)

				// The agent knows one track of the three asked for.
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", 3).
					Return([]agents.Song{{Name: "Agent Pick", MBID: "mbid-agent"}}, nil).Once()
				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song One", MBID: "mbid-1"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("agent-1", "song-1"))
			})

			It("reaches a unique pick that sits past the count-th repeat", func() {
				putTrackEntity()
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "t-a", Title: "Song A", MbzRecordingID: "mbid-a"},
					model.MediaFile{ID: "t-b", Title: "Song B", MbzRecordingID: "mbid-b"},
				)
				mockAgent.On("GetSimilarArtists", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Artist{}, nil).Maybe()
				mockAgent.On("GetArtistTopSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{}, nil).Maybe()

				// "Song B" sits past the second repeat, so a matcher capped at count never reaches it.
				repeated := agents.Song{Name: "Song A", MBID: "mbid-a"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", 2).
					Return([]agents.Song{repeated, repeated, {Name: "Song B", MBID: "mbid-b"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 2)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("t-a", "t-b"))
			})

			It("keeps topping up when the agent's picks repeat a track", func() {
				putTrackEntity()
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "t-a", Title: "Song A", MbzRecordingID: "mbid-a"},
					model.MediaFile{ID: "t-b", Title: "Song B", MbzRecordingID: "mbid-b"},
					model.MediaFile{ID: "t-c", Title: "Song C", MbzRecordingID: "mbid-c"},
				)

				// The matcher re-emits a track when the same input song repeats, so these three
				// picks resolve to only two distinct library tracks.
				repeated := agents.Song{Name: "Song A", MBID: "mbid-a"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist", "", 3).
					Return([]agents.Song{repeated, repeated, {Name: "Song B", MBID: "mbid-b"}}, nil).Once()
				mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Song C", MBID: "mbid-c"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "track-1", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("t-a", "t-b", "t-c"))
			})
		})

		Context("when ID is an Album", func() {
			It("calls GetSimilarSongsByAlbum and returns matched songs", func() {
				persistencetest.PutAlbums(ctx, ds, model.Album{ID: "album-1", Name: "Speak & Spell", AlbumArtist: "Depeche Mode", MbzAlbumID: "album-mbid"})
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "matched-1", Title: "New Life", Artist: "Depeche Mode", MbzRecordingID: "song-mbid"})

				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "album-1", "Speak & Spell", "Depeche Mode", "album-mbid", 1).
					Return([]agents.Song{
						{Name: "New Life", MBID: "song-mbid", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
					}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "album-1", 1)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"matched-1"}))
			})

			It("falls back to sampled album tracks when GetSimilarSongsByAlbum returns ErrNotFound", func() {
				persistencetest.PutAlbums(ctx, ds,
					model.Album{ID: "album-1", Name: "Album", AlbumArtist: "Artist", AlbumArtistID: "artist-1"},
					model.Album{ID: "album-2", Name: "Other"},
				)
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "seed-1", Title: "Seed", Artist: "Artist", AlbumID: "album-1"},
					model.MediaFile{ID: "gone", Title: "Gone", AlbumID: "album-1", Missing: true},
					model.MediaFile{ID: "other", Title: "Other", AlbumID: "album-2"},
				)

				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "album-1", "Album", "Artist", "", mock.Anything).
					Return(nil, agents.ErrNotFound).Once()
				// seedMix falls back to the seed itself when the agent finds nothing.
				anySimilarByTrack().Return([]agents.Song{}, nil)

				songs, err := provider.SimilarSongs(ctx, "album-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"seed-1"}))
			})
		})

		Context("when ID is an Album and the album agent returns nothing (AudioMuse-only)", func() {
			It("samples the album's tracks and returns their track-similars", func() {
				aArtist := model.Artist{ID: "a1", Name: "A", OrderArtistName: "a"}
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, aArtist)
				persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al-1", Name: "The Album", AlbumArtist: "A"})
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed", Artist: "A", AlbumID: "al-1"},
					model.MediaFile{ID: "m1", Title: "AudioMuseResult", Artist: "A", Participants: credited(aArtist)},
				)

				// AudioMuse doesn't implement album similarity -> empty.
				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "al-1", "The Album", "A", "", 5).
					Return([]agents.Song{}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "A", "", 5).
					Return([]agents.Song{{Name: "AudioMuseResult", Artists: []agents.Artist{{Name: "A"}}}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "al-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"m1"}))
			})
		})

		Context("when ID is an Artist", func() {
			It("calls GetSimilarSongsByArtist and returns matched songs", func() {
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "artist-1", Name: "Depeche Mode", MbzArtistID: "artist-mbid"})
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "matched-1", Title: "Enjoy the Silence", Artist: "Depeche Mode", MbzRecordingID: "song-mbid"})

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Depeche Mode", "artist-mbid", 1).
					Return([]agents.Song{
						{Name: "Enjoy the Silence", MBID: "song-mbid", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
					}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"matched-1"}))
			})
		})

		Context("when ID is an Artist and both the artist agent and the similar-artists fallback are empty", func() {
			It("samples the artist's tracks and returns their track-similars", func() {
				artist := model.Artist{ID: "ar-1", Name: "The Artist"}
				other := model.Artist{ID: "ar-2", Name: "Someone Else"}
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist, other)
				albumCredit := model.Participants{}
				albumCredit.Add(model.RoleAlbumArtist, artist)
				// An artist credited only on the album (compilations, classical) still yields seeds.
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed", Participants: albumCredit},
					model.MediaFile{ID: "s2", Title: "Seed Two", Participants: credited(artist)},
					model.MediaFile{ID: "gone", Title: "Gone", Participants: credited(artist), Missing: true},
					model.MediaFile{ID: "theirs", Title: "Theirs", Participants: credited(other)},
				)

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "ar-1", "The Artist", "", 5).
					Return([]agents.Song{}, nil).Once()
				mockAgent.On("GetSimilarArtists", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Artist{}, nil).Maybe()
				mockAgent.On("GetArtistTopSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]agents.Song{}, nil).Maybe()
				anySimilarByTrack().Return([]agents.Song{{Name: "Result"}}, nil)

				songs, err := provider.SimilarSongs(ctx, "ar-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("s1", "s2"))
			})
		})

		Context("when ID is an Artist and the similar-artists fallback can't fill the mix", func() {
			It("tops the mix up with the artist's own track-similars", func() {
				artist := model.Artist{ID: "ar-1", Name: "Thin Artist"}
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist)
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "top-1", Title: "Top Song", ArtistID: "ar-1", MbzRecordingID: "mbid-top"},
					model.MediaFile{ID: "s1", Title: "Seed", Artist: "Thin Artist", Participants: credited(artist)},
					model.MediaFile{ID: "mix-1", Title: "Mix Song", MbzRecordingID: "mbid-mix"},
				)

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "ar-1", "Thin Artist", "", 5).
					Return([]agents.Song{}, nil).Once()

				// No similar artist is in the library, so the fallback yields only the seed artist's
				// own matching top song: one track for a mix of five.
				mockAgent.On("GetSimilarArtists", mock.Anything, "ar-1", "Thin Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "ar-1", "Thin Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Top Song", MBID: "mbid-top"}}, nil).Once()

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed", "Thin Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "Mix Song", MBID: "mbid-mix"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "ar-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("top-1", "mix-1"))
			})
		})

		Context("when ID is an Artist and the agent plus the similar-artists fallback already fill the mix", func() {
			It("does not pay for seed-track sampling", func() {
				artist := model.Artist{ID: "ar-1", Name: "The Artist"}
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist)
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "t-a", Title: "A", MbzRecordingID: "mbid-a", Participants: credited(artist)},
					model.MediaFile{ID: "t-b", Title: "B", MbzRecordingID: "mbid-b", Participants: credited(artist)},
					model.MediaFile{ID: "t-c", Title: "C", MbzRecordingID: "mbid-c", Participants: credited(artist)},
				)

				agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "ar-1", "The Artist", "", 3).
					Return([]agents.Song{{Name: "A", MBID: "mbid-a"}, {Name: "B", MBID: "mbid-b"}}, nil).Once()

				// The similar-artists fallback supplies the third track, so the mix is full.
				mockAgent.On("GetSimilarArtists", mock.Anything, "ar-1", "The Artist", "", 15).
					Return([]agents.Artist{}, nil).Once()
				mockAgent.On("GetArtistTopSongs", mock.Anything, "ar-1", "The Artist", "", mock.Anything).
					Return([]agents.Song{{Name: "C", MBID: "mbid-c"}}, nil).Once()

				anySimilarByTrack().Return([]agents.Song{}, nil).Maybe()

				songs, err := provider.SimilarSongs(ctx, "ar-1", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("t-a", "t-b", "t-c"))
				agentsCombined.AssertNotCalled(GinkgoT(), "GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		})

		Context("when ID is a Playlist", func() {
			It("samples playlist tracks and returns their track-similars", func() {
				aArtist := model.Artist{ID: "a1", Name: "A", OrderArtistName: "a"}
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, aArtist)
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed One", Artist: "A"},
					model.MediaFile{ID: "m1", Title: "Similar", Artist: "A", Participants: credited(aArtist)},
				)
				putPlaylist(model.Playlist{ID: "pl-1", Name: "My List"}, "s1")

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "A", "", 5).
					Return([]agents.Song{{Name: "Similar", Artists: []agents.Artist{{Name: "A"}}}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-1", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"m1"}))
			})

			It("asks for a smart-playlist refresh so an unevaluated one still yields seeds", func() {
				// A smart playlist materializes no playlist_tracks until it is evaluated, so sampling
				// without the refresh would mix an empty seed set.
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "s1", Title: "Seed One"})
				putPlaylist(model.Playlist{ID: "pl-smart", Name: "Smart", Rules: &criteria.Criteria{
					Expression: criteria.All{criteria.Contains{"title": "seed"}},
				}})
				anySimilarByTrack().Return([]agents.Song{}, nil)

				songs, err := provider.SimilarSongs(ctx, "pl-smart", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"s1"}))
			})

			It("returns an error instead of panicking when the track repository is unavailable", func() {
				putPlaylist(model.Playlist{ID: "pl-nil", Name: "Gone"})
				useStore(persistencetest.Stub(ds, persistencetest.Repos{Playlist: func(r model.PlaylistRepository) model.PlaylistRepository {
					return nilTracksRepo{r}
				}}))

				_, err := provider.SimilarSongs(ctx, "pl-nil", 5)

				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("does not seed a mix with missing tracks", func() {
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed One"},
					model.MediaFile{ID: "gone", Title: "Gone", Missing: true},
				)
				putPlaylist(model.Playlist{ID: "pl-missing", Name: "Missing"}, "s1", "gone")
				anySimilarByTrack().Return([]agents.Song{}, nil)

				songs, err := provider.SimilarSongs(ctx, "pl-missing", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"s1"}), "must exclude missing files, not select them")
			})

			It("keeps a later seed's picks when an earlier seed overlaps it", func() {
				// The matcher stops once it has count matches, and it re-emits the shared track, so
				// matching only count of the merged set would spend slots on the duplicate.
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed One"},
					model.MediaFile{ID: "s2", Title: "Seed Two"},
					model.MediaFile{ID: "x", Title: "X"},
					model.MediaFile{ID: "y", Title: "Y"},
					model.MediaFile{ID: "z", Title: "Z"},
				)
				putPlaylist(model.Playlist{ID: "pl-overlap2", Name: "Overlap2"}, "s1", "s2")
				shared := agents.Song{ID: "x", Name: "X"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 3).
					Return([]agents.Song{shared}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "", "", 3).
					Return([]agents.Song{shared, {ID: "y", Name: "Y"}, {ID: "z", Name: "Z"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-overlap2", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("x", "y", "z"), "the duplicate must not cost a slot")
			})

			It("returns a track once when two seeds recommend it", func() {
				// The matcher re-emits a track when two inputs are identical, so overlapping
				// recommendations would otherwise take two slots in the mix.
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed One"},
					model.MediaFile{ID: "s2", Title: "Seed Two"},
					model.MediaFile{ID: "m1", Title: "Shared"},
				)
				putPlaylist(model.Playlist{ID: "pl-overlap", Name: "Overlap"}, "s1", "s2")
				shared := agents.Song{ID: "m1", Name: "Shared"}
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 5).
					Return([]agents.Song{shared}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "", "", 5).
					Return([]agents.Song{shared}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-overlap", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"m1"}), "the shared recommendation must appear once")
			})

			It("does not seed a mix twice with a track the playlist repeats", func() {
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "s1", Title: "Seed One"})
				putPlaylist(model.Playlist{ID: "pl-dup", Name: "Dupes"}, "s1", "s1")
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 5).
					Return([]agents.Song{}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-dup", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"s1"}), "the repeated track must appear once")
				agentsCombined.AssertNumberOfCalls(GinkgoT(), "GetSimilarSongsByTrack", 1)
			})

			It("clamps an enormous count before it reaches the queries", func() {
				// count+1 in the local agent overflows on MaxInt64, and GetRandom omits the SQL
				// limit unless Max is positive, so the query would hydrate the whole library.
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "s1", Title: "Seed One"})
				putPlaylist(model.Playlist{ID: "pl-huge", Name: "Huge"}, "s1")
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 500).
					Return([]agents.Song{}, nil).Once()

				_, err := provider.SimilarSongs(ctx, "pl-huge", math.MaxInt64)

				Expect(err).ToNot(HaveOccurred())
				agentsCombined.AssertExpectations(GinkgoT())
			})

			It("does not panic when the caller asks for a non-positive count", func() {
				// Subsonic passes count straight through, so a negative one reaches the provider.
				songs, err := provider.SimilarSongs(ctx, "pl-neg", -1)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(BeEmpty())
			})

			It("blends results from every seed, not just the first", func() {
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed One"},
					model.MediaFile{ID: "s2", Title: "Seed Two"},
					model.MediaFile{ID: "a1", Title: "A1"},
					model.MediaFile{ID: "a2", Title: "A2"},
					model.MediaFile{ID: "b1", Title: "B1"},
					model.MediaFile{ID: "b2", Title: "B2"},
				)
				putPlaylist(model.Playlist{ID: "pl-blend", Name: "Blend"}, "s1", "s2")

				// Each seed returns a full count's worth, as a real similarity agent does. Asking for
				// one more than seed one can supply makes this independent of the final shuffle:
				// three of the four matches always include a seed-two track.
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "", "", 3).
					Return([]agents.Song{{ID: "a1", Name: "A1"}, {ID: "a2", Name: "A2"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "", "", 3).
					Return([]agents.Song{{ID: "b1", Name: "B1"}, {ID: "b2", Name: "B2"}}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-blend", 3)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(3))
				Expect(ids(songs)).To(ContainElement(BeElementOf("b1", "b2")), "seed two must be represented in the mix")
			})

			It("falls back to the seed tracks themselves when no similar songs are found", func() {
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed One", Artist: "A"},
					model.MediaFile{ID: "s2", Title: "Seed Two", Artist: "B"},
				)
				putPlaylist(model.Playlist{ID: "pl-2", Name: "Fallback List"}, "s1", "s2")

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Seed One", "A", "", 5).
					Return([]agents.Song{}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s2", "Seed Two", "B", "", 5).
					Return([]agents.Song{}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "pl-2", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(ConsistOf("s1", "s2"))
			})

			It("samples the album when the agent's picks are not in this library", func() {
				// Last.fm answers from its own catalogue, so a small library can match none of it.
				// An unmatched non-empty answer must not shortcut the sampling fallback.
				persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al-nm", Name: "NoMatch", AlbumArtist: "A"})
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "s1", Title: "Album Track", AlbumID: "al-nm"})
				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "al-nm", "NoMatch", "A", "", 5).
					Return([]agents.Song{{Name: "Not In Library"}}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "s1", "Album Track", "", "", 5).
					Return([]agents.Song{}, nil).Once()

				songs, err := provider.SimilarSongs(ctx, "al-nm", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"s1"}))
			})

			It("caps agent calls at maxSeeds when the repository ignores the bound", func() {
				// Isolates seedMix's own cap: the album sampler bounds the query with Max, so this
				// exercises the guard by having the repo hand back more rows than were asked for.
				persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al-cap", Name: "Cap", AlbumArtist: "A"})
				for _, id := range []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8"} {
					persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: id, Title: id, AlbumID: "al-cap"})
				}
				useStore(persistencetest.Stub(ds, persistencetest.Repos{MediaFile: func(r model.MediaFileRepository) model.MediaFileRepository {
					return unboundedRandomRepo{r}
				}}))
				agentsCombined.On("GetSimilarSongsByAlbum", mock.Anything, "al-cap", "Cap", "A", "", 5).
					Return([]agents.Song{}, nil).Once()
				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, 5).
					Return([]agents.Song{}, nil)

				_, err := provider.SimilarSongs(ctx, "al-cap", 5)

				Expect(err).ToNot(HaveOccurred())
				agentsCombined.AssertNumberOfCalls(GinkgoT(), "GetSimilarSongsByTrack", 5)
			})

			It("still finds maxSeeds distinct seeds when the leading positions repeat", func() {
				// The sampler over-fetches for exactly this case: a page bounded at maxSeeds could
				// be entirely one repeated file and collapse to a single seed.
				trackIDs := []string{"dup", "dup", "dup", "dup", "dup"}
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "dup", Title: "Dup", Artist: "A"})
				for _, id := range []string{"seed-1", "seed-2", "seed-3", "seed-4", "seed-5"} {
					persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: id, Title: id, Artist: "A"})
					trackIDs = append(trackIDs, id)
				}
				putPlaylist(model.Playlist{ID: "pl-3", Name: "Big List"}, trackIDs...)

				agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, 5).
					Return([]agents.Song{}, nil)

				songs, err := provider.SimilarSongs(ctx, "pl-3", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(songs).To(HaveLen(5))
				agentsCombined.AssertNumberOfCalls(GinkgoT(), "GetSimilarSongsByTrack", 5)
			})

			It("samples a different set of seeds across calls", func() {
				var trackIDs []string
				for _, id := range []string{"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9"} {
					persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: id, Title: id})
					trackIDs = append(trackIDs, id)
				}
				putPlaylist(model.Playlist{ID: "pl-random", Name: "Random"}, trackIDs...)
				anySimilarByTrack().Return([]agents.Song{}, nil)

				seen := map[string]struct{}{}
				for range 10 {
					songs, err := provider.SimilarSongs(ctx, "pl-random", 5)
					Expect(err).ToNot(HaveOccurred())
					seeds := ids(songs)
					slices.Sort(seeds)
					seen[strings.Join(seeds, ",")] = struct{}{}
				}

				Expect(len(seen)).To(BeNumerically(">", 1))
			})
		})

		Context("when ID is a Genre (not resolved by GetEntityByID)", func() {
			It("samples genre songs and returns their track-similars", func() {
				jazz := model.NewTag(model.TagGenre, "Jazz")
				rock := model.NewTag(model.TagGenre, "Rock")
				Expect(ds.Tag().Add(ctx, 1, jazz, rock)).To(Succeed())
				persistencetest.PutMediaFiles(ctx, ds,
					model.MediaFile{ID: "s1", Title: "Seed", Tags: model.Tags{model.TagGenre: {"Jazz"}}},
					model.MediaFile{ID: "gone", Title: "Gone", Tags: model.Tags{model.TagGenre: {"Jazz"}}, Missing: true},
					model.MediaFile{ID: "rock", Title: "Rock", Tags: model.Tags{model.TagGenre: {"Rock"}}},
				)
				anySimilarByTrack().Return([]agents.Song{{Name: "Similar"}}, nil)

				songs, err := provider.SimilarSongs(ctx, jazz.ID, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(ids(songs)).To(Equal([]string{"s1"}))
			})
		})
	})

	It("returns similar songs from main artist and similar artists", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID,
			model.Artist{ID: "artist-1", Name: "Artist One"},
			model.Artist{ID: "artist-3", Name: "Similar Artist"},
		)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"},
			model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-2"},
			model.MediaFile{ID: "song-3", Title: "Song Three", ArtistID: "artist-3", MbzRecordingID: "mbid-3"},
		)

		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{{Name: "Similar Artist", MBID: "similar-mbid"}}, nil).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}, nil).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-3", "Similar Artist", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song Three", MBID: "mbid-3"},
			}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 3)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(ConsistOf("song-1", "song-2", "song-3"))
	})

	It("keeps picking until the fallback holds count distinct tracks", func() {
		// A collaboration in two artists' top songs lands in the chooser twice. Picking a fixed
		// count of entries lets those duplicates eat the budget and strand unique candidates.
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID,
			model.Artist{ID: "artist-1", Name: "Artist One"},
			model.Artist{ID: "artist-3", Name: "Similar Artist"},
		)
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "track-1", Title: "Track", Artist: "Artist One", ArtistID: "artist-1"},
			model.MediaFile{ID: "t-a", Title: "Shared", MbzRecordingID: "mbid-a"},
			model.MediaFile{ID: "t-b", Title: "B", MbzRecordingID: "mbid-b"},
			model.MediaFile{ID: "t-c", Title: "C", MbzRecordingID: "mbid-c"},
		)

		agentsCombined.On("GetSimilarSongsByTrack", mock.Anything, "track-1", "Track", "Artist One", "", 3).
			Return([]agents.Song{}, nil).Once()
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{{Name: "Similar Artist"}}, nil).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{{Name: "Shared", MBID: "mbid-a"}}, nil).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-3", "Similar Artist", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Shared", MBID: "mbid-a"},
				{Name: "B", MBID: "mbid-b"},
				{Name: "C", MBID: "mbid-c"},
			}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "track-1", 3)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(ConsistOf("t-a", "t-b", "t-c"))
	})

	It("returns ErrNotFound when artist is not found", func() {
		songs, err := provider.SimilarSongs(ctx, "artist-unknown-artist", 5)

		Expect(err).To(Equal(model.ErrNotFound))
		Expect(songs).To(BeNil())
	})

	It("returns songs from main artist when GetSimilarArtists returns error", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "artist-1", Name: "Artist One"})
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"})

		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return(nil, errors.New("error getting similar artists")).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
			}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(Equal([]string{"song-1"}))
	})

	It("returns empty list when GetArtistTopSongs returns error", func() {
		artist := model.Artist{ID: "artist-1", Name: "Artist One"}
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, artist)
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "gone", Title: "Gone", Participants: credited(artist), Missing: true})

		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{}, nil).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, errors.New("error getting top songs")).Once()
		// The fallback yields nothing, so the sampling path is tried and finds only a missing track.
		anySimilarByTrack().Return([]agents.Song{}, nil).Maybe()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 5)

		Expect(err).ToNot(HaveOccurred())
		Expect(songs).To(BeEmpty())
	})

	It("respects count parameter", func() {
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "artist-1", Name: "Artist One"})
		persistencetest.PutMediaFiles(ctx, ds,
			model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzRecordingID: "mbid-1"},
			model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzRecordingID: "mbid-2"},
		)

		agentsCombined.On("GetSimilarSongsByArtist", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return(nil, agents.ErrNotFound).Once()
		mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
			Return([]agents.Artist{}, nil).Once()
		mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
			Return([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}, nil).Once()

		songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

		Expect(err).ToNot(HaveOccurred())
		Expect(ids(songs)).To(ConsistOf(BeElementOf("song-1", "song-2")))
	})
})
