package extdata

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
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
	var agentsCombined Agents
	var artistRepo *mockSimArtistRepo
	var mediaFileRepo *mockSimMediaFileRepo
	var ctx context.Context
	var originalAgentsConfig string

	BeforeEach(func() {
		ctx = context.Background()

		// Store the original agents config to restore it later
		originalAgentsConfig = conf.Server.Agents

		// Setup mocks - Initialize here, but set expectations within each test
		artistRepo = newMockSimArtistRepo()
		mediaFileRepo = newMockSimMediaFileRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
		}

		// Clear the agents map to prevent interference from previous tests
		agents.Map = nil

		// Create a mock agent that implements both required interfaces
		// Re-initialize mockAgent in each test if necessary, or ensure Calls are cleared
		mockAgent = &mockSimilarArtistAgent{}
		mockTopAgent = mockAgent
		mockSimilarAgent = mockAgent

		// Create a mock for the combined Agents interface
		agentsCombined = &mockCombinedAgents{
			topSongsAgent: mockTopAgent,
			similarAgent:  mockSimilarAgent,
		}

		// Create the provider instance with our mock Agents implementation
		provider = NewProvider(ds, agentsCombined)
	})

	AfterEach(func() {
		// Restore original config
		conf.Server.Agents = originalAgentsConfig
	})

	Describe("SimilarSongs", func() {
		It("returns similar songs from main artist and similar artists", func() {
			// --- Test-specific setup ---
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			similarArtist := model.Artist{ID: "artist-3", Name: "Similar Artist"}
			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1"}
			song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1"}
			song3 := model.MediaFile{ID: "song-3", Title: "Song Three", ArtistID: "artist-3"}

			// Configure the Get method (needed for GetEntityByID in getArtist)
			artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
			artistRepo.On("Get", "artist-3").Return(&similarArtist, nil).Maybe() // For similar artist lookup if needed

			// Configure the GetAll mock for finding the main artist in getArtist
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 1 && opt.Filters != nil
			})).Return(model.Artists{artist1}, nil).Once()

			// Setup similar artists response from agent
			similarAgentsResp := []agents.Artist{
				{Name: "Similar Artist", MBID: "similar-mbid"},
			}
			mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
				Return(similarAgentsResp, nil).Once()

			// Setup for mapping similar artists in mapSimilarArtists
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				// This will match the query for similar artists (no Max limit set by caller)
				return opt.Max == 0 && opt.Filters != nil
			})).Return(model.Artists{similarArtist}, nil).Once()

			// Setup Top Songs responses
			// Main artist songs
			mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
				Return([]agents.Song{
					{Name: "Song One", MBID: "mbid-1"},
					{Name: "Song Two", MBID: "mbid-2"},
				}, nil).Once()

			// Similar artist songs
			mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-3", "Similar Artist", "", mock.Anything).
				Return([]agents.Song{
					{Name: "Song Three", MBID: "mbid-3"},
				}, nil).Once()

			// Setup mediafile repository to find songs by MBID (via GetAll)
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)
			mediaFileRepo.FindByMBID("mbid-3", song3)
			// --- End Test-specific setup ---

			songs, err := provider.SimilarSongs(ctx, "artist-1", 3)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			for _, song := range songs {
				Expect(song.ID).To(BeElementOf("song-1", "song-2", "song-3"))
			}
		})

		It("returns nil when artist is not found", func() {
			// --- Test-specific setup ---
			// Use prefixed ID for GetEntityByID
			artistRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)
			mediaFileRepo.On("Get", "artist-unknown-artist").Return(nil, model.ErrNotFound)

			// Setup for getArtist fallback to GetAll
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 1 && opt.Filters != nil
			})).Return(model.Artists{}, nil).Maybe()
			// --- End Test-specific setup ---

			// Use prefixed ID
			songs, err := provider.SimilarSongs(ctx, "artist-unknown-artist", 5)

			Expect(err).To(Equal(model.ErrNotFound))
			Expect(songs).To(BeNil())
		})

		It("returns songs from main artist when GetSimilarArtists returns error", func() {
			// --- Test-specific setup ---
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1"}

			// Configure artist repo Get method for getArtist
			artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
			// Configure GetAll fallback for getArtist
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 1 && opt.Filters != nil
			})).Return(model.Artists{artist1}, nil).Maybe() // Maybe because Get should find it

			// Set the error for GetSimilarArtists
			mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
				Return(nil, errors.New("error getting similar artists")).Once()

			// Expect call to mapSimilarArtists -> artistRepo.GetAll (returns empty)
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 0 && opt.Filters != nil // Matcher for mapSimilarArtists
			})).Return(model.Artists{}, nil).Once()

			// Setup for main artist's top songs
			mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
				Return([]agents.Song{
					{Name: "Song One", MBID: "mbid-1"},
				}, nil).Once()

			// Setup mediafile repo for finding the song
			mediaFileRepo.FindByMBID("mbid-1", song1)
			// --- End Test-specific setup ---

			songs, err := provider.SimilarSongs(ctx, "artist-1", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].ID).To(Equal("song-1"))
		})

		It("returns empty list when GetArtistTopSongs returns error", func() {
			// --- Test-specific setup ---
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}

			// Configure artist repo Get method for getArtist
			artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
			// Configure GetAll fallback for getArtist
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 1 && opt.Filters != nil
			})).Return(model.Artists{artist1}, nil).Maybe() // Maybe because Get should find it

			// Expect GetSimilarArtists call (returns empty)
			mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
				Return([]agents.Artist{}, nil).Once()

			// Expect call to mapSimilarArtists -> artistRepo.GetAll (returns empty)
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 0 && opt.Filters != nil // Matcher for mapSimilarArtists
			})).Return(model.Artists{}, nil).Once()

			// Set error for GetArtistTopSongs (for the main artist)
			mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
				Return(nil, errors.New("error getting top songs")).Once()
			// --- End Test-specific setup ---

			songs, err := provider.SimilarSongs(ctx, "artist-1", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(BeEmpty())
		})

		It("respects count parameter", func() {
			// --- Test-specific setup ---
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			song1 := model.MediaFile{ID: "song-1", Title: "Song One", ArtistID: "artist-1"}
			song2 := model.MediaFile{ID: "song-2", Title: "Song Two", ArtistID: "artist-1"}

			// Configure artist repo Get method for getArtist
			artistRepo.On("Get", "artist-1").Return(&artist1, nil).Maybe()
			// Configure GetAll fallback for getArtist
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 1 && opt.Filters != nil
			})).Return(model.Artists{artist1}, nil).Maybe() // Maybe because Get should find it

			// Expect GetSimilarArtists call (returns empty)
			mockAgent.On("GetSimilarArtists", mock.Anything, "artist-1", "Artist One", "", 15).
				Return([]agents.Artist{}, nil).Once()

			// Expect call to mapSimilarArtists -> artistRepo.GetAll (returns empty)
			artistRepo.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
				return opt.Max == 0 && opt.Filters != nil // Matcher for mapSimilarArtists
			})).Return(model.Artists{}, nil).Once()

			// Setup for main artist's top songs
			mockAgent.On("GetArtistTopSongs", mock.Anything, "artist-1", "Artist One", "", mock.Anything).
				Return([]agents.Song{
					{Name: "Song One", MBID: "mbid-1"},
					{Name: "Song Two", MBID: "mbid-2"},
				}, nil).Once()

			// Setup mediafile repo for finding the songs
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)
			// --- End Test-specific setup ---

			songs, err := provider.SimilarSongs(ctx, "artist-1", 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].ID).To(BeElementOf("song-1", "song-2"))
		})
	})
})

