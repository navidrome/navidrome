package extdata

import (
	"context"
	"errors"
	"reflect"
	"unsafe"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	_ "github.com/navidrome/navidrome/core/agents/lastfm"
	_ "github.com/navidrome/navidrome/core/agents/listenbrainz"
	_ "github.com/navidrome/navidrome/core/agents/spotify"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

// Gets unexported field value from a struct using reflection and unsafe package
func getField(obj interface{}, fieldName string) interface{} {
	v := reflect.ValueOf(obj).Elem()
	f := v.FieldByName(fieldName)
	rf := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	return rf.Interface()
}

// Extended MockArtistRepo with GetAll query behavior needed for tests
type testArtistRepo struct {
	*tests.MockArtistRepo
}

func (m *testArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	// Get the error state using reflection
	if getField(m.MockArtistRepo, "Err").(bool) {
		return nil, errors.New("error")
	}

	// Get the data using reflection
	dataMap := getField(m.MockArtistRepo, "Data").(map[string]*model.Artist)

	// Convert map to slice
	artists := make(model.Artists, 0, len(dataMap))
	for _, a := range dataMap {
		artists = append(artists, *a)
	}

	// No filters means return all
	if len(options) == 0 || options[0].Filters == nil {
		return artists, nil
	}

	// Process filters
	if len(options) > 0 && options[0].Filters != nil {
		switch f := options[0].Filters.(type) {
		case squirrel.Like:
			if nameFilter, ok := f["artist.name"]; ok {
				// Convert to string and remove any SQL wildcard characters for simple comparison
				name := str.Clear(nameFilter.(string))
				log.Debug("ArtistRepo.GetAll: Looking for artist by name", "name", name)

				for _, a := range artists {
					if a.Name == name {
						log.Debug("ArtistRepo.GetAll: Found artist", "id", a.ID, "name", a.Name)
						return model.Artists{a}, nil
					}
				}
			}
		case squirrel.Eq:
			if ids, ok := f["artist.id"]; ok {
				var result model.Artists
				for _, a := range artists {
					for _, id := range ids.([]string) {
						if a.ID == id {
							result = append(result, a)
						}
					}
				}
				return result, nil
			}
		}
	}

	log.Debug("ArtistRepo.GetAll: No matching artist found")
	// If no filter matches or no options, return empty
	return model.Artists{}, nil
}

// Extended MockMediaFileRepo with GetAll query behavior needed for tests
type testMediaFileRepo struct {
	*tests.MockMediaFileRepo
}

func (m *testMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	// Get the error state using reflection
	if getField(m.MockMediaFileRepo, "Err").(bool) {
		return nil, errors.New("error")
	}

	// Get the data using reflection
	dataMap := getField(m.MockMediaFileRepo, "Data").(map[string]*model.MediaFile)

	// Convert map to slice
	mediaFiles := make(model.MediaFiles, 0, len(dataMap))
	for _, mf := range dataMap {
		mediaFiles = append(mediaFiles, *mf)
	}

	if len(options) == 0 {
		return mediaFiles, nil
	}

	// Process filters
	if options[0].Filters != nil {
		switch filter := options[0].Filters.(type) {
		case squirrel.And:
			// This handles the case where we search by artist ID and title
			log.Debug("MediaFileRepo.GetAll: Processing AND filter")
			return m.handleAndFilter(filter, options[0], mediaFiles)
		case squirrel.Eq:
			// This handles the case where we search by mbz_recording_id
			log.Debug("MediaFileRepo.GetAll: Processing EQ filter")
			if mbid, ok := filter["mbz_recording_id"]; ok {
				log.Debug("MediaFileRepo.GetAll: Looking for MBID", "mbid", mbid)
				for _, mf := range mediaFiles {
					if mf.MbzReleaseTrackID == mbid.(string) && !mf.Missing {
						log.Debug("MediaFileRepo.GetAll: Found matching file by MBID", "id", mf.ID, "title", mf.Title)
						return model.MediaFiles{mf}, nil
					}
				}
			}
		}
	}

	log.Debug("MediaFileRepo.GetAll: No matches found")
	return model.MediaFiles{}, nil
}

func (m *testMediaFileRepo) handleAndFilter(andFilter squirrel.And, option model.QueryOptions, mediaFiles model.MediaFiles) (model.MediaFiles, error) {
	// Get matches for each condition
	var artistMatches []model.MediaFile
	var titleMatches []model.MediaFile
	var notMissingMatches []model.MediaFile

	// First identify non-missing files
	for _, mf := range mediaFiles {
		if !mf.Missing {
			notMissingMatches = append(notMissingMatches, mf)
		}
	}

	log.Debug("MediaFileRepo.handleAndFilter: Processing filters", "filterCount", len(andFilter))

	// Now look for matches on other criteria
	for _, sqlizer := range andFilter {
		switch filter := sqlizer.(type) {
		case squirrel.Or:
			// Handle artist ID matching
			log.Debug("MediaFileRepo.handleAndFilter: Processing OR filter")
			for _, orCond := range filter {
				if eqCond, ok := orCond.(squirrel.Eq); ok {
					if artistID, ok := eqCond["artist_id"]; ok {
						log.Debug("MediaFileRepo.handleAndFilter: Looking for artist_id", "artistID", artistID)
						for _, mf := range notMissingMatches {
							if mf.ArtistID == artistID.(string) {
								log.Debug("MediaFileRepo.handleAndFilter: Found match by artist_id", "id", mf.ID, "title", mf.Title)
								artistMatches = append(artistMatches, mf)
							}
						}
					}
					if albumArtistID, ok := eqCond["album_artist_id"]; ok {
						log.Debug("MediaFileRepo.handleAndFilter: Looking for album_artist_id", "albumArtistID", albumArtistID)
						for _, mf := range notMissingMatches {
							if mf.AlbumArtistID == albumArtistID.(string) {
								log.Debug("MediaFileRepo.handleAndFilter: Found match by album_artist_id", "id", mf.ID, "title", mf.Title)
								artistMatches = append(artistMatches, mf)
							}
						}
					}
				}
			}
		case squirrel.Like:
			// Handle title matching
			log.Debug("MediaFileRepo.handleAndFilter: Processing LIKE filter")
			if orderTitle, ok := filter["order_title"]; ok {
				normalizedTitle := str.SanitizeFieldForSorting(orderTitle.(string))
				log.Debug("MediaFileRepo.handleAndFilter: Looking for title match", "normalizedTitle", normalizedTitle)
				for _, mf := range notMissingMatches {
					normalizedMfTitle := str.SanitizeFieldForSorting(mf.Title)
					log.Debug("MediaFileRepo.handleAndFilter: Comparing titles", "fileTitle", mf.Title, "normalizedFileTitle", normalizedMfTitle)
					if normalizedTitle == normalizedMfTitle {
						log.Debug("MediaFileRepo.handleAndFilter: Found title match", "id", mf.ID, "title", mf.Title)
						titleMatches = append(titleMatches, mf)
					}
				}
			}
		case squirrel.Eq:
			// Handle missing check
			if missingFlag, ok := filter["missing"]; ok && !missingFlag.(bool) {
				// This is already handled above when we build notMissingMatches
				continue
			}
		}
	}

	log.Debug("MediaFileRepo.handleAndFilter: Matching stats", "artistMatches", len(artistMatches), "titleMatches", len(titleMatches))

	// Find matches that satisfy all conditions
	var result model.MediaFiles
	for _, am := range artistMatches {
		for _, tm := range titleMatches {
			if am.ID == tm.ID {
				log.Debug("MediaFileRepo.handleAndFilter: Found complete match", "id", am.ID, "title", am.Title)
				result = append(result, am)
			}
		}
	}

	// Apply any sort and limit from the options
	if option.Max > 0 && len(result) > option.Max {
		result = result[:option.Max]
	}

	log.Debug("MediaFileRepo.handleAndFilter: Returning results", "count", len(result))
	return result, nil
}

var _ = Describe("Provider", func() {
	var ds model.DataStore
	var em Provider
	var mockAgent *mockArtistTopSongsAgent
	var mockArtistRepo *testArtistRepo
	var mockMediaFileRepo *testMediaFileRepo
	var ctx context.Context
	var originalAgentsConfig string

	BeforeEach(func() {
		ctx = context.Background()

		// Store the original agents config to restore it later
		originalAgentsConfig = conf.Server.Agents

		// Setup mocks
		artistRepo := tests.CreateMockArtistRepo()
		mockArtistRepo = &testArtistRepo{MockArtistRepo: artistRepo}

		mediaFileRepo := tests.CreateMockMediaFileRepo()
		mockMediaFileRepo = &testMediaFileRepo{MockMediaFileRepo: mediaFileRepo}

		ds = &tests.MockDataStore{
			MockedArtist:    mockArtistRepo,
			MockedMediaFile: mockMediaFileRepo,
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
			// Set up artists data
			mockArtistRepo.SetData(model.Artists{
				{ID: "artist-1", Name: "Artist One"},
				{ID: "artist-2", Name: "Artist Two"},
			})

			// Set up mediafiles data with all necessary fields for matching
			mockMediaFileRepo.SetData(model.MediaFiles{
				{
					ID:                "song-1",
					Title:             "Song One",
					Artist:            "Artist One",
					ArtistID:          "artist-1",
					AlbumArtistID:     "artist-1",
					MbzReleaseTrackID: "mbid-1",
					Missing:           false,
				},
				{
					ID:                "song-2",
					Title:             "Song Two",
					Artist:            "Artist One",
					ArtistID:          "artist-1",
					AlbumArtistID:     "artist-1",
					MbzReleaseTrackID: "mbid-2",
					Missing:           false,
				},
				{
					ID:                "song-3",
					Title:             "Song Three",
					Artist:            "Artist Two",
					ArtistID:          "artist-2",
					AlbumArtistID:     "artist-2",
					MbzReleaseTrackID: "mbid-3",
					Missing:           false,
				},
			})

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
			// Set an error for mockArtistRepo to simulate artist not found
			mockArtistRepo.SetError(true)

			songs, err := em.TopSongs(ctx, "Unknown Artist", 5)

			Expect(err).To(BeNil())
			Expect(songs).To(BeNil())
		})

		It("returns empty list when no matching songs are found", func() {
			// Configure the agent to return songs that don't match our repo
			mockAgent.topSongs = []agents.Song{
				{Name: "Nonexistent Song", MBID: "unknown-mbid"},
			}

			songs, err := em.TopSongs(ctx, "Artist One", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(0))
		})

		It("returns nil when agent returns errors", func() {
			// Reset the agent error state
			mockArtistRepo.SetError(false)

			// Set the error
			testError := errors.New("some agent error")
			mockAgent.err = testError

			songs, err := em.TopSongs(ctx, "Artist One", 5)

			// Current behavior returns nil for both error and songs
			Expect(err).To(BeNil())
			Expect(songs).To(BeNil())
		})

		It("respects count parameter", func() {
			mockAgent.topSongs = []agents.Song{
				{Name: "Song One", MBID: "mbid-1"},
				{Name: "Song Two", MBID: "mbid-2"},
				{Name: "Song Three", MBID: "mbid-3"},
			}

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

			// Set up artists data
			mockArtistRepo.SetData(model.Artists{
				{ID: "artist-1", Name: "Artist One"},
			})

			// Set up mediafiles data
			mockMediaFileRepo.SetData(model.MediaFiles{
				{
					ID:                "song-1",
					Title:             "Song One",
					Artist:            "Artist One",
					ArtistID:          "artist-1",
					MbzReleaseTrackID: "mbid-1",
					Missing:           false,
				},
				{
					ID:                "song-2",
					Title:             "Song Two",
					Artist:            "Artist One",
					ArtistID:          "artist-1",
					MbzReleaseTrackID: "mbid-2",
					Missing:           false,
				},
			})

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
			// Set up artists data
			mockArtistRepo.SetData(model.Artists{
				{ID: "artist-1", Name: "Artist One"},
			})

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
