package matcher_test

import (
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
	var ctx = GinkgoT().Context
	var m *matcher.Matcher

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		mediaFileRepo = newMockMediaFileRepo()
		ds = &tests.MockDataStore{
			MockedMediaFile: mediaFileRepo,
		}
		m = matcher.New(ds)
	})

	// Helper to set up expectations for title+artist queries only (no ID/MBID/ISRC)
	setupTitleOnlyExpectations := func(artistTracks model.MediaFiles) {
		// loadTracksByTitleAndArtist - queries by artist name
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

	// Helper to set up expectations for all four matching phases
	setupAllPhaseExpectations := func(idMatches, mbidMatches, isrcMatches, artistTracks model.MediaFiles) {
		// loadTracksByID - queries by media_file.id
		mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			and, ok := opt.Filters.(squirrel.And)
			if !ok || len(and) < 2 {
				return false
			}
			eq, hasEq := and[0].(squirrel.Eq)
			if !hasEq {
				return false
			}
			_, hasID := eq["media_file.id"]
			return hasID
		})).Return(idMatches, nil).Once()

		// loadTracksByMBID - queries by mbz_recording_id
		mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			and, ok := opt.Filters.(squirrel.And)
			if !ok || len(and) < 2 {
				return false
			}
			eq, hasEq := and[0].(squirrel.Eq)
			if !hasEq {
				return false
			}
			_, hasMBID := eq["mbz_recording_id"]
			return hasMBID
		})).Return(mbidMatches, nil).Once()

		// loadTracksByISRC - queries by missing:false (via GetAllByTags -> GetAll)
		mediaFileRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
			eq, ok := opt.Filters.(squirrel.Eq)
			if !ok {
				return false
			}
			_, hasMissing := eq["missing"]
			return hasMissing
		})).Return(isrcMatches, nil).Once()

		// loadTracksByTitleAndArtist - queries by order_artist_name
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

				setupAllPhaseExpectations(
					model.MediaFiles{idMatch},
					model.MediaFiles{},
					model.MediaFiles{},
					model.MediaFiles{},
				)

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				setupAllPhaseExpectations(
					model.MediaFiles{},
					model.MediaFiles{mbidMatch},
					model.MediaFiles{},
					model.MediaFiles{},
				)

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				setupAllPhaseExpectations(
					model.MediaFiles{},
					model.MediaFiles{},
					model.MediaFiles{isrcMatch},
					model.MediaFiles{},
				)

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].ID).To(Equal("br"))
				Expect(result[1].ID).To(Equal("br"))
			})
		})

		Context("priority ordering", func() {
			It("prefers ID match over MBID match", func() {
				conf.Server.SimilarSongsMatchThreshold = 100

				songs := []agents.Song{
					{ID: "track-id", Name: "Song", MBID: "mbid-1", Artist: "Artist"},
				}
				idMatch := model.MediaFile{
					ID: "track-id", Title: "Song", Artist: "Artist",
				}
				mbidMatch := model.MediaFile{
					ID: "track-mbid", Title: "Song", Artist: "Artist",
					MbzRecordingID: "mbid-1",
				}

				setupAllPhaseExpectations(
					model.MediaFiles{idMatch},
					model.MediaFiles{mbidMatch},
					model.MediaFiles{},
					model.MediaFiles{},
				)

				result, err := m.MatchSongsToLibrary(ctx(), songs, 5)

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

				result, err := m.MatchSongsToLibrary(ctx(), songs, 2)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(2))
			})
		})

		Context("empty input", func() {
			It("returns empty results for no songs", func() {
				result, err := m.MatchSongsToLibrary(ctx(), []agents.Song{}, 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeEmpty())
			})
		})
	})
})

// mockMediaFileRepo mocks model.MediaFileRepository for matcher tests
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
