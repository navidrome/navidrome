package matcher_test

import (
	"context"
	"errors"

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
	var ctx context.Context
	var m *matcher.Matcher

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		mediaFileRepo = newMockMediaFileRepo()
		DeferCleanup(func() {
			mediaFileRepo.AssertExpectations(GinkgoT())
		})
		ds = &tests.MockDataStore{
			MockedMediaFile: mediaFileRepo,
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

	// allowOtherPhases installs .Maybe() catch-alls so phases that short-circuit (return
	// early without hitting the DB) don't cause test failures for unexpected calls. Call
	// this after expect*Phase for the phases the test actually wants to verify.
	allowOtherPhases := func() {
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("media_file.id"))).
			Return(model.MediaFiles{}, nil).Maybe()
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("mbz_recording_id"))).
			Return(model.MediaFiles{}, nil).Maybe()
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInEq("missing"))).
			Return(model.MediaFiles{}, nil).Maybe()
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("order_artist_name"))).
			Return(model.MediaFiles{}, nil).Maybe()
	}

	// setupTitleOnlyExpectations is a convenience for fuzzy-match tests that only exercise
	// the title+artist phase. The title phase uses .Maybe() because it may short-circuit
	// when no songs have an artist.
	setupTitleOnlyExpectations := func(artistTracks model.MediaFiles) {
		mediaFileRepo.On("GetAll", mock.MatchedBy(matchFieldInAnd("order_artist_name"))).
			Return(artistTracks, nil).Maybe()
	}

	Describe("MatchSongsToLibrary", func() {
		Context("matching by direct ID", func() {
			It("matches songs with an ID field to MediaFiles by ID", func() {
				conf.Server.SimilarSongsMatchThreshold = 100
				songs := []agents.Song{
					{ID: "track-1", Name: "Some Song", Artist: "Some Artist"},
				}
				idMatch := model.MediaFile{
					ID: "track-1", Title: "Some Song", Artist: "Some Artist",
				}
				expectIDPhase(model.MediaFiles{idMatch})
				allowOtherPhases()
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-1"))
			})
		})

		Context("matching by MBID", func() {
			It("matches songs with MBID to tracks with matching mbz_recording_id", func() {
				conf.Server.SimilarSongsMatchThreshold = 100
				songs := []agents.Song{
					{Name: "Paranoid Android", MBID: "abc-123", Artist: "Radiohead"},
				}
				mbidMatch := model.MediaFile{
					ID: "track-mbid", Title: "Paranoid Android", Artist: "Radiohead",
					MbzRecordingID: "abc-123",
				}
				expectMBIDPhase(model.MediaFiles{mbidMatch})
				allowOtherPhases()
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-mbid"))
			})
		})

		Context("matching by ISRC", func() {
			It("matches songs with ISRC to tracks with matching ISRC tag", func() {
				conf.Server.SimilarSongsMatchThreshold = 100
				songs := []agents.Song{
					{Name: "Paranoid Android", ISRC: "GBAYE0000351", Artist: "Radiohead"},
				}
				isrcMatch := model.MediaFile{
					ID: "track-isrc", Title: "Paranoid Android", Artist: "Radiohead",
					Tags: model.Tags{model.TagISRC: []string{"GBAYE0000351"}},
				}
				expectISRCPhase(model.MediaFiles{isrcMatch})
				allowOtherPhases()
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-isrc"))
			})
		})

		Context("fuzzy title+artist matching", func() {
			It("matches songs by title and artist name", func() {
				conf.Server.SimilarSongsMatchThreshold = 100
				songs := []agents.Song{
					{Name: "Enjoy the Silence", Artist: "Depeche Mode"},
				}
				titleMatch := model.MediaFile{
					ID: "track-title", Title: "Enjoy the Silence", Artist: "Depeche Mode",
				}
				setupTitleOnlyExpectations(model.MediaFiles{titleMatch})
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-title"))
			})

			It("matches songs with fuzzy title similarity", func() {
				conf.Server.SimilarSongsMatchThreshold = 85
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artist: "Queen"},
				}
				fuzzyMatch := model.MediaFile{
					ID: "track-fuzzy", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
				}
				setupTitleOnlyExpectations(model.MediaFiles{fuzzyMatch})
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-fuzzy"))
			})

			It("does not match completely different titles", func() {
				conf.Server.SimilarSongsMatchThreshold = 85
				songs := []agents.Song{
					{Name: "Yesterday", Artist: "The Beatles"},
				}
				differentTracks := model.MediaFiles{
					{ID: "different", Title: "Tomorrow Never Knows", Artist: "The Beatles"},
				}
				setupTitleOnlyExpectations(differentTracks)
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("deduplication", func() {
			It("removes duplicates when different input songs match the same library track", func() {
				conf.Server.SimilarSongsMatchThreshold = 85
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody (Live)", Artist: "Queen"},
					{Name: "Bohemian Rhapsody (Original Mix)", Artist: "Queen"},
				}
				libraryTrack := model.MediaFile{
					ID: "br-live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen",
				}
				setupTitleOnlyExpectations(model.MediaFiles{libraryTrack})
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("br-live"))
			})

			It("preserves duplicates when identical input songs match the same library track", func() {
				conf.Server.SimilarSongsMatchThreshold = 85
				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera"},
					{Name: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera"},
				}
				libraryTrack := model.MediaFile{
					ID: "br", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera",
				}
				setupTitleOnlyExpectations(model.MediaFiles{libraryTrack})
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].ID).To(Equal("br"))
				Expect(result[1].ID).To(Equal("br"))
			})
		})

		Context("priority ordering", func() {
			It("prefers ID match over MBID match", func() {
				conf.Server.SimilarSongsMatchThreshold = 100
				// Song has both ID and MBID set. The matcher should resolve via ID
				// and short-circuit the MBID phase entirely, so no MBID fetch should
				// occur even though an mbz_recording_id exists in the input.
				songs := []agents.Song{
					{ID: "track-id", Name: "Song", MBID: "mbid-1", Artist: "Artist"},
				}
				idMatch := model.MediaFile{
					ID: "track-id", Title: "Song", Artist: "Artist",
				}
				expectIDPhase(model.MediaFiles{idMatch})
				allowOtherPhases()
				result, err := m.MatchSongsToLibrary(ctx, songs, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("track-id"))
			})
		})

		Context("count limit", func() {
			It("returns at most 'count' results", func() {
				conf.Server.SimilarSongsMatchThreshold = 100
				songs := []agents.Song{
					{Name: "Song A", Artist: "Artist"},
					{Name: "Song B", Artist: "Artist"},
					{Name: "Song C", Artist: "Artist"},
				}
				tracks := model.MediaFiles{
					{ID: "a", Title: "Song A", Artist: "Artist"},
					{ID: "b", Title: "Song B", Artist: "Artist"},
					{ID: "c", Title: "Song C", Artist: "Artist"},
				}
				setupTitleOnlyExpectations(tracks)
				result, err := m.MatchSongsToLibrary(ctx, songs, 2)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
			})
		})

		Context("empty input", func() {
			It("returns empty results for no songs", func() {
				result, err := m.MatchSongsToLibrary(ctx, []agents.Song{}, 5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})
	})

	Describe("specificity level matching", func() {
		BeforeEach(func() {
			conf.Server.SimilarSongsMatchThreshold = 100
		})

		It("matches by title + artist MBID + album MBID (highest priority)", func() {
			correctMatch := model.MediaFile{
				ID: "correct-match", Title: "Similar Song", Artist: "Depeche Mode", Album: "Violator",
				MbzArtistID: "artist-mbid-123", MbzAlbumID: "album-mbid-456",
			}
			wrongMatch := model.MediaFile{
				ID: "wrong-match", Title: "Similar Song", Artist: "Depeche Mode", Album: "Some Other Album",
				MbzArtistID: "artist-mbid-123", MbzAlbumID: "different-album-mbid",
			}
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Depeche Mode", ArtistMBID: "artist-mbid-123", Album: "Violator", AlbumMBID: "album-mbid-456"},
			}

			setupTitleOnlyExpectations(model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("matches by title + artist name + album name when MBIDs unavailable", func() {
			correctMatch := model.MediaFile{
				ID: "correct-match", Title: "Similar Song", Artist: "depeche mode", Album: "violator",
			}
			wrongMatch := model.MediaFile{
				ID: "wrong-match", Title: "Similar Song", Artist: "Other Artist", Album: "Other Album",
			}
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Depeche Mode", Album: "Violator"},
			}

			setupTitleOnlyExpectations(model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("matches by title + artist only when album info unavailable", func() {
			correctMatch := model.MediaFile{
				ID: "correct-match", Title: "Similar Song", Artist: "depeche mode", Album: "Some Album",
			}
			wrongMatch := model.MediaFile{
				ID: "wrong-match", Title: "Similar Song", Artist: "Other Artist", Album: "Other Album",
			}
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Depeche Mode"},
			}

			setupTitleOnlyExpectations(model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-match"))
		})

		It("does not match songs without artist info", func() {
			songs := []agents.Song{
				{Name: "Similar Song"},
			}

			setupTitleOnlyExpectations(model.MediaFiles{})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("returns distinct matches for each artist's version (covers scenario)", func() {
			cover1 := model.MediaFile{ID: "cover-1", Title: "Yesterday", Artist: "The Beatles", Album: "Help!"}
			cover2 := model.MediaFile{ID: "cover-2", Title: "Yesterday", Artist: "Ray Charles", Album: "Greatest Hits"}
			cover3 := model.MediaFile{ID: "cover-3", Title: "Yesterday", Artist: "Frank Sinatra", Album: "My Way"}

			songs := []agents.Song{
				{Name: "Yesterday", Artist: "The Beatles", Album: "Help!"},
				{Name: "Yesterday", Artist: "Ray Charles", Album: "Greatest Hits"},
				{Name: "Yesterday", Artist: "Frank Sinatra", Album: "My Way"},
			}

			setupTitleOnlyExpectations(model.MediaFiles{cover1, cover2, cover3})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(3))
			ids := []string{result[0].ID, result[1].ID, result[2].ID}
			Expect(ids).To(ContainElements("cover-1", "cover-2", "cover-3"))
		})

		It("prefers more precise matches for each song", func() {
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

			songs := []agents.Song{
				{Name: "Song A", Artist: "Artist One", ArtistMBID: "mbid-1", Album: "Album One", AlbumMBID: "album-mbid-1"},
				{Name: "Song B", Artist: "Artist Two"},
			}

			setupTitleOnlyExpectations(model.MediaFiles{lessAccurateMatch, preciseMatch, artistTwoMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("precise"))
			Expect(result[1].ID).To(Equal("artist-two"))
		})
	})

	Describe("fuzzy matching thresholds", func() {
		Context("with default threshold (85%)", func() {
			It("matches songs with remastered suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				songs := []agents.Song{
					{Name: "Paranoid Android", Artist: "Radiohead"},
				}
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead"},
				}

				setupTitleOnlyExpectations(artistTracks)

				result, err := m.MatchSongsToLibrary(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("remastered"))
			})

			It("matches songs with live suffix", func() {
				conf.Server.SimilarSongsMatchThreshold = 85

				songs := []agents.Song{
					{Name: "Bohemian Rhapsody", Artist: "Queen"},
				}
				artistTracks := model.MediaFiles{
					{ID: "live", Title: "Bohemian Rhapsody (Live)", Artist: "Queen"},
				}

				setupTitleOnlyExpectations(artistTracks)

				result, err := m.MatchSongsToLibrary(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("live"))
			})
		})

		Context("with threshold set to 100 (exact match only)", func() {
			It("only matches exact titles", func() {
				conf.Server.SimilarSongsMatchThreshold = 100

				songs := []agents.Song{
					{Name: "Paranoid Android", Artist: "Radiohead"},
				}
				artistTracks := model.MediaFiles{
					{ID: "remastered", Title: "Paranoid Android - Remastered", Artist: "Radiohead"},
				}

				setupTitleOnlyExpectations(artistTracks)

				result, err := m.MatchSongsToLibrary(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})

		Context("with lower threshold (75%)", func() {
			It("matches more aggressively", func() {
				conf.Server.SimilarSongsMatchThreshold = 75

				songs := []agents.Song{
					{Name: "Song", Artist: "Artist"},
				}
				artistTracks := model.MediaFiles{
					{ID: "extended", Title: "Song (Extended Mix)", Artist: "Artist"},
				}

				setupTitleOnlyExpectations(artistTracks)

				result, err := m.MatchSongsToLibrary(ctx, songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal("extended"))
			})
		})
	})

	Describe("fuzzy album matching", func() {
		BeforeEach(func() {
			conf.Server.SimilarSongsMatchThreshold = 85
		})

		It("matches album with (Remaster) suffix", func() {
			songs := []agents.Song{
				{Name: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera"},
			}
			correctMatch := model.MediaFile{
				ID: "correct", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "A Night at the Opera (2011 Remaster)",
			}
			wrongMatch := model.MediaFile{
				ID: "wrong", Title: "Bohemian Rhapsody", Artist: "Queen", Album: "Greatest Hits",
			}

			setupTitleOnlyExpectations(model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct"))
		})

		It("matches album with (Deluxe Edition) suffix", func() {
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"},
			}
			correctMatch := model.MediaFile{
				ID: "correct", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator (Deluxe Edition)",
			}
			wrongMatch := model.MediaFile{
				ID: "wrong", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "101",
			}

			setupTitleOnlyExpectations(model.MediaFiles{wrongMatch, correctMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct"))
		})

		It("prefers exact album match over fuzzy album match", func() {
			songs := []agents.Song{
				{Name: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator"},
			}
			exactMatch := model.MediaFile{
				ID: "exact", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator",
			}
			fuzzyMatch := model.MediaFile{
				ID: "fuzzy", Title: "Enjoy the Silence", Artist: "Depeche Mode", Album: "Violator (Deluxe Edition)",
			}

			setupTitleOnlyExpectations(model.MediaFiles{fuzzyMatch, exactMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("exact"))
		})
	})

	Describe("duration matching", func() {
		BeforeEach(func() {
			conf.Server.SimilarSongsMatchThreshold = 100
		})

		It("prefers tracks with matching duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Test Artist", Duration: 180000},
			}
			correctMatch := model.MediaFile{
				ID: "correct", Title: "Similar Song", Artist: "Test Artist", Duration: 180.0,
			}
			wrongDuration := model.MediaFile{
				ID: "wrong", Title: "Similar Song", Artist: "Test Artist", Duration: 240.0,
			}

			setupTitleOnlyExpectations(model.MediaFiles{wrongDuration, correctMatch})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct"))
		})

		It("matches tracks with close duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Test Artist", Duration: 180000},
			}
			closeDuration := model.MediaFile{
				ID: "close-duration", Title: "Similar Song", Artist: "Test Artist", Duration: 182.5,
			}

			setupTitleOnlyExpectations(model.MediaFiles{closeDuration})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("close-duration"))
		})

		It("prefers closer duration over farther duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Test Artist", Duration: 180000},
			}
			closeDuration := model.MediaFile{
				ID: "close", Title: "Similar Song", Artist: "Test Artist", Duration: 181.0,
			}
			farDuration := model.MediaFile{
				ID: "far", Title: "Similar Song", Artist: "Test Artist", Duration: 190.0,
			}

			setupTitleOnlyExpectations(model.MediaFiles{farDuration, closeDuration})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("close"))
		})

		It("still matches when no tracks have matching duration", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Test Artist", Duration: 180000},
			}
			differentDuration := model.MediaFile{
				ID: "different", Title: "Similar Song", Artist: "Test Artist", Duration: 300.0,
			}

			setupTitleOnlyExpectations(model.MediaFiles{differentDuration})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("different"))
		})

		It("prefers title match over duration match when titles differ", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Test Artist", Duration: 180000},
			}
			differentTitle := model.MediaFile{
				ID: "wrong-title", Title: "Different Song", Artist: "Test Artist", Duration: 180.0,
			}
			correctTitle := model.MediaFile{
				ID: "correct-title", Title: "Similar Song", Artist: "Test Artist", Duration: 300.0,
			}

			setupTitleOnlyExpectations(model.MediaFiles{differentTitle, correctTitle})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("correct-title"))
		})

		It("matches without duration filtering when agent duration is 0", func() {
			songs := []agents.Song{
				{Name: "Similar Song", Artist: "Test Artist", Duration: 0},
			}
			anyTrack := model.MediaFile{
				ID: "any", Title: "Similar Song", Artist: "Test Artist", Duration: 999.0,
			}

			setupTitleOnlyExpectations(model.MediaFiles{anyTrack})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("any"))
		})

		It("handles very short songs with close duration", func() {
			songs := []agents.Song{
				{Name: "Short Song", Artist: "Test Artist", Duration: 30000},
			}
			shortTrack := model.MediaFile{
				ID: "short", Title: "Short Song", Artist: "Test Artist", Duration: 31.0,
			}

			setupTitleOnlyExpectations(model.MediaFiles{shortTrack})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal("short"))
		})
	})

	Describe("deduplication edge cases", func() {
		BeforeEach(func() {
			conf.Server.SimilarSongsMatchThreshold = 85
		})

		It("handles mixed scenario with both identical and different input songs", func() {
			songs := []agents.Song{
				{Name: "Yesterday", Artist: "The Beatles", Album: "Help!"},
				{Name: "Yesterday (Remastered)", Artist: "The Beatles", Album: "1"},
				{Name: "Yesterday", Artist: "The Beatles", Album: "Help!"},
				{Name: "Yesterday (Anthology)", Artist: "The Beatles", Album: "Anthology"},
			}
			libraryTrack := model.MediaFile{
				ID: "yesterday", Title: "Yesterday", Artist: "The Beatles", Album: "Help!",
			}

			setupTitleOnlyExpectations(model.MediaFiles{libraryTrack})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].ID).To(Equal("yesterday"))
			Expect(result[1].ID).To(Equal("yesterday"))
		})

		It("does not deduplicate songs that match different library tracks", func() {
			songs := []agents.Song{
				{Name: "Song A", Artist: "Artist"},
				{Name: "Song B", Artist: "Artist"},
				{Name: "Song C", Artist: "Artist"},
			}
			trackA := model.MediaFile{ID: "track-a", Title: "Song A", Artist: "Artist"}
			trackB := model.MediaFile{ID: "track-b", Title: "Song B", Artist: "Artist"}
			trackC := model.MediaFile{ID: "track-c", Title: "Song C", Artist: "Artist"}

			setupTitleOnlyExpectations(model.MediaFiles{trackA, trackB, trackC})

			result, err := m.MatchSongsToLibrary(ctx, songs, 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0].ID).To(Equal("track-a"))
			Expect(result[1].ID).To(Equal("track-b"))
			Expect(result[2].ID).To(Equal("track-c"))
		})

		It("respects count limit after deduplication", func() {
			songs := []agents.Song{
				{Name: "Song A", Artist: "Artist"},
				{Name: "Song A (Live)", Artist: "Artist"},
				{Name: "Song B", Artist: "Artist"},
				{Name: "Song B (Remix)", Artist: "Artist"},
			}
			trackA := model.MediaFile{ID: "track-a", Title: "Song A", Artist: "Artist"}
			trackB := model.MediaFile{ID: "track-b", Title: "Song B", Artist: "Artist"}

			setupTitleOnlyExpectations(model.MediaFiles{trackA, trackB})

			result, err := m.MatchSongsToLibrary(ctx, songs, 2)

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
