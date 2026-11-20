package matcher_test

import (
	"context"
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Matcher", func() {
	var ds model.DataStore
	var ctx context.Context
	var m *matcher.Matcher

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		user := model.User{ID: "user-1", UserName: "user-1"}
		ds = persistence.New(db.Db())
		Expect(ds.User().Put(ctx, &user)).To(Succeed())
		ctx = request.WithUser(ctx, user)
		m = matcher.New(ds)
	})

	Describe("MatchSongs", func() {
		Context("matching by direct ID", func() {
			It("matches songs with an ID field to MediaFiles by ID", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
				songs := []agents.Song{
					{ID: "track-1", Name: "Some Song", Artists: []agents.Artist{{Name: "Some Artist"}}},
				}
				idMatch := model.MediaFile{
					ID: "track-1", Title: "Some Song", Artist: "Some Artist",
				}
				putTracks(ctx, ds, model.MediaFiles{idMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-1"))
			})
		})

		Context("matching by MBID", func() {
			It("matches songs with MBID to tracks with matching mbz_recording_id", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
				songs := []agents.Song{
					{Name: "Paranoid Android", MBID: "abc-123", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				mbidMatch := model.MediaFile{
					ID: "track-mbid", Title: "Paranoid Android", Artist: "Radiohead",
					MbzRecordingID: "abc-123",
				}
				putTracks(ctx, ds, model.MediaFiles{mbidMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-mbid"))
			})
		})

		Context("matching by ISRC", func() {
			It("matches songs with ISRC to tracks with matching ISRC tag", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
				songs := []agents.Song{
					{Name: "Paranoid Android", ISRC: "GBAYE0000351", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				isrcMatch := model.MediaFile{
					ID: "track-isrc", Title: "Paranoid Android", Artist: "Radiohead",
					Tags: model.Tags{model.TagISRC: []string{"GBAYE0000351"}},
				}
				putTracks(ctx, ds, model.MediaFiles{isrcMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-isrc"))
			})
		})

		Context("fuzzy title+artist matching", func() {
			It("matches songs by title and artist name", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
				songs := []agents.Song{
					{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
				}
				titleMatch := model.MediaFile{
					ID: "track-title", Title: "Enjoy the Silence", Artist: "Depeche Mode",
					Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
				}
				putTracks(ctx, ds, model.MediaFiles{titleMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-title"))
			})

			It("matches songs with fuzzy title similarity", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}},
				}
				fuzzyMatch := model.MediaFile{
					ID: "track-fuzzy", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
					Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
				}
				putTracks(ctx, ds, model.MediaFiles{fuzzyMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-fuzzy"))
			})

			It("does not match completely different titles", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Yesterday", Artists: []agents.Artist{{Name: "The Beatles"}}},
				}
				differentTracks := model.MediaFiles{
					{ID: "different", Title: "Tomorrow Never Knows", Artist: "The Beatles",
						Participants: artistParticipants(model.Artist{ID: "beatles", Name: "The Beatles", OrderArtistName: "beatles"}),
					},
				}
				putTracks(ctx, ds, differentTracks)
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("deduplication", func() {
			It("removes duplicates when different input songs match the same library track", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody (Live)", Artists: []agents.Artist{{Name: "Queen"}}},
					{Name: "Bohemian Rhapsody (Original Mix)", Artists: []agents.Artist{{Name: "Queen"}}},
				}
				libraryTrack := model.MediaFile{
					ID: "br-live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
					Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
				}
				putTracks(ctx, ds, model.MediaFiles{libraryTrack})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("br-live"))
			})

			It("preserves duplicates when identical input songs match the same library track", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}, Album: "A Night at the Opera"},
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}, Album: "A Night at the Opera"},
				}
				libraryTrack := model.MediaFile{
					ID: "br", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera",
					Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
				}
				putTracks(ctx, ds, model.MediaFiles{libraryTrack})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].ID).To(Equal("br"))
				Expect(result[1].ID).To(Equal("br"))
			})
		})

		Context("priority ordering", func() {
			It("prefers ID match over MBID match", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
				songs := []agents.Song{
					{ID: "track-id", Name: "Song", MBID: "mbid-1", Artists: []agents.Artist{{Name: "Artist"}}},
				}
				putTracks(ctx, ds, model.MediaFiles{
					{ID: "track-mbid", Title: "Song", Artist: "Artist", MbzRecordingID: "mbid-1"},
					{ID: "track-id", Title: "Song", Artist: "Artist"},
				})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-id"))
			})
		})

		Context("count limit", func() {
			It("returns at most 'count' results", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{Name: "Artist"}}},
					{Name: "Song B", Artists: []agents.Artist{{Name: "Artist"}}},
					{Name: "Song C", Artists: []agents.Artist{{Name: "Artist"}}},
				}
				tracks := model.MediaFiles{
					{ID: "a", Title: "Song A", Artist: "Artist",
						Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
					},
					{ID: "b", Title: "Song B", Artist: "Artist",
						Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
					},
					{ID: "c", Title: "Song C", Artist: "Artist",
						Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
					},
				}
				putTracks(ctx, ds, tracks)
				result, err := m.MatchSongs(ctx, songs, 2)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
			})
		})

		Context("empty input", func() {
			It("returns empty results for no songs", func() {
				result, err := m.MatchSongs(ctx, []agents.Song{}, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("artist grouping", func() {
			It("groups title-phase tracks by participant artist ID, not display Artist", func() {
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{Name: "Daft Punk"}}},
				}
				// Display Artist differs from the query artist; only the participant
				// with order_artist_name "daft punk" routes to this query bucket.
				track := model.MediaFile{
					ID: "oan-track", Title: "Song A",
					Artist: "Daft Punk feat. Pharrell",
					Participants: artistParticipants(
						model.Artist{ID: "dp", Name: "Daft Punk", OrderArtistName: "daft punk"},
						model.Artist{ID: "ph", Name: "Pharrell", OrderArtistName: "pharrell"},
					),
				}
				putTracks(ctx, ds, model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("oan-track"))
			})

			It("matches a track that credits the searched artist as a collaborator", func() {
				songs := []agents.Song{
					{Name: "Crazy", Artists: []agents.Artist{{Name: "INXS"}}},
				}
				// "Par-T-One vs. INXS" — display Artist is the collaboration, but INXS is a
				// credited artist participant. Searching INXS must match it.
				track := model.MediaFile{
					ID: "collab", Title: "Crazy", Artist: "Par-T-One vs. INXS",
					Participants: artistParticipants(
						model.Artist{ID: "a-partone", Name: "Par-T-One", OrderArtistName: "par-t-one"},
						model.Artist{ID: "a-inxs", Name: "INXS", OrderArtistName: "inxs"},
					),
				}
				putTracks(ctx, ds, model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("collab"))
			})

			It("does not match a track where the searched artist is only the album artist", func() {
				songs := []agents.Song{
					{Name: "Qmart", Artists: []agents.Artist{{Name: "808 State"}}},
					{Name: "Hyperballad", Artists: []agents.Artist{{Name: "Björk"}}},
				}
				// Track performed by Björk on an "808 State" compilation: 808 State is the
				// albumartist, Björk is the performer. Searching 808 State must NOT match it.
				track := model.MediaFile{
					ID: "comp", Title: "Qmart", Artist: "Björk",
					Participants: model.Participants{
						model.RoleArtist: model.ParticipantList{
							{ID: "a-bjork", Name: "Björk", OrderArtistName: "bjork"},
						},
						model.RoleAlbumArtist: model.ParticipantList{
							{ID: "a-808", Name: "808 State", OrderArtistName: "808 state"},
						},
					},
				}
				// The Björk query fetches the track, so only the back-mapping keeps it away from 808 State.
				putTracks(ctx, ds, model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})

			It("resolves the artist by ArtistMBID when the name differs", func() {
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{Name: "Typo Artist", MBID: "mbid-9"}}},
				}
				track := model.MediaFile{
					ID: "by-mbid", Title: "Song A", Artist: "Correct Artist",
					Participants: artistParticipants(model.Artist{ID: "a9", Name: "Correct Artist", OrderArtistName: "correct artist", MbzArtistID: "mbid-9"}),
				}
				putTracks(ctx, ds, model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("by-mbid"))
			})

			It("resolves both queries when two share one ArtistMBID under different names", func() {
				// Two agent results for the same MusicBrainz artist but spelled differently
				// (an alias). Both must match the artist's track via the shared MBID.
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{Name: "Alias One", MBID: "mbid-shared"}}},
					{Name: "Song B", Artists: []agents.Artist{{Name: "Alias Two", MBID: "mbid-shared"}}},
				}
				artist := model.Artist{ID: "a-shared", Name: "Canonical", OrderArtistName: "canonical", MbzArtistID: "mbid-shared"}
				trackA := model.MediaFile{ID: "ta", Title: "Song A", Artist: "Canonical", Participants: artistParticipants(artist)}
				trackB := model.MediaFile{ID: "tb", Title: "Song B", Artist: "Canonical", Participants: artistParticipants(artist)}
				putTracks(ctx, ds, model.MediaFiles{trackA, trackB})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect([]string{result[0].ID, result[1].ID}).To(ConsistOf("ta", "tb"))
			})
		})

		Context("title phase DB errors", func() {
			BeforeEach(func() {
				ds = persistencetest.Stub(ds, persistencetest.Repos{MediaFile: func(r model.MediaFileRepository) model.MediaFileRepository {
					return failingTitleFetchRepo{r}
				}})
				m = matcher.New(ds)
			})

			It("returns an error when the title query fails and nothing else matched", func() {
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{Name: "Artist One"}}},
					{Name: "Song B", Artists: []agents.Artist{{Name: "Artist Two"}}},
				}
				putTracks(ctx, ds, model.MediaFiles{
					{ID: "a", Title: "Song A", Participants: artistParticipants(model.Artist{ID: "a1", Name: "Artist One", OrderArtistName: "artist one"})},
					{ID: "b", Title: "Song B", Participants: artistParticipants(model.Artist{ID: "a2", Name: "Artist Two", OrderArtistName: "artist two"})},
				})

				_, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).To(MatchError(errTitleFetch))
			})

			It("keeps exact-phase matches when the title query fails", func() {
				songs := []agents.Song{
					{ID: "track-1", Name: "Exact Song", Artists: []agents.Artist{{Name: "Exact Artist"}}},
					{Name: "Fuzzy Song", Artists: []agents.Artist{{Name: "Fuzzy Artist"}}},
				}
				putTracks(ctx, ds, model.MediaFiles{
					{ID: "track-1", Title: "Exact Song", Artist: "Exact Artist"},
					{ID: "fuzzy", Title: "Fuzzy Song", Participants: artistParticipants(model.Artist{ID: "fa", Name: "Fuzzy Artist", OrderArtistName: "fuzzy artist"})},
				})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-1"))
			})
		})

		Context("multiple artists", func() {
			It("prefers the track that shares more of the song's artists", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Life Is Good", Artists: []agents.Artist{{Name: "Drake"}, {Name: "Future"}}},
				}
				// Both candidates have display Artist "Drake", so they tie at specificity level 1
				// (name match). The deciding factor is artistOverlap: "both" credits Drake AND
				// Future (overlap 2), "one" credits only Drake (overlap 1).
				bothArtists := model.MediaFile{
					ID: "both", Title: "Life Is Good", Artist: "Drake",
					Participants: artistParticipants(
						model.Artist{ID: "drake", Name: "Drake", OrderArtistName: "drake"},
						model.Artist{ID: "future", Name: "Future", OrderArtistName: "future"},
					),
				}
				oneArtist := model.MediaFile{
					ID: "one", Title: "Life Is Good", Artist: "Drake",
					Participants: artistParticipants(
						model.Artist{ID: "drake", Name: "Drake", OrderArtistName: "drake"},
					),
				}
				putTracks(ctx, ds, model.MediaFiles{oneArtist, bothArtists})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("both"))
			})

			It("matches a single-artist song against a track crediting several artists", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Life Is Good", Artists: []agents.Artist{{Name: "Future"}}},
				}
				track := model.MediaFile{
					ID: "multi", Title: "Life Is Good", Artist: "Future feat. Drake",
					Participants: artistParticipants(
						model.Artist{ID: "future", Name: "Future", OrderArtistName: "future"},
						model.Artist{ID: "drake", Name: "Drake", OrderArtistName: "drake"},
					),
				}
				putTracks(ctx, ds, model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("multi"))
			})

			It("matches by a directly-supplied Navidrome artist ID (fast-path)", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{ID: "ar-x"}}},
				}
				track := model.MediaFile{
					ID: "by-id", Title: "Song A", Artist: "Some Artist",
					Participants: artistParticipants(
						model.Artist{ID: "ar-x", Name: "Some Artist", OrderArtistName: "some artist"},
					),
				}
				putTracks(ctx, ds, model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("by-id"))
			})

			It("prefers a higher artist-overlap track over a starred lower-overlap track", func() {
				conf.Update(func(c *conf.Options) {
					c.Matcher.PreferStarred = true
					c.Matcher.FuzzyThreshold = 85
				})
				songs := []agents.Song{
					{Name: "Collab Hit", Artists: []agents.Artist{{Name: "Drake"}, {Name: "Future"}}},
				}
				// Shares only Drake (overlap 1) but starred.
				starredOne := model.MediaFile{
					ID: "starred-one", Title: "Collab Hit",
					Starred:      true,
					Participants: artistParticipants(model.Artist{ID: "id-drake", Name: "Drake", OrderArtistName: "drake"}),
				}
				// Shares both (overlap 2), not starred.
				shareTwo := model.MediaFile{
					ID: "share-two", Title: "Collab Hit",
					Participants: artistParticipants(
						model.Artist{ID: "id-drake", Name: "Drake", OrderArtistName: "drake"},
						model.Artist{ID: "id-future", Name: "Future", OrderArtistName: "future"},
					),
				}
				putTracks(ctx, ds, model.MediaFiles{starredOne, shareTwo})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("share-two")) // overlap outranks the starred flag
			})
		})
	})

	Describe("MatchSongsIndexed", func() {
		It("returns index-keyed map of matched songs", func() {
			songs := []agents.Song{
				{ID: "track-1", Name: "Song One", Artists: []agents.Artist{{Name: "Artist A"}}},
				{ID: "track-2", Name: "Song Two", Artists: []agents.Artist{{Name: "Artist B"}}},
				{ID: "track-3", Name: "Song Three", Artists: []agents.Artist{{Name: "Artist C"}}},
			}
			mf1 := model.MediaFile{ID: "track-1", Title: "Song One", Artist: "Artist A"}
			mf2 := model.MediaFile{ID: "track-2", Title: "Song Two", Artist: "Artist B"}

			putTracks(ctx, ds, model.MediaFiles{mf1, mf2})

			result, err := m.MatchSongsIndexed(ctx, songs)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("track-1"))
			Expect(result[1].ID).To(Equal("track-2"))
			_, exists := result[2]
			Expect(exists).To(BeFalse())
		})

		It("preserves original indices when some songs don't match", func() {
			songs := []agents.Song{
				{Name: "Unknown Song", Artists: []agents.Artist{{Name: "Unknown Artist"}}},
				{ID: "track-1", Name: "Known Song", Artists: []agents.Artist{{Name: "Known Artist"}}},
			}
			mf1 := model.MediaFile{ID: "track-1", Title: "Known Song", Artist: "Known Artist"}

			putTracks(ctx, ds, model.MediaFiles{mf1})

			result, err := m.MatchSongsIndexed(ctx, songs)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			_, exists := result[0]
			Expect(exists).To(BeFalse())
			Expect(result[1].ID).To(Equal("track-1"))
		})

		It("returns empty map for empty input", func() {
			result, err := m.MatchSongsIndexed(ctx, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("specificity level matching", func() {
		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
		})

		It("matches by title + artist MBID + album MBID (highest priority)", func() {
			correctMatch := model.MediaFile{
				ID: "correct-match", Title: "Similar Song", Artist: "Depeche Mode", Album: "Violator",
				MbzArtistID: "artist-mbid-123", MbzAlbumID: "album-mbid-456",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode", MbzArtistID: "artist-mbid-123"}),
			}
			wrongMatch := model.MediaFile{
				ID: "wrong-match", Title: "Similar Song", Artist: "Depeche Mode", Album: "Some Other Album",
				MbzArtistID: "artist-mbid-123", MbzAlbumID: "different-album-mbid",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode", MbzArtistID: "artist-mbid-123"}),
			}
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Depeche Mode", MBID: "artist-mbid-123"}}, Album: "Violator", AlbumMBID: "album-mbid-456"},
			}

			putTracks(ctx, ds, model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("matches by title + artist name + album name when MBIDs unavailable", func() {
			correctMatch := model.MediaFile{
				ID: "correct-match", Title: "Similar Song", Artist: "depeche mode", Album: "violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			wrongMatch := model.MediaFile{
				ID: "wrong-match", Title: "Similar Song", Artist: "Other Artist", Album: "Other Album",
				Participants: artistParticipants(model.Artist{ID: "oa", Name: "Other Artist", OrderArtistName: "other artist"}),
			}
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}

			putTracks(ctx, ds, model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("matches by title + artist only when album info unavailable", func() {
			correctMatch := model.MediaFile{
				ID: "correct-match", Title: "Similar Song", Artist: "depeche mode", Album: "Some Album",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			wrongMatch := model.MediaFile{
				ID: "wrong-match", Title: "Similar Song", Artist: "Other Artist", Album: "Other Album",
				Participants: artistParticipants(model.Artist{ID: "oa", Name: "Other Artist", OrderArtistName: "other artist"}),
			}
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
			}

			putTracks(ctx, ds, model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("does not match songs without artist info", func() {
			songs := []agents.Song{
				{Name: "Similar Song"},
			}
			putTracks(ctx, ds, model.MediaFiles{{ID: "t", Title: "Similar Song", Artist: "Depeche Mode",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("returns distinct matches for each artist's version (covers scenario)", func() {
			cover1 := model.MediaFile{ID: "cover-1", Title: "Yesterday", Artist: "The Beatles", Album: "Help!",
				Participants: artistParticipants(model.Artist{ID: "beatles", Name: "The Beatles", OrderArtistName: "beatles"}),
			}
			cover2 := model.MediaFile{ID: "cover-2", Title: "Yesterday", Artist: "Ray Charles", Album: "Greatest Hits",
				Participants: artistParticipants(model.Artist{ID: "ray-charles", Name: "Ray Charles", OrderArtistName: "ray charles"}),
			}
			cover3 := model.MediaFile{ID: "cover-3", Title: "Yesterday", Artist: "Frank Sinatra", Album: "My Way",
				Participants: artistParticipants(model.Artist{ID: "sinatra", Name: "Frank Sinatra", OrderArtistName: "frank sinatra"}),
			}

			songs := []agents.Song{
				{Name: "Yesterday", Artists: []agents.Artist{{Name: "The Beatles"}}, Album: "Help!"},
				{Name: "Yesterday", Artists: []agents.Artist{{Name: "Ray Charles"}}, Album: "Greatest Hits"},
				{Name: "Yesterday", Artists: []agents.Artist{{Name: "Frank Sinatra"}}, Album: "My Way"},
			}

			putTracks(ctx, ds, model.MediaFiles{cover1, cover2, cover3})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(3))
			ids := []string{result[0].ID, result[1].ID, result[2].ID}
			Expect(ids).To(ContainElements("cover-1", "cover-2", "cover-3"))
		})

		It("prefers more precise matches for each song", func() {
			preciseMatch := model.MediaFile{
				ID: "precise", Title: "Song A", Artist: "Artist One", Album: "Album One",
				MbzArtistID: "mbid-1", MbzAlbumID: "album-mbid-1",
				Participants: artistParticipants(model.Artist{ID: "a1", Name: "Artist One", OrderArtistName: "artist one", MbzArtistID: "mbid-1"}),
			}
			lessAccurateMatch := model.MediaFile{
				ID: "less-accurate", Title: "Song A", Artist: "Artist One", Album: "Compilation",
				MbzArtistID:  "mbid-1",
				Participants: artistParticipants(model.Artist{ID: "a1", Name: "Artist One", OrderArtistName: "artist one", MbzArtistID: "mbid-1"}),
			}
			artistTwoMatch := model.MediaFile{
				ID: "artist-two", Title: "Song B", Artist: "Artist Two",
				Participants: artistParticipants(model.Artist{ID: "a2", Name: "Artist Two", OrderArtistName: "artist two"}),
			}

			songs := []agents.Song{
				{Name: "Song A", Artists: []agents.Artist{{Name: "Artist One", MBID: "mbid-1"}}, Album: "Album One", AlbumMBID: "album-mbid-1"},
				{Name: "Song B", Artists: []agents.Artist{{Name: "Artist Two"}}},
			}

			putTracks(ctx, ds, model.MediaFiles{lessAccurateMatch, preciseMatch, artistTwoMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("precise"))
			Expect(result[1].ID).To(Equal("artist-two"))
		})

		It("uses the resolved artist MBID for specificity (level 5)", func() {
			songs := []agents.Song{
				{Name: "Song A", Artists: []agents.Artist{{Name: "Artist One", MBID: "mbid-1"}}, Album: "Album One", AlbumMBID: "album-mbid-1"},
			}
			// Two tracks with the same title and album; only the one whose resolved artist
			// carries mbid-1 (and whose album MBID matches) wins via Level 5. Without the
			// resolved MBID, both tracks tie at Level 3 (name+album) and the first wins by
			// chance — verifiable by RED-proof: see task-2-report.md.
			precise := model.MediaFile{
				ID: "precise", Title: "Song A", Artist: "Artist One", Album: "Album One", MbzAlbumID: "album-mbid-1",
				Participants: artistParticipants(model.Artist{ID: "a1", Name: "Artist One", OrderArtistName: "artist one", MbzArtistID: "mbid-1"}),
			}
			other := model.MediaFile{
				ID: "other", Title: "Song A", Artist: "Artist One", Album: "Album One", MbzAlbumID: "wrong-album-mbid",
				Participants: artistParticipants(model.Artist{ID: "a1b", Name: "Artist One", OrderArtistName: "artist one", MbzArtistID: ""}),
			}
			// Artist resolution returns both a1 (by name+mbid) and a1b (by name).
			putTracks(ctx, ds, model.MediaFiles{other, precise})

			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("precise"))
		})
	})

	Describe("fuzzy matching thresholds", func() {
		Context("with default threshold (85%)", func() {
			It("matches songs with remastered suffix", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })

				songs := []agents.Song{
					{Name: "Paranoid Android", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead",
						Participants: artistParticipants(model.Artist{ID: "rh", Name: "Radiohead", OrderArtistName: "radiohead"}),
					},
				}

				putTracks(ctx, ds, artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("remastered"))
			})

			It("matches songs with live suffix", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })

				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
						Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
					},
				}

				putTracks(ctx, ds, artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("live"))
			})
		})

		Context("with threshold set to 100 (exact match only)", func() {
			It("only matches exact titles", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })

				songs := []agents.Song{
					{Name: "Paranoid Android", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead",
						Participants: artistParticipants(model.Artist{ID: "rh", Name: "Radiohead", OrderArtistName: "radiohead"}),
					},
				}

				putTracks(ctx, ds, artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("with lower threshold (75%)", func() {
			It("matches more aggressively", func() {
				conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 75 })

				songs := []agents.Song{
					{Name: "Song", Artists: []agents.Artist{{Name: "Artist"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "extended", Title: "Song (Extended Mix)", Artist: "Artist",
						Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
					},
				}

				putTracks(ctx, ds, artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("extended"))
			})
		})
	})

	Describe("fuzzy album matching", func() {
		BeforeEach(func() {
			conf.Update(func(c *conf.Options) {
				c.Matcher.FuzzyThreshold = 85
				c.Matcher.PreferStarred = false
			})
		})

		It("matches album with (Remaster) suffix", func() {
			songs := []agents.Song{
				{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}, Album: "A Night at the Opera"},
			}
			correctMatch := model.MediaFile{
				ID: "correct", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera (2011 Remaster)",
				Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
			}
			wrongMatch := model.MediaFile{
				ID: "wrong", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "Greatest Hits",
				Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
			}

			putTracks(ctx, ds, model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct"))
		})

		It("matches album with (Deluxe Edition) suffix", func() {
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			correctMatch := model.MediaFile{
				ID: "correct", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator (Deluxe Edition)",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			wrongMatch := model.MediaFile{
				ID: "wrong", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "101",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}

			putTracks(ctx, ds, model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct"))
		})

		It("prefers exact album match over fuzzy album match", func() {
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			exactMatch := model.MediaFile{
				ID: "exact", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			fuzzyMatch := model.MediaFile{
				ID: "fuzzy", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator (Deluxe Edition)",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}

			putTracks(ctx, ds, model.MediaFiles{fuzzyMatch, exactMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("exact"))
		})

		It("prefers a more specific match over a starred track when PreferStarred is enabled", func() {
			conf.Update(func(c *conf.Options) { c.Matcher.PreferStarred = true })
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			albumMatch := model.MediaFile{
				ID: "album-match", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			starredTrack := model.MediaFile{
				ID: "starred", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Singles",
				Starred:      true,
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			putTracks(ctx, ds, model.MediaFiles{albumMatch, starredTrack})
			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("album-match")) // specificity now outranks the starred flag
		})

		It("prefers a more specific match over a 4-star track when PreferStarred is enabled", func() {
			conf.Update(func(c *conf.Options) { c.Matcher.PreferStarred = true })
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			albumMatch := model.MediaFile{
				ID: "album-match", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			ratedTrack := model.MediaFile{
				ID: "rated", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Singles",
				Rating:       4,
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			putTracks(ctx, ds, model.MediaFiles{albumMatch, ratedTrack})
			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("album-match")) // specificity now outranks the 4-star rating
		})

		It("prefers a starred track when specificity and overlap are equal", func() {
			conf.Update(func(c *conf.Options) { c.Matcher.PreferStarred = true })
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			// Both credit the same single artist and the same album → equal specificity AND equal overlap.
			plain := model.MediaFile{
				ID: "plain", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			starred := model.MediaFile{
				ID: "starred", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Starred:      true,
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			putTracks(ctx, ds, model.MediaFiles{plain, starred})
			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("starred")) // preferred still wins the tie
		})
	})

	Describe("duration matching", func() {
		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 100 })
		})

		It("prefers tracks with matching duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 180000},
			}
			correctMatch := model.MediaFile{
				ID: "correct", Title: "Similar Song", Artist: "Test Artist", Duration: 180.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}
			wrongDuration := model.MediaFile{
				ID: "wrong", Title: "Similar Song", Artist: "Test Artist", Duration: 240.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{wrongDuration, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct"))
		})

		It("matches tracks with close duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 180000},
			}
			closeDuration := model.MediaFile{
				ID: "close-duration", Title: "Similar Song", Artist: "Test Artist", Duration: 182.5,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{closeDuration})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("close-duration"))
		})

		It("prefers closer duration over farther duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 180000},
			}
			closeDuration := model.MediaFile{
				ID: "close", Title: "Similar Song", Artist: "Test Artist", Duration: 181.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}
			farDuration := model.MediaFile{
				ID: "far", Title: "Similar Song", Artist: "Test Artist", Duration: 190.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{farDuration, closeDuration})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("close"))
		})

		It("still matches when no tracks have matching duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 180000},
			}
			differentDuration := model.MediaFile{
				ID: "different", Title: "Similar Song", Artist: "Test Artist", Duration: 300.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{differentDuration})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("different"))
		})

		It("prefers title match over duration match when titles differ", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 180000},
			}
			differentTitle := model.MediaFile{
				ID: "wrong-title", Title: "Different Song", Artist: "Test Artist", Duration: 180.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}
			correctTitle := model.MediaFile{
				ID: "correct-title", Title: "Similar Song", Artist: "Test Artist", Duration: 300.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{differentTitle, correctTitle})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-title"))
		})

		It("matches without duration filtering when agent duration is 0", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 0},
			}
			anyTrack := model.MediaFile{
				ID: "any", Title: "Similar Song", Artist: "Test Artist", Duration: 999.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{anyTrack})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("any"))
		})

		It("handles very short songs with close duration", func() {
			songs := []agents.Song{
				{Name: "Short Song", Artists: []agents.Artist{{Name: "Test Artist"}}, Duration: 30000},
			}
			shortTrack := model.MediaFile{
				ID: "short", Title: "Short Song", Artist: "Test Artist", Duration: 31.0,
				Participants: artistParticipants(model.Artist{ID: "ta", Name: "Test Artist", OrderArtistName: "test artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{shortTrack})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("short"))
		})

		It("matches same title+artist songs to their own closest-duration track", func() {
			songs := []agents.Song{
				{Name: "Same Song", Artists: []agents.Artist{{Name: "Same Artist"}}, Duration: 180000},
				{Name: "Same Song", Artists: []agents.Artist{{Name: "Same Artist"}}, Duration: 240000},
			}
			shortTrack := model.MediaFile{
				ID: "short", Title: "Same Song", Artist: "Same Artist", Duration: 180.0,
				Participants: artistParticipants(model.Artist{ID: "sa", Name: "Same Artist", OrderArtistName: "same artist"}),
			}
			longTrack := model.MediaFile{
				ID: "long", Title: "Same Song", Artist: "Same Artist", Duration: 240.0,
				Participants: artistParticipants(model.Artist{ID: "sa", Name: "Same Artist", OrderArtistName: "same artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{shortTrack, longTrack})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("short"))
			Expect(result[1].ID).To(Equal("long"))
		})
	})

	Describe("deduplication edge cases", func() {
		BeforeEach(func() {
			conf.Update(func(c *conf.Options) { c.Matcher.FuzzyThreshold = 85 })
		})

		It("handles mixed scenario with both identical and different input songs", func() {
			songs := []agents.Song{
				{Name: "Yesterday", Artists: []agents.Artist{{Name: "The Beatles"}}, Album: "Help!"},
				{Name: "Yesterday (Remastered)", Artists: []agents.Artist{{Name: "The Beatles"}}, Album: "1"},
				{Name: "Yesterday", Artists: []agents.Artist{{Name: "The Beatles"}}, Album: "Help!"},
				{Name: "Yesterday (Anthology)", Artists: []agents.Artist{{Name: "The Beatles"}}, Album: "Anthology"},
			}
			libraryTrack := model.MediaFile{
				ID: "yesterday", Title: "Yesterday", Artist: "The Beatles", Album: "Help!",
				Participants: artistParticipants(model.Artist{ID: "beatles", Name: "The Beatles", OrderArtistName: "beatles"}),
			}

			putTracks(ctx, ds, model.MediaFiles{libraryTrack})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("yesterday"))
			Expect(result[1].ID).To(Equal("yesterday"))
		})

		It("does not deduplicate songs that match different library tracks", func() {
			songs := []agents.Song{
				{Name: "Song A", Artists: []agents.Artist{{Name: "Artist"}}},
				{Name: "Song B", Artists: []agents.Artist{{Name: "Artist"}}},
				{Name: "Song C", Artists: []agents.Artist{{Name: "Artist"}}},
			}
			trackA := model.MediaFile{ID: "track-a", Title: "Song A", Artist: "Artist",
				Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
			}
			trackB := model.MediaFile{ID: "track-b", Title: "Song B", Artist: "Artist",
				Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
			}
			trackC := model.MediaFile{ID: "track-c", Title: "Song C", Artist: "Artist",
				Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{trackA, trackB, trackC})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0].ID).To(Equal("track-a"))
			Expect(result[1].ID).To(Equal("track-b"))
			Expect(result[2].ID).To(Equal("track-c"))
		})

		It("respects count limit after deduplication", func() {
			songs := []agents.Song{
				{Name: "Song A", Artists: []agents.Artist{{Name: "Artist"}}},
				{Name: "Song A (Live)", Artists: []agents.Artist{{Name: "Artist"}}},
				{Name: "Song B", Artists: []agents.Artist{{Name: "Artist"}}},
				{Name: "Song B (Remix)", Artists: []agents.Artist{{Name: "Artist"}}},
			}
			trackA := model.MediaFile{ID: "track-a", Title: "Song A", Artist: "Artist",
				Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
			}
			trackB := model.MediaFile{ID: "track-b", Title: "Song B", Artist: "Artist",
				Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
			}

			putTracks(ctx, ds, model.MediaFiles{trackA, trackB})

			result, err := m.MatchSongs(ctx, songs, 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("track-a"))
			Expect(result[1].ID).To(Equal("track-b"))
		})
	})
})

