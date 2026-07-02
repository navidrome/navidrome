package matcher_test

import (
	"context"
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Matcher", func() {
	var ds model.DataStore
	var mediaFileRepo *mockMediaFileRepo
	var artistRepo *mockArtistRepo
	var ctx context.Context
	var m *matcher.Matcher

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		mediaFileRepo = newMockMediaFileRepo()
		artistRepo = newMockArtistRepo()
		DeferCleanup(func() {
			mediaFileRepo.AssertExpectations(GinkgoT())
		})
		ds = &tests.MockDataStore{
			MockedMediaFile: mediaFileRepo,
			MockedArtist:    artistRepo,
		}
		m = matcher.New(ds)
	})

	// Per-phase expectation helpers. Each `expect*Phase` registers a .Once() expectation
	// that will fail the suite via AssertExpectations if the phase is NOT called. Tests
	// use these to deterministically verify which matching phases fire. Phases that may
	// or may not fire should use the `allow*Phase` variants instead, which register
	// .Maybe() fallbacks.
	expectIDPhase := func(matches model.MediaFiles) {
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("media_file.id"))).
			Return(matches, nil).Once()
	}
	expectMBIDPhase := func(matches model.MediaFiles) {
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("mbz_recording_id"))).
			Return(matches, nil).Once()
	}
	expectISRCPhase := func(matches model.MediaFiles) {
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInEq("missing"))).
			Return(matches, nil).Once()
	}

	// allowIdentifierPhases installs .Maybe() catch-alls for the ID/MBID/ISRC phases so
	// tests that only care about the title phase don't fail on those unexpected calls.
	allowIdentifierPhases := func() {
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("media_file.id"))).
			Return(model.MediaFiles{}, nil).Maybe()
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("mbz_recording_id"))).
			Return(model.MediaFiles{}, nil).Maybe()
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInEq("missing"))).
			Return(model.MediaFiles{}, nil).Maybe()
	}

	// allowOtherPhases installs .Maybe() catch-alls so phases that short-circuit (return
	// early without hitting the DB) don't cause test failures for unexpected calls. Call
	// this after expect*Phase for the phases the test actually wants to verify.
	allowOtherPhases := func() {
		allowIdentifierPhases()
		artistRepo.On("GetAll", mock.Anything).Return(model.Artists{}, nil).Maybe()
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
			Return(model.MediaFiles{}, nil).Maybe()
	}

	// allowTitlePhase wires title matching from a list of library tracks. Each track must carry
	// Participants[RoleArtist] with the artist IDs that credit it; the helper derives the artist
	// rows the artist resolution returns from those participants, then returns the tracks from
	// the track-fetch query.
	allowTitlePhase := func(tracks model.MediaFiles) {
		// Artist resolution: build artist rows from the tracks' participants.
		seen := map[string]model.Artist{}
		for _, t := range tracks {
			for _, p := range t.Participants[model.RoleArtist] {
				if _, ok := seen[p.ID]; !ok {
					seen[p.ID] = p.Artist
				}
			}
		}
		artists := make(model.Artists, 0, len(seen))
		for _, a := range seen {
			artists = append(artists, a)
		}
		artistRepo.On("GetAll", mock.Anything).Return(artists, nil).Maybe()
		// Track fetch (media_file_artists subquery).
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
			Return(tracks, nil).Maybe()
	}

	Describe("MatchSongs", func() {
		Context("matching by direct ID", func() {
			It("matches songs with an ID field to MediaFiles by ID", func() {
				conf.Server.Matcher.FuzzyThreshold = 100
				songs := []agents.Song{
					{ID: "track-1", Name: "Some Song", Artists: []agents.Artist{{Name: "Some Artist"}}},
				}
				idMatch := model.MediaFile{
					ID: "track-1", Title: "Some Song", Artist: "Some Artist",
				}
				expectIDPhase(model.MediaFiles{idMatch})
				allowOtherPhases()
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-1"))
			})
		})

		Context("matching by MBID", func() {
			It("matches songs with MBID to tracks with matching mbz_recording_id", func() {
				conf.Server.Matcher.FuzzyThreshold = 100
				songs := []agents.Song{
					{Name: "Paranoid Android", MBID: "abc-123", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				mbidMatch := model.MediaFile{
					ID: "track-mbid", Title: "Paranoid Android", Artist: "Radiohead",
					MbzRecordingID: "abc-123",
				}
				expectMBIDPhase(model.MediaFiles{mbidMatch})
				allowOtherPhases()
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-mbid"))
			})
		})

		Context("matching by ISRC", func() {
			It("matches songs with ISRC to tracks with matching ISRC tag", func() {
				conf.Server.Matcher.FuzzyThreshold = 100
				songs := []agents.Song{
					{Name: "Paranoid Android", ISRC: "GBAYE0000351", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				isrcMatch := model.MediaFile{
					ID: "track-isrc", Title: "Paranoid Android", Artist: "Radiohead",
					Tags: model.Tags{model.TagISRC: []string{"GBAYE0000351"}},
				}
				expectISRCPhase(model.MediaFiles{isrcMatch})
				allowOtherPhases()
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-isrc"))
			})
		})

		Context("fuzzy title+artist matching", func() {
			It("matches songs by title and artist name", func() {
				conf.Server.Matcher.FuzzyThreshold = 100
				songs := []agents.Song{
					{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}},
				}
				titleMatch := model.MediaFile{
					ID: "track-title", Title: "Enjoy the Silence", Artist: "Depeche Mode",
					Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
				}
				allowTitlePhase(model.MediaFiles{titleMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-title"))
			})

			It("matches songs with fuzzy title similarity", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}},
				}
				fuzzyMatch := model.MediaFile{
					ID: "track-fuzzy", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
					Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
				}
				allowTitlePhase(model.MediaFiles{fuzzyMatch})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-fuzzy"))
			})

			It("does not match completely different titles", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
				songs := []agents.Song{
					{Name: "Yesterday", Artists: []agents.Artist{{Name: "The Beatles"}}},
				}
				differentTracks := model.MediaFiles{
					{ID: "different", Title: "Tomorrow Never Knows", Artist: "The Beatles",
						Participants: artistParticipants(model.Artist{ID: "beatles", Name: "The Beatles", OrderArtistName: "beatles"}),
					},
				}
				allowTitlePhase(differentTracks)
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("deduplication", func() {
			It("removes duplicates when different input songs match the same library track", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody (Live)", Artists: []agents.Artist{{Name: "Queen"}}},
					{Name: "Bohemian Rhapsody (Original Mix)", Artists: []agents.Artist{{Name: "Queen"}}},
				}
				libraryTrack := model.MediaFile{
					ID: "br-live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
					Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
				}
				allowTitlePhase(model.MediaFiles{libraryTrack})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("br-live"))
			})

			It("preserves duplicates when identical input songs match the same library track", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}, Album: "A Night at the Opera"},
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}, Album: "A Night at the Opera"},
				}
				libraryTrack := model.MediaFile{
					ID: "br", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera",
					Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
				}
				allowTitlePhase(model.MediaFiles{libraryTrack})
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].ID).To(Equal("br"))
				Expect(result[1].ID).To(Equal("br"))
			})
		})

		Context("priority ordering", func() {
			It("prefers ID match over MBID match", func() {
				conf.Server.Matcher.FuzzyThreshold = 100
				// Song has both ID and MBID set. The matcher should resolve via ID
				// and short-circuit the MBID phase entirely, so no MBID fetch should
				// occur even though an mbz_recording_id exists in the input.
				songs := []agents.Song{
					{ID: "track-id", Name: "Song", MBID: "mbid-1", Artists: []agents.Artist{{Name: "Artist"}}},
				}
				idMatch := model.MediaFile{
					ID: "track-id", Title: "Song", Artist: "Artist",
				}
				expectIDPhase(model.MediaFiles{idMatch})
				allowOtherPhases()
				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-id"))
			})
		})

		Context("count limit", func() {
			It("returns at most 'count' results", func() {
				conf.Server.Matcher.FuzzyThreshold = 100
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
				allowTitlePhase(tracks)
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
				allowTitlePhase(model.MediaFiles{track})

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
				allowTitlePhase(model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("collab"))
			})

			It("does not match a track where the searched artist is only the album artist", func() {
				songs := []agents.Song{
					{Name: "Qmart", Artists: []agents.Artist{{Name: "808 State"}}},
				}
				// Track performed by Björk on an "808 State" compilation: 808 State is the
				// albumartist, Björk is the performer. Searching 808 State must NOT match it.
				track := model.MediaFile{
					ID: "comp", Title: "Qmart", Artist: "Björk",
					Participants: model.Participants{
						model.RoleArtist: model.ParticipantList{
							{Artist: model.Artist{ID: "a-bjork", Name: "Björk", OrderArtistName: "bjork"}},
						},
						model.RoleAlbumArtist: model.ParticipantList{
							{Artist: model.Artist{ID: "a-808", Name: "808 State", OrderArtistName: "808 state"}},
						},
					},
				}
				// Artist resolution returns "808 state" only if some artist row matches; here the
				// album-artist participant exists but is NOT role='artist', so the track-fetch query's
				// EXISTS (role='artist') would not return the track in production. The mock
				// returns it anyway; back-mapping must drop it because no role='artist'
				// participant is a resolved artist for the query "808 state".
				artistRepo.On("GetAll", mock.Anything).
					Return(model.Artists{{ID: "a-808", Name: "808 State", OrderArtistName: "808 state"}}, nil).Maybe()
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
					Return(model.MediaFiles{track}, nil).Maybe()

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
				// Artist resolution returns the artist matched by mbz_artist_id; its order name
				// ("correct artist") differs from the query name ("typo artist"), so
				// resolution must come from the MBID branch.
				artistRepo.On("GetAll", mock.Anything).
					Return(model.Artists{{ID: "a9", Name: "Correct Artist", OrderArtistName: "correct artist", MbzArtistID: "mbid-9"}}, nil).Maybe()
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
					Return(model.MediaFiles{track}, nil).Maybe()

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
				artistRepo.On("GetAll", mock.Anything).
					Return(model.Artists{{ID: "a-shared", Name: "Canonical", OrderArtistName: "canonical", MbzArtistID: "mbid-shared"}}, nil).Maybe()
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
					Return(model.MediaFiles{trackA, trackB}, nil).Maybe()

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect([]string{result[0].ID, result[1].ID}).To(ConsistOf("ta", "tb"))
			})
		})

		// These tests register their own track-fetch expectations per-test (to inject
		// an error), so they use allowIdentifierPhases — NOT allowOtherPhases, which would
		// add a .Maybe() title-phase catch-all that masks the injected error.
		Context("title phase DB errors", func() {
			It("returns an error when the title query fails and nothing else matched", func() {
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{Name: "Artist One"}}},
					{Name: "Song B", Artists: []agents.Artist{{Name: "Artist Two"}}},
				}
				allowIdentifierPhases()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{
					{ID: "a1", Name: "Artist One", OrderArtistName: "artist one"},
					{ID: "a2", Name: "Artist Two", OrderArtistName: "artist two"},
				}, nil)
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
					Return(nil, errors.New("db down"))

				_, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("db down"))
			})

			It("keeps exact-phase matches when the title query fails", func() {
				songs := []agents.Song{
					{ID: "track-1", Name: "Exact Song", Artists: []agents.Artist{{Name: "Exact Artist"}}},
					{Name: "Fuzzy Song", Artists: []agents.Artist{{Name: "Fuzzy Artist"}}},
				}
				idMatch := model.MediaFile{ID: "track-1", Title: "Exact Song", Artist: "Exact Artist"}
				expectIDPhase(model.MediaFiles{idMatch})
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("mbz_recording_id"))).
					Return(model.MediaFiles{}, nil).Maybe()
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInEq("missing"))).
					Return(model.MediaFiles{}, nil).Maybe()
				artistRepo.On("GetAll", mock.Anything).Return(model.Artists{
					{ID: "fa", Name: "Fuzzy Artist", OrderArtistName: "fuzzy artist"},
				}, nil)
				mediaFileRepo.On("GetAll", mock.MatchedBy(matchTracksByArtistQuery())).
					Return(nil, errors.New("db down"))

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-1"))
			})
		})

		Context("multiple artists", func() {
			It("prefers the track that shares more of the song's artists", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
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
				allowTitlePhase(model.MediaFiles{oneArtist, bothArtists})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("both"))
			})

			It("matches a single-artist song against a track crediting several artists", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
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
				allowTitlePhase(model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("multi"))
			})

			It("matches by a directly-supplied Navidrome artist ID (fast-path)", func() {
				conf.Server.Matcher.FuzzyThreshold = 85
				songs := []agents.Song{
					{Name: "Song A", Artists: []agents.Artist{{ID: "ar-x"}}},
				}
				track := model.MediaFile{
					ID: "by-id", Title: "Song A", Artist: "Some Artist",
					Participants: artistParticipants(
						model.Artist{ID: "ar-x", Name: "Some Artist", OrderArtistName: "some artist"},
					),
				}
				allowTitlePhase(model.MediaFiles{track})

				result, err := m.MatchSongs(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("by-id"))
			})

			It("prefers a higher artist-overlap track over a starred lower-overlap track", func() {
				conf.Server.Matcher.PreferStarred = true
				conf.Server.Matcher.FuzzyThreshold = 85
				songs := []agents.Song{
					{Name: "Collab Hit", Artists: []agents.Artist{{Name: "Drake"}, {Name: "Future"}}},
				}
				// Shares only Drake (overlap 1) but starred.
				starredOne := model.MediaFile{
					ID: "starred-one", Title: "Collab Hit",
					Annotations:  model.Annotations{Starred: true},
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
				allowTitlePhase(model.MediaFiles{starredOne, shareTwo})
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

			expectIDPhase(model.MediaFiles{mf1, mf2})
			allowOtherPhases()

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

			expectIDPhase(model.MediaFiles{mf1})
			allowOtherPhases()

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
			conf.Server.Matcher.FuzzyThreshold = 100
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

			allowTitlePhase(model.MediaFiles{wrongMatch, correctMatch})

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

			allowTitlePhase(model.MediaFiles{wrongMatch, correctMatch})

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

			allowTitlePhase(model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("does not match songs without artist info", func() {
			songs := []agents.Song{
				{Name: "Similar Song"},
			}

			allowTitlePhase(model.MediaFiles{})

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

			allowTitlePhase(model.MediaFiles{cover1, cover2, cover3})

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

			allowTitlePhase(model.MediaFiles{lessAccurateMatch, preciseMatch, artistTwoMatch})

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
			allowTitlePhase(model.MediaFiles{other, precise})

			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("precise"))
		})
	})

	Describe("fuzzy matching thresholds", func() {
		Context("with default threshold (85%)", func() {
			It("matches songs with remastered suffix", func() {
				conf.Server.Matcher.FuzzyThreshold = 85

				songs := []agents.Song{
					{Name: "Paranoid Android", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead",
						Participants: artistParticipants(model.Artist{ID: "rh", Name: "Radiohead", OrderArtistName: "radiohead"}),
					},
				}

				allowTitlePhase(artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("remastered"))
			})

			It("matches songs with live suffix", func() {
				conf.Server.Matcher.FuzzyThreshold = 85

				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artists: []agents.Artist{{Name: "Queen"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
						Participants: artistParticipants(model.Artist{ID: "queen", Name: "Queen", OrderArtistName: "queen"}),
					},
				}

				allowTitlePhase(artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("live"))
			})
		})

		Context("with threshold set to 100 (exact match only)", func() {
			It("only matches exact titles", func() {
				conf.Server.Matcher.FuzzyThreshold = 100

				songs := []agents.Song{
					{Name: "Paranoid Android", Artists: []agents.Artist{{Name: "Radiohead"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead",
						Participants: artistParticipants(model.Artist{ID: "rh", Name: "Radiohead", OrderArtistName: "radiohead"}),
					},
				}

				allowTitlePhase(artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("with lower threshold (75%)", func() {
			It("matches more aggressively", func() {
				conf.Server.Matcher.FuzzyThreshold = 75

				songs := []agents.Song{
					{Name: "Song", Artists: []agents.Artist{{Name: "Artist"}}},
				}
				artistTracks := model.MediaFiles{
					{ID: "extended", Title: "Song (Extended Mix)", Artist: "Artist",
						Participants: artistParticipants(model.Artist{ID: "art", Name: "Artist", OrderArtistName: "artist"}),
					},
				}

				allowTitlePhase(artistTracks)

				result, err := m.MatchSongs(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("extended"))
			})
		})
	})

	Describe("fuzzy album matching", func() {
		BeforeEach(func() {
			conf.Server.Matcher.FuzzyThreshold = 85
			conf.Server.Matcher.PreferStarred = false
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

			allowTitlePhase(model.MediaFiles{wrongMatch, correctMatch})

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

			allowTitlePhase(model.MediaFiles{wrongMatch, correctMatch})

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

			allowTitlePhase(model.MediaFiles{fuzzyMatch, exactMatch})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("exact"))
		})

		It("prefers a more specific match over a starred track when PreferStarred is enabled", func() {
			conf.Server.Matcher.PreferStarred = true
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			albumMatch := model.MediaFile{
				ID: "album-match", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			starredTrack := model.MediaFile{
				ID: "starred", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Singles",
				Annotations:  model.Annotations{Starred: true},
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			allowTitlePhase(model.MediaFiles{albumMatch, starredTrack})
			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("album-match")) // specificity now outranks the starred flag
		})

		It("prefers a more specific match over a 4-star track when PreferStarred is enabled", func() {
			conf.Server.Matcher.PreferStarred = true
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artists: []agents.Artist{{Name: "Depeche Mode"}}, Album: "Violator"},
			}
			albumMatch := model.MediaFile{
				ID: "album-match", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			ratedTrack := model.MediaFile{
				ID: "rated", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Singles",
				Annotations:  model.Annotations{Rating: 4},
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			allowTitlePhase(model.MediaFiles{albumMatch, ratedTrack})
			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("album-match")) // specificity now outranks the 4-star rating
		})

		It("prefers a starred track when specificity and overlap are equal", func() {
			conf.Server.Matcher.PreferStarred = true
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
				Annotations:  model.Annotations{Starred: true},
				Participants: artistParticipants(model.Artist{ID: "dm", Name: "Depeche Mode", OrderArtistName: "depeche mode"}),
			}
			allowTitlePhase(model.MediaFiles{plain, starred})
			result, err := m.MatchSongs(ctx, songs, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("starred")) // preferred still wins the tie
		})
	})

	Describe("duration matching", func() {
		BeforeEach(func() {
			conf.Server.Matcher.FuzzyThreshold = 100
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

			allowTitlePhase(model.MediaFiles{wrongDuration, correctMatch})

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

			allowTitlePhase(model.MediaFiles{closeDuration})

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

			allowTitlePhase(model.MediaFiles{farDuration, closeDuration})

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

			allowTitlePhase(model.MediaFiles{differentDuration})

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

			allowTitlePhase(model.MediaFiles{differentTitle, correctTitle})

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

			allowTitlePhase(model.MediaFiles{anyTrack})

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

			allowTitlePhase(model.MediaFiles{shortTrack})

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

			allowTitlePhase(model.MediaFiles{shortTrack, longTrack})

			result, err := m.MatchSongs(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("short"))
			Expect(result[1].ID).To(Equal("long"))
		})
	})

	Describe("deduplication edge cases", func() {
		BeforeEach(func() {
			conf.Server.Matcher.FuzzyThreshold = 85
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

			allowTitlePhase(model.MediaFiles{libraryTrack})

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

			allowTitlePhase(model.MediaFiles{trackA, trackB, trackC})

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

			allowTitlePhase(model.MediaFiles{trackA, trackB})

			result, err := m.MatchSongs(ctx, songs, 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("track-a"))
			Expect(result[1].ID).To(Equal("track-b"))
		})
	})
})

type mockMediaFileRepo struct {
	mock.Mock
	model.MediaFileRepository
}

func newMockMediaFileRepo() *mockMediaFileRepo {
	return &mockMediaFileRepo{}
}

func (m *mockMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	argsSlice := make([]any, len(options))
	for i, v := range options {
		argsSlice[i] = v
	}
	args := m.Called(argsSlice...)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.MediaFiles), args.Error(1)
}

func (m *mockMediaFileRepo) GetAllByTags(_ model.TagName, _ []string, options ...model.QueryOptions) (model.MediaFiles, error) {
	return m.GetAll(options...)
}

func (m *mockMediaFileRepo) SetError(hasError bool) {
	if hasError {
		m.On("GetAll", mock.Anything).Return(nil, errors.New("mock repo error"))
	}
}

type mockArtistRepo struct {
	mock.Mock
	model.ArtistRepository
}

func newMockArtistRepo() *mockArtistRepo {
	return &mockArtistRepo{}
}

func (m *mockArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	argsSlice := make([]any, len(options))
	for i, v := range options {
		argsSlice[i] = v
	}
	args := m.Called(argsSlice...)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.Artists), args.Error(1)
}

// matchFieldInAnd returns a matcher that checks whether QueryOptions.Filters is a
// squirrel.And whose first element is a squirrel.Eq containing the given field name.
func matchFieldInAnd(fieldName string) func(opt model.QueryOptions) bool {
	return func(opt model.QueryOptions) bool {
		and, ok := opt.Filters.(squirrel.And)
		if !ok || len(and) < 2 {
			return false
		}
		eq, hasEq := and[0].(squirrel.Eq)
		if !hasEq {
			return false
		}
		_, hasField := eq[fieldName]
		return hasField
	}
}

// matchFieldInEq returns a matcher that checks whether QueryOptions.Filters is a
// squirrel.Eq containing the given field name.
func matchFieldInEq(fieldName string) func(opt model.QueryOptions) bool {
	return func(opt model.QueryOptions) bool {
		eq, ok := opt.Filters.(squirrel.Eq)
		if !ok {
			return false
		}
		_, hasField := eq[fieldName]
		return hasField
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

// matchTracksByArtistQuery matches the title phase's track-fetch query, identified by its
// squirrel.And containing a squirrel.Expr whose SQL references media_file_artists.
func matchTracksByArtistQuery() func(opt model.QueryOptions) bool {
	return func(opt model.QueryOptions) bool {
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
}
