package extdata

import (
	"context"
	"errors"
	"reflect"
	"unsafe"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/listenbrainz"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Provider", func() {
	var ds model.DataStore
	var em Provider
	var mockAgent *mockArtistTopSongsAgent
	var artistRepo *mockArtistRepo
	var mediaFileRepo *mockMediaFileRepo
	var ctx context.Context
	var originalAgentsConfig string

	BeforeEach(func() {
		ctx = context.Background()

		// Store the original agents config to restore it later
		originalAgentsConfig = conf.Server.Agents

		// Setup mocks
		artistRepo = newMockArtistRepo()
		mediaFileRepo = newMockMediaFileRepo()

		ds = &tests.MockDataStore{
			MockedArtist:    artistRepo,
			MockedMediaFile: mediaFileRepo,
		}

		// Clear the agents map to prevent interference from previous tests
		agents.Map = nil

		// Create a mock agent
		mockAgent = &mockArtistTopSongsAgent{}
		log.Debug(ctx, "Creating mock agent", "agent", mockAgent)
	})

	AfterEach(func() {
		// Restore original config
		conf.Server.Agents = originalAgentsConfig
	})

	Describe("TopSongs with direct agent injection", func() {
		BeforeEach(func() {
			// Set up test data
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artist2 := model.Artist{ID: "artist-2", Name: "Artist Two"}

			song1 := model.MediaFile{
				ID:                "song-1",
				Title:             "Song One",
				Artist:            "Artist One",
				ArtistID:          "artist-1",
				AlbumArtistID:     "artist-1",
				MbzReleaseTrackID: "mbid-1",
				Missing:           false,
			}

			song2 := model.MediaFile{
				ID:                "song-2",
				Title:             "Song Two",
				Artist:            "Artist One",
				ArtistID:          "artist-1",
				AlbumArtistID:     "artist-1",
				MbzReleaseTrackID: "mbid-2",
				Missing:           false,
			}

			song3 := model.MediaFile{
				ID:                "song-3",
				Title:             "Song Three",
				Artist:            "Artist Two",
				ArtistID:          "artist-2",
				AlbumArtistID:     "artist-2",
				MbzReleaseTrackID: "mbid-3",
				Missing:           false,
			}

			// Set up basic data for the repos
			artistRepo.SetData(model.Artists{artist1, artist2})
			mediaFileRepo.SetData(model.MediaFiles{song1, song2, song3})

			// Set up the specific mock responses needed for the TopSongs method
			artistRepo.FindByName("Artist One", artist1)
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)

			// Setup default behavior for empty searches
			mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{}, nil).Maybe()

			// Configure the mockAgent to return some top songs
			mockAgent.topSongs = []agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}

			// Create a custom agents instance directly with our mock agent
			agentsImpl := &agents.Agents{}

			// Use reflection to set the unexported fields
			setAgentField(agentsImpl, "ds", ds)
			setAgentField(agentsImpl, "agents", []agents.Interface{mockAgent})

			// Create the provider instance with our custom Agents implementation
			em = NewExternalMetadata(ds, agentsImpl)
		})

		It("returns matching songs from the agent results", func() {
			songs, err := em.TopSongs(ctx, "Artist One", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(2))
			Expect(songs[0].ID).To(Equal("song-1"))
			Expect(songs[1].ID).To(Equal("song-2"))

			// Verify the agent was called with the right parameters
			Expect(mockAgent.lastArtistID).To(Equal("artist-1"))
			Expect(mockAgent.lastArtistName).To(Equal("Artist One"))
			Expect(mockAgent.lastCount).To(Equal(5))
		})

		It("returns nil when artist is not found", func() {
			// Clear previous expectations
			artistRepo = newMockArtistRepo()
			mediaFileRepo = newMockMediaFileRepo()

			ds = &tests.MockDataStore{
				MockedArtist:    artistRepo,
				MockedMediaFile: mediaFileRepo,
			}

			// Create a custom agents instance directly with our mock agent
			agentsImpl := &agents.Agents{}
			setAgentField(agentsImpl, "ds", ds)
			setAgentField(agentsImpl, "agents", []agents.Interface{mockAgent})
			em = NewExternalMetadata(ds, agentsImpl)

			// Set up for artist not found scenario - return empty list
			artistRepo.On("GetAll", mock.Anything).Return(model.Artists{}, nil).Once()

			songs, err := em.TopSongs(ctx, "Unknown Artist", 5)

			Expect(err).To(BeNil())
			Expect(songs).To(BeNil())
		})

		It("returns empty list when no matching songs are found", func() {
			// Clear previous expectations
			artistRepo = newMockArtistRepo()
			mediaFileRepo = newMockMediaFileRepo()

			ds = &tests.MockDataStore{
				MockedArtist:    artistRepo,
				MockedMediaFile: mediaFileRepo,
			}

			// Create a custom agents instance directly with our mock agent
			agentsImpl := &agents.Agents{}
			setAgentField(agentsImpl, "ds", ds)
			setAgentField(agentsImpl, "agents", []agents.Interface{mockAgent})
			em = NewExternalMetadata(ds, agentsImpl)

			// Set up artist data
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.SetData(model.Artists{artist1})
			artistRepo.FindByName("Artist One", artist1)

			// Configure the agent to return songs that don't match our repo
			mockAgent.topSongs = []agents.Song{
				{Name: "Nonexistent Song", MBID: "unknown-mbid"},
			}

			// Default to empty response for any queries
			mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{}, nil).Maybe()

			songs, err := em.TopSongs(ctx, "Artist One", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(0))
		})

		It("returns nil when agent returns errors", func() {
			// New set of mocks for this test
			artistRepo = newMockArtistRepo()
			mediaFileRepo = newMockMediaFileRepo()

			ds = &tests.MockDataStore{
				MockedArtist:    artistRepo,
				MockedMediaFile: mediaFileRepo,
			}

			// Set up artist data
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			artistRepo.SetData(model.Artists{artist1})
			artistRepo.FindByName("Artist One", artist1)

			// Create a custom agents instance directly with our mock agent
			agentsImpl := &agents.Agents{}
			setAgentField(agentsImpl, "ds", ds)
			setAgentField(agentsImpl, "agents", []agents.Interface{mockAgent})
			em = NewExternalMetadata(ds, agentsImpl)

			// Set the error
			testError := errors.New("some agent error")
			mockAgent.err = testError

			songs, err := em.TopSongs(ctx, "Artist One", 5)

			// Current behavior returns nil for both error and songs
			Expect(err).To(BeNil())
			Expect(songs).To(BeNil())
		})

		It("respects count parameter", func() {
			// New set of mocks for this test
			artistRepo = newMockArtistRepo()
			mediaFileRepo = newMockMediaFileRepo()

			ds = &tests.MockDataStore{
				MockedArtist:    artistRepo,
				MockedMediaFile: mediaFileRepo,
			}

			// Set up test data
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}
			song1 := model.MediaFile{
				ID:                "song-1",
				Title:             "Song One",
				ArtistID:          "artist-1",
				MbzReleaseTrackID: "mbid-1",
				Missing:           false,
			}

			// Set up mocks
			artistRepo.SetData(model.Artists{artist1})
			artistRepo.FindByName("Artist One", artist1)
			mediaFileRepo.FindByMBID("mbid-1", song1)

			// Configure the mockAgent
			mockAgent.topSongs = []agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
				{Name: "Song Three", MBID: "mbid-3"},
			}

			// Create a custom agents instance directly with our mock agent
			agentsImpl := &agents.Agents{}
			setAgentField(agentsImpl, "ds", ds)
			setAgentField(agentsImpl, "agents", []agents.Interface{mockAgent})
			em = NewExternalMetadata(ds, agentsImpl)

			// Default to empty response for any queries
			mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{}, nil).Maybe()

			songs, err := em.TopSongs(ctx, "Artist One", 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(1))
			Expect(songs[0].ID).To(Equal("song-1"))
		})
	})

	Describe("TopSongs with agent registration", func() {
		BeforeEach(func() {
			// Set our mock agent as the only agent
			conf.Server.Agents = "mock"

			// Set up test data
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}

			song1 := model.MediaFile{
				ID:                "song-1",
				Title:             "Song One",
				Artist:            "Artist One",
				ArtistID:          "artist-1",
				MbzReleaseTrackID: "mbid-1",
				Missing:           false,
			}

			song2 := model.MediaFile{
				ID:                "song-2",
				Title:             "Song Two",
				Artist:            "Artist One",
				ArtistID:          "artist-1",
				MbzReleaseTrackID: "mbid-2",
				Missing:           false,
			}

			// Set up basic data for the repos
			artistRepo.SetData(model.Artists{artist1})
			mediaFileRepo.SetData(model.MediaFiles{song1, song2})

			// Set up the specific mock responses needed for the TopSongs method
			artistRepo.FindByName("Artist One", artist1)
			mediaFileRepo.FindByMBID("mbid-1", song1)
			mediaFileRepo.FindByMBID("mbid-2", song2)

			// Setup default behavior for empty searches
			mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{}, nil).Maybe()

			// Configure and register the agent
			mockAgent.topSongs = []agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
			}

			// Register our mock agent
			agents.Register("mock", func(model.DataStore) agents.Interface { return mockAgent })

			// Create the provider instance with registered agents
			em = NewExternalMetadata(ds, agents.GetAgents(ds))
		})

		It("returns matching songs from the registered agent", func() {
			songs, err := em.TopSongs(ctx, "Artist One", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(2))
			Expect(songs[0].ID).To(Equal("song-1"))
			Expect(songs[1].ID).To(Equal("song-2"))
		})
	})

	Describe("Error propagation from agents", func() {
		BeforeEach(func() {
			// Set up test data
			artist1 := model.Artist{ID: "artist-1", Name: "Artist One"}

			// Set up basic data for the repos
			artistRepo.SetData(model.Artists{artist1})
			artistRepo.FindByName("Artist One", artist1)

			// Setup default behavior for empty searches
			mediaFileRepo.On("GetAll", mock.Anything).Return(model.MediaFiles{}, nil).Maybe()

			// Create a direct agent that returns an error
			testError := errors.New("direct agent error")
			directAgent := &mockArtistTopSongsAgent{
				getArtistTopSongsFn: func(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
					return nil, testError
				},
			}

			// Create a custom implementation of agents.Agents that will return our error
			directAgentsImpl := &agents.Agents{}
			setAgentField(directAgentsImpl, "ds", ds)
			setAgentField(directAgentsImpl, "agents", []agents.Interface{directAgent})

			// Create a new external metadata instance
			em = NewExternalMetadata(ds, directAgentsImpl)
		})

		It("handles errors from the agent according to current behavior", func() {
			songs, err := em.TopSongs(ctx, "Artist One", 5)

			// Current behavior returns nil for both error and songs
			Expect(err).To(BeNil())
			Expect(songs).To(BeNil())
		})
	})
})