var errTitleFetch = errors.New("db down")

type failingTitleFetchRepo struct{ model.MediaFileRepository }

func (r failingTitleFetchRepo) GetAll(ctx context.Context, options ...model.QueryOptions) (model.MediaFiles, error) {
	if len(options) > 0 && isTitleFetch(options[0]) {
		return nil, errTitleFetch
	}
	return r.MediaFileRepository.GetAll(ctx, options...)
}

// isTitleFetch reports whether opt is the title phase's track fetch, the only query that reads media_file_artists.
func isTitleFetch(opt model.QueryOptions) bool {
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
}

// putTracks stores the tracks in library 1 with the artists credited in their participants.
// Starred and Rating are saved as the logged user's annotations.
func putTracks(ctx context.Context, ds model.DataStore, tracks model.MediaFiles) {
	GinkgoHelper()
	for _, t := range tracks {
		for _, list := range t.Participants {
			for _, p := range list {
				persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, p.Artist)
			}
		}
		persistencetest.PutMediaFiles(ctx, ds, t)
		if t.Starred {
			Expect(ds.MediaFile().SetStar(ctx, true, t.ID)).To(Succeed())
		}
		if t.Rating > 0 {
			Expect(ds.MediaFile().SetRating(ctx, t.Rating, t.ID)).To(Succeed())
		}
	}
}

// artistParticipants builds a Participants map crediting the given artists under RoleArtist.
func artistParticipants(artists ...model.Artist) model.Participants {
	list := make(model.ParticipantList, len(artists))
	for i, a := range artists {
		list[i] = model.Participant{Artist: a}
	}
	return model.Participants{model.RoleArtist: list}
}