// Combined implementation of both ArtistTopSongsRetriever and ArtistSimilarRetriever interfaces
type mockSimilarArtistAgent struct {
	mock.Mock
	agents.Interface
}

func (m *mockSimilarArtistAgent) AgentName() string {
	return "mock"
}

func (m *mockSimilarArtistAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	args := m.Called(ctx, id, artistName, mbid, count)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Song), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSimilarArtistAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	args := m.Called(ctx, id, name, mbid, limit)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Artist), args.Error(1)
	}
	return nil, args.Error(1)
}

// A simple implementation of the Agents interface that combines separate implementations
type mockCombinedAgents struct {
	topSongsAgent agents.ArtistTopSongsRetriever
	similarAgent  agents.ArtistSimilarRetriever
}

func (m *mockCombinedAgents) AgentName() string {
	return "mockCombined"
}

func (m *mockCombinedAgents) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	return "", nil
}

func (m *mockCombinedAgents) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	return "", nil
}

func (m *mockCombinedAgents) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	return "", nil
}

func (m *mockCombinedAgents) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	if m.similarAgent != nil {
		return m.similarAgent.GetSimilarArtists(ctx, id, name, mbid, limit)
	}
	return nil, agents.ErrNotFound
}

func (m *mockCombinedAgents) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	return nil, agents.ErrNotFound
}

