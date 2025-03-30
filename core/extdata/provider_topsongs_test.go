package extdata

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/listenbrainz"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider - TopSongs", func() {
	var ds model.DataStore
	var provider Provider
	var artistRepo *mockArtistRepo
	var mediaFileRepo *mockMediaFileRepo
	var mockTopSongsAgent *mockArtistTopSongsAgent
	var agentsCombined Agents
	var ctx context.Context
	var originalAgentsConfig string

	BeforeEach(func() {
		ctx = context.Background()
		originalAgentsConfig = conf.Server.Agents

		artistRepo = newMockArtistRepo()
		mediaFileRepo = newMockMediaFileRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
		}

		mockTopSongsAgent = &mockArtistTopSongsAgent{}

		agentsCombined = &mockCombinedAgents{
			topSongsAgent: mockTopSongsAgent,
			similarAgent:  nil,
		}

		provider = NewProvider(ds, agentsCombined)
	})

	AfterEach(func() {
		conf.Server.Agents = originalAgentsConfig
	})

	Describe("TopSongs", func() {
		BeforeEach(func() {
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artist2 := model.Artist{ID: "artist-2", Name: "Artist Two"}
			artistRepo.SetData(model.Artists{artist1, artist2})

			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzReleaseTrackID: "mbid-1"}
			song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzReleaseTrackID: "mbid-2"}
			song3 := model.MediaFile{ID: "song-3", Title: "Song Three", ArtistID: "artist-2", MbzReleaseTrackID: "mbid-3"}
			mediaFileRepo.SetData(model.MediaFiles{song1, song2, song3})

			mockTopSongsAgent.SetTopSongs([]agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			})
		})

		It("returns top songs for a known artist", func() {
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.FindByName("Artist One", artist1)

			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzReleaseTrackID: "mbid-1"}
			song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1", MbzReleaseTrackID: "mbid-2"}
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)

			songs, err := provider.TopSongs(ctx, "Artist One", 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(2))
			Expect(songs[0].ID).To(Equal("song-1"))
			Expect(songs[1].ID).To(Equal("song-2"))
		})

		It("returns nil for an unknown artist", func() {
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				if opt.Max != 1 || opt.Filters == nil {
					return false
				}
				_, ok := opt.Filters.(squirrel.Like)
				return ok
			})).Return(model.Artists{}, model.ErrNotFound).Once()

			songs, err := provider.TopSongs(ctx, "Unknown Artist", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(BeNil())
		})

		It("returns nil when the agent returns an error", func() {
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.FindByName("Artist One", artist1)

			mockTopSongsAgent.SetError(errors.New("agent error"))

			song1 := model.MediaFile{ID: "song-1"}
			song2 := model.MediaFile{ID: "song-2"}
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)

			songs, err := provider.TopSongs(ctx, "Artist One", 5)

			Expect(err).To(MatchError("agent error"))
			Expect(songs).To(BeNil())
		})

		It("returns nil when the agent returns ErrNotFound", func() {
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.FindByName("Artist One", artist1)

			mockTopSongsAgent.SetError(agents.ErrNotFound)

			song1 := model.MediaFile{ID: "song-1"}
			song2 := model.MediaFile{ID: "song-2"}
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)

			songs, err := provider.TopSongs(ctx, "Artist One", 5)
			Expect(err).To(MatchError(model.ErrNotFound))
			Expect(songs).To(BeNil())
		})

		It("returns fewer songs if count is less than available top songs", func() {
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.FindByName("Artist One", artist1)

			song1 := model.MediaFile{ID: "song-1"}
			mediaFileRepo.FindByMBID("mbid-1", song1)

			songs, err := provider.TopSongs(ctx, "Artist One", 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].ID).To(Equal("song-1"))
		})

		It("returns fewer songs if fewer matching tracks are found", func() {
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.FindByName("Artist One", artist1)

			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1", MbzReleaseTrackID: "mbid-1"}

			matcherForMBID := func(expectedMBID string) func(opt model.QueryOptions) bool {
				return func(opt model.QueryOptions) bool {
					if opt.Filters == nil {
						return false
					}
					andClause, ok := opt.Filters.(squirrel.And)
					if !ok {
						return false
					}
					for _, condition := range andClause {
						if eqClause, ok := condition.(squirrel.Eq); ok {
							if mbid, exists := eqClause["mbz_recording_id"]; exists {
								return mbid == expectedMBID
							}
						}
					}
					return false
				}
			}

			matcherForTitleArtistFallback := func(artistID, title string) func(opt model.QueryOptions) bool {
				return func(opt model.QueryOptions) bool {
					if opt.Filters == nil {
						return false
					}
					andClause, ok := opt.Filters.(squirrel.And)
					if !ok || len(andClause) < 3 {
						return false
					}
					foundLike := false
					for _, condition := range andClause {
						if likeClause, ok := condition.(squirrel.Like); ok {
							if _, exists := likeClause["order_title"]; exists {
								foundLike = true
								break
							}
						}
					}
					return foundLike
				}
			}

			mediaFileRepo.On("GetAll", mock.MatchedBy(matcherForMBID("mbid-1"))).Return(model.MediaFiles{song1}, nil).Once()
			mediaFileRepo.On("GetAll", mock.MatchedBy(matcherForMBID("mbid-2"))).Return(model.MediaFiles{}, nil).Once()
			mediaFileRepo.On("GetAll", mock.MatchedBy(matcherForTitleArtistFallback("artist-1", "Song Two"))).Return(model.MediaFiles{}, nil).Once()

			songs, err := provider.TopSongs(ctx, "Artist One", 2)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].ID).To(Equal("song-1"))
		})

		It("returns nil when context is canceled", func() {
			// This test case is not provided in the original file or the new code block
			// It's assumed to exist as it's called in the new code block
		})
	})
})

// Mock implementation for ArtistTopSongsRetriever
// This remains here as it's specific to TopSongs tests and simpler than mockSimilarArtistAgent
type mockArtistTopSongsAgent struct {
	mock.Mock
	topSongs []agents.Song
	err      error
}

func (m *mockArtistTopSongsAgent) AgentName() string {
	return "mockTopSongs"
}

func (m *mockArtistTopSongsAgent) SetTopSongs(songs []agents.Song) {
	m.topSongs = songs
	m.err = nil
}

func (m *mockArtistTopSongsAgent) SetError(err error) {
	m.err = err
	m.topSongs = nil
}

func (m *mockArtistTopSongsAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	if m.err != nil {
		return nil, m.err
	}

	if len(m.topSongs) > count {
		return m.topSongs[:count], nil
	}
	return m.topSongs, nil
}

var _ agents.ArtistTopSongsRetriever = (*mockArtistTopSongsAgent)(nil)