// Mock agent implementation for testing
type mockArtistTopSongsAgent struct {
	agents.Interface
	err                 error
	topSongs            []agents.Song
	lastArtistID        string
	lastArtistName      string
	lastMBID            string
	lastCount           int
	getArtistTopSongsFn func(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error)
}

func (m *mockArtistTopSongsAgent) AgentName() string {
	return "mock"
}

func (m *mockArtistTopSongsAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	m.lastCount = count
	m.lastArtistID = id
	m.lastArtistName = artistName
	m.lastMBID = mbid

	log.Debug(ctx, "MockAgent.GetArtistTopSongs called", "id", id, "name", artistName, "mbid", mbid, "count", count)

	// Use the custom function if available
	if m.getArtistTopSongsFn != nil {
		return m.getArtistTopSongsFn(ctx, id, artistName, mbid, count)
	}

	if m.err != nil {
		log.Debug(ctx, "MockAgent.GetArtistTopSongs returning error", "err", m.err)
		return nil, m.err
	}

	log.Debug(ctx, "MockAgent.GetArtistTopSongs returning songs", "count", len(m.topSongs))
	return m.topSongs, nil
}

// Make sure the mock agent implements the necessary interface
var _ agents.ArtistTopSongsRetriever = (*mockArtistTopSongsAgent)(nil)