func (m *mockCombinedAgents) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	if m.topSongsAgent != nil {
		return m.topSongsAgent.GetArtistTopSongs(ctx, id, artistName, mbid, count)
	}
	return nil, agents.ErrNotFound
}

func (m *mockCombinedAgents) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	return nil, agents.ErrNotFound
}

// Mocked ArtistRepo for similar songs tests
type mockSimArtistRepo struct {
	mock.Mock
	model.ArtistRepository
}

func newMockSimArtistRepo() *mockSimArtistRepo {
	return &mockSimArtistRepo{}
}

func (m *mockSimArtistRepo) SetData(artists model.Artists) {
	// Store the data for Get queries
	for _, a := range artists {
		// Capture the loop variable by value
		artistCopy := a
		// Revert: Get does not take context
		m.On("Get", artistCopy.ID).Return(&artistCopy, nil)
	}
}

// Revert: Get does not take context
func (m *mockSimArtistRepo) Get(id string) (*model.Artist, error) {
	// Revert: Remove context from Called
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Artist), args.Error(1)
}

func (m *mockSimArtistRepo) SetError(hasError bool) {
	if hasError {
		// Revert: Remove context from GetAll mock setup
		m.On("GetAll", mock.Anything).Return(nil, errors.New("error"))
	}
}

func (m *mockSimArtistRepo) FindByName(name string, artist model.Artist) {
	// Set up a mock for finding an artist by name with LIKE filter, using Anything matcher for flexibility
	// Revert: Remove context from GetAll mock setup
	m.On("GetAll", mock.Anything).Return(model.Artists{artist}, nil).Once()
}

// Revert: Remove context from GetAll signature
func (m *mockSimArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	// Pass options correctly to Called
	// Convert options slice to []interface{} for Called
	argsSlice := make([]interface{}, len(options))
	for i, v := range options {
		argsSlice[i] = v
	}
	args := m.Called(argsSlice...)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.Artists), args.Error(1)
}

// Mocked MediaFileRepo for similar songs tests
type mockSimMediaFileRepo struct {
	mock.Mock
	model.MediaFileRepository
}

func newMockSimMediaFileRepo() *mockSimMediaFileRepo {
	return &mockSimMediaFileRepo{}
}

func (m *mockSimMediaFileRepo) SetData(mediaFiles model.MediaFiles) {
	// Store the data for Get queries
	for _, mf := range mediaFiles {
		mfCopy := mf // Capture loop variable
		// Revert: Get does not take context
		m.On("Get", mfCopy.ID).Return(&mfCopy, nil)
	}
}

// Revert: Get does not take context
func (m *mockSimMediaFileRepo) Get(id string) (*model.MediaFile, error) {
	// Revert: Remove context from Called
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MediaFile), args.Error(1)
}

func (m *mockSimMediaFileRepo) SetError(hasError bool) {
	if hasError {
		// Revert: Remove context from GetAll mock setup
		m.On("GetAll", mock.Anything).Return(nil, errors.New("error"))
	}
}

func (m *mockSimMediaFileRepo) FindByMBID(mbid string, mediaFile model.MediaFile) {
	// Set up a mock for finding a media file by MBID using Anything matcher for flexibility
	// Revert: Remove context from GetAll mock setup
	m.On("GetAll", mock.Anything).Return(model.MediaFiles{mediaFile}, nil).Once()
}

func (m *mockSimMediaFileRepo) FindByArtistAndTitle(artistID string, title string, mediaFile model.MediaFile) {
	// Set up a mock for finding a media file by artist ID and title
	// Revert: Remove context from GetAll mock setup
	m.On("GetAll", mock.Anything).Return(model.MediaFiles{mediaFile}, nil).Once()
}

// Revert: Remove context from GetAll signature
func (m *mockSimMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	// Pass options correctly to Called
	// Convert options slice to []interface{} for Called
	argsSlice := make([]interface{}, len(options))
	for i, v := range options {
		argsSlice[i] = v
	}
	args := m.Called(argsSlice...)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.MediaFiles), args.Error(1)
}