// Sets unexported fields in a struct using reflection and unsafe package
func setAgentField(obj interface{}, fieldName string, value interface{}) {
	v := reflect.ValueOf(obj).Elem()
	f := v.FieldByName(fieldName)
	rf := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	rf.Set(reflect.ValueOf(value))
}

// Mocked ArtistRepo that uses testify's mock
type mockArtistRepo struct {
	mock.Mock
	model.ArtistRepository
}

func newMockArtistRepo() *mockArtistRepo {
	return &mockArtistRepo{}
}

func (m *mockArtistRepo) SetData(artists model.Artists) {
	// Store the data for Get queries
	for _, a := range artists {
		m.On("Get", a.ID).Return(&a, nil)
	}
}

func (m *mockArtistRepo) SetError(hasError bool) {
	if hasError {
		m.On("GetAll", mock.Anything).Return(nil, errors.New("error"))
	}
}

func (m *mockArtistRepo) FindByName(name string, artist model.Artist) {
	// Set up a mock for finding an artist by name with LIKE filter, using Anything matcher for flexibility
	m.On("GetAll", mock.Anything).Return(model.Artists{artist}, nil).Once()
}

func (m *mockArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	args := m.Called(mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.Artists), args.Error(1)
}

// Mocked MediaFileRepo that uses testify's mock
type mockMediaFileRepo struct {
	mock.Mock
	model.MediaFileRepository
}

func newMockMediaFileRepo() *mockMediaFileRepo {
	return &mockMediaFileRepo{}
}

func (m *mockMediaFileRepo) SetData(mediaFiles model.MediaFiles) {
	// Store the data for Get queries
	for _, mf := range mediaFiles {
		m.On("Get", mf.ID).Return(&mf, nil)
	}
}

func (m *mockMediaFileRepo) SetError(hasError bool) {
	if hasError {
		m.On("GetAll", mock.Anything).Return(nil, errors.New("error"))
	}
}

func (m *mockMediaFileRepo) FindByMBID(mbid string, mediaFile model.MediaFile) {
	// Set up a mock for finding a media file by MBID using Anything matcher for flexibility
	m.On("GetAll", mock.Anything).Return(model.MediaFiles{mediaFile}, nil).Once()
}

func (m *mockMediaFileRepo) FindByArtistAndTitle(artistID string, title string, mediaFile model.MediaFile) {
	// Set up a mock for finding a media file by artist ID and title
	m.On("GetAll", mock.Anything).Return(model.MediaFiles{mediaFile}, nil).Once()
}

func (m *mockMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	args := m.Called(mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.MediaFiles), args.Error(1)
}
