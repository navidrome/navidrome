package core

import (
	"context"
	"errors"
	"log"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/str"
	"github.com/stretchr/testify/assert"
)

// Custom implementation of ArtistRepository for testing
type customArtistRepo struct {
	model.ArtistRepository
	data map[string]*model.Artist
	err  bool
}

func newCustomArtistRepo() *customArtistRepo {
	return &customArtistRepo{
		data: make(map[string]*model.Artist),
	}
}

func (m *customArtistRepo) SetError(err bool) {
	m.err = err
}

func (m *customArtistRepo) SetData(artists model.Artists) {
	m.data = make(map[string]*model.Artist)
	for i, a := range artists {
		m.data[a.ID] = &artists[i]
	}
}

// Key implementation needed for the test
func (m *customArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	if m.err {
		return nil, errors.New("error")
	}

	// No filters means return all
	if len(options) == 0 || options[0].Filters == nil {
		result := make(model.Artists, 0, len(m.data))
		for _, a := range m.data {
			result = append(result, *a)
		}
		return result, nil
	}

	// Handle filter by name (for findArtistByName)
	if len(options) > 0 && options[0].Filters != nil {
		switch filter := options[0].Filters.(type) {
		case squirrel.Like:
			if nameFilter, ok := filter["artist.name"]; ok {
				name := strings.Trim(nameFilter.(string), "%")
				log.Printf("ArtistRepo.GetAll: Looking for artist by name: %s", name)

				for _, a := range m.data {
					log.Printf("ArtistRepo.GetAll: Comparing with artist: %s", a.Name)
					if a.Name == name {
						log.Printf("ArtistRepo.GetAll: Found artist: %s (ID: %s)", a.Name, a.ID)
						return model.Artists{*a}, nil
					}
				}
			}
		}
	}

	log.Println("ArtistRepo.GetAll: No matching artist found")
	return model.Artists{}, nil
}

// Custom implementation of MediaFileRepository for testing
type customMediaFileRepo struct {
	model.MediaFileRepository
	data map[string]*model.MediaFile
	err  bool
}

func newCustomMediaFileRepo() *customMediaFileRepo {
	return &customMediaFileRepo{
		data: make(map[string]*model.MediaFile),
	}
}

func (m *customMediaFileRepo) SetError(err bool) {
	m.err = err
}

func (m *customMediaFileRepo) SetData(mediaFiles model.MediaFiles) {
	m.data = make(map[string]*model.MediaFile)
	for i, mf := range mediaFiles {
		m.data[mf.ID] = &mediaFiles[i]
	}
}

// Key implementation needed for the test
func (m *customMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("error")
	}

	// No filters means return all
	if len(options) == 0 || options[0].Filters == nil {
		result := make(model.MediaFiles, 0, len(m.data))
		for _, mf := range m.data {
			result = append(result, *mf)
		}
		return result, nil
	}

	// Check if we're searching by MBID
	if len(options) > 0 && options[0].Filters != nil {
		// Log all filter types
		log.Printf("MediaFileRepo.GetAll: Filter type: %T", options[0].Filters)

		switch filter := options[0].Filters.(type) {
		case squirrel.And:
			log.Printf("MediaFileRepo.GetAll: Processing AND filter with %d conditions", len(filter))

			// First check if there's a mbz_recording_id in one of the AND conditions
			for i, cond := range filter {
				log.Printf("MediaFileRepo.GetAll: AND condition %d is of type %T", i, cond)

				if eq, ok := cond.(squirrel.Eq); ok {
					log.Printf("MediaFileRepo.GetAll: Eq condition: %+v", eq)

					if mbid, hasMbid := eq["mbz_recording_id"]; hasMbid {
						log.Printf("MediaFileRepo.GetAll: Looking for MBID: %s", mbid)

						for _, mf := range m.data {
							if mf.MbzReleaseTrackID == mbid.(string) && !mf.Missing {
								log.Printf("MediaFileRepo.GetAll: Found match by MBID: %s (Title: %s)", mf.ID, mf.Title)
								return model.MediaFiles{*mf}, nil
							}
						}
					}
				}
			}

			// Otherwise, find by artist ID and title
			var artistMatches model.MediaFiles
			var titleMatches model.MediaFiles
			var notMissingMatches model.MediaFiles

			// Get non-missing files
			for _, mf := range m.data {
				if !mf.Missing {
					notMissingMatches = append(notMissingMatches, *mf)
				}
			}

			log.Printf("MediaFileRepo.GetAll: Found %d non-missing files", len(notMissingMatches))

			for i, cond := range filter {
				log.Printf("MediaFileRepo.GetAll: Processing condition %d of type %T", i, cond)

				switch subFilter := cond.(type) {
				case squirrel.Or:
					log.Printf("MediaFileRepo.GetAll: Processing OR condition with %d subconditions", len(subFilter))

					// Check for artist_id
					for j, orCond := range subFilter {
						log.Printf("MediaFileRepo.GetAll: OR subcondition %d is of type %T", j, orCond)

						if eq, ok := orCond.(squirrel.Eq); ok {
							log.Printf("MediaFileRepo.GetAll: Eq condition: %+v", eq)

							if artistID, ok := eq["artist_id"]; ok {
								log.Printf("MediaFileRepo.GetAll: Looking for artist_id: %s", artistID)

								for _, mf := range notMissingMatches {
									if mf.ArtistID == artistID.(string) {
										log.Printf("MediaFileRepo.GetAll: Found match by artist_id: %s (Title: %s)", mf.ID, mf.Title)
										artistMatches = append(artistMatches, mf)
									}
								}
							}
						}
					}
				case squirrel.Like:
					log.Printf("MediaFileRepo.GetAll: Processing LIKE condition: %+v", subFilter)

					// Check for title match
					if orderTitle, ok := subFilter["order_title"]; ok {
						normalizedTitle := str.SanitizeFieldForSorting(orderTitle.(string))
						log.Printf("MediaFileRepo.GetAll: Looking for normalized title: %s", normalizedTitle)

						for _, mf := range notMissingMatches {
							normalizedMfTitle := str.SanitizeFieldForSorting(mf.Title)
							log.Printf("MediaFileRepo.GetAll: Comparing with title: %s (normalized: %s)", mf.Title, normalizedMfTitle)

							if normalizedTitle == normalizedMfTitle {
								log.Printf("MediaFileRepo.GetAll: Found title match: %s", mf.ID)
								titleMatches = append(titleMatches, mf)
							}
						}
					}
				}
			}

			log.Printf("MediaFileRepo.GetAll: Found %d artist matches and %d title matches", len(artistMatches), len(titleMatches))

			// Find records that match both artist and title
			var results model.MediaFiles
			for _, am := range artistMatches {
				for _, tm := range titleMatches {
					if am.ID == tm.ID {
						log.Printf("MediaFileRepo.GetAll: Found complete match: %s", am.ID)
						results = append(results, am)
					}
				}
			}

			if len(results) > 0 {
				// Apply Max if specified
				if options[0].Max > 0 && len(results) > options[0].Max {
					results = results[:options[0].Max]
				}
				log.Printf("MediaFileRepo.GetAll: Returning %d results", len(results))
				return results, nil
			}
		case squirrel.Eq:
			log.Printf("MediaFileRepo.GetAll: Processing Eq filter: %+v", filter)

			// Handle direct MBID lookup
			if mbid, ok := filter["mbz_recording_id"]; ok {
				log.Printf("MediaFileRepo.GetAll: Looking for MBID: %s", mbid)

				for _, mf := range m.data {
					if mf.MbzReleaseTrackID == mbid.(string) && !mf.Missing {
						log.Printf("MediaFileRepo.GetAll: Found match by MBID: %s (Title: %s)", mf.ID, mf.Title)
						return model.MediaFiles{*mf}, nil
					}
				}
			}
		}
	}

	log.Println("MediaFileRepo.GetAll: No matches found")
	return model.MediaFiles{}, nil
}

// Mock Agent implementation
type MockAgent struct {
	agents.Interface
	topSongs []agents.Song
	err      error
}

func (m *MockAgent) AgentName() string {
	return "mock"
}

func (m *MockAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	log.Printf("MockAgent.GetArtistTopSongs called: id=%s, name=%s, mbid=%s, count=%d", id, artistName, mbid, count)

	if m.err != nil {
		log.Printf("MockAgent.GetArtistTopSongs returning error: %v", m.err)
		return nil, m.err
	}

	log.Printf("MockAgent.GetArtistTopSongs returning %d songs", len(m.topSongs))
	return m.topSongs, nil
}

// Ensure MockAgent implements the necessary interface
var _ agents.ArtistTopSongsRetriever = (*MockAgent)(nil)

// Sets unexported fields in a struct using reflection and unsafe package
func setStructField(obj interface{}, fieldName string, value interface{}) {
	v := reflect.ValueOf(obj).Elem()
	f := v.FieldByName(fieldName)
	rf := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	rf.Set(reflect.ValueOf(value))
}

// Direct implementation of ExternalMetadata for testing that avoids agents registration issues
func TestDirectTopSongs(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create custom mock repositories
	mockArtistRepo := newCustomArtistRepo()
	mockMediaFileRepo := newCustomMediaFileRepo()

	// Configure mock data
	artist := model.Artist{ID: "artist-1", Name: "Artist One"}
	mockArtistRepo.SetData(model.Artists{artist})

	log.Printf("Test: Set up artist: %s (ID: %s)", artist.Name, artist.ID)

	mediaFiles := []model.MediaFile{
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
	}
	mockMediaFileRepo.SetData(model.MediaFiles(mediaFiles))

	for _, mf := range mediaFiles {
		log.Printf("Test: Set up media file: %s (ID: %s, MBID: %s, ArtistID: %s)",
			mf.Title, mf.ID, mf.MbzReleaseTrackID, mf.ArtistID)
	}

	// Create mock datastore
	mockDS := &tests.MockDataStore{
		MockedArtist:    mockArtistRepo,
		MockedMediaFile: mockMediaFileRepo,
	}

	// Create mock agent
	mockAgent := &MockAgent{
		topSongs: []agents.Song{
			{Name: "Song One", MBID: "mbid-1"},
			{Name: "Song Two", MBID: "mbid-2"},
		},
	}

	// Use the real agents.Agents implementation but with our mock agent
	agentsImpl := &agents.Agents{}

	// Set unexported fields using reflection and unsafe
	setStructField(agentsImpl, "ds", mockDS)
	setStructField(agentsImpl, "agents", []agents.Interface{mockAgent})

	// Create our service under test
	em := NewExternalMetadata(mockDS, agentsImpl)

	// Test
	log.Printf("Test: Calling TopSongs for artist: %s", "Artist One")
	songs, err := em.TopSongs(ctx, "Artist One", 5)

	log.Printf("Test: Result: error=%v, songs=%v", err, songs)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, songs, 2)
	if len(songs) > 0 {
		assert.Equal(t, "song-1", songs[0].ID)
	}
	if len(songs) > 1 {
		assert.Equal(t, "song-2", songs[1].ID)
	}
}

func TestTopSongs(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Store the original config to restore it later
	originalAgentsConfig := conf.Server.Agents

	// Set our mock agent as the only agent
	conf.Server.Agents = "mock"
	defer func() {
		conf.Server.Agents = originalAgentsConfig
	}()

	// Clear the agents map to prevent interference
	agents.Map = nil

	// Create custom mock repositories
	mockArtistRepo := newCustomArtistRepo()
	mockMediaFileRepo := newCustomMediaFileRepo()

	// Configure mock data
	artist := model.Artist{ID: "artist-1", Name: "Artist One"}
	mockArtistRepo.SetData(model.Artists{artist})

	log.Printf("Test: Set up artist: %s (ID: %s)", artist.Name, artist.ID)

	mediaFiles := []model.MediaFile{
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
	}
	mockMediaFileRepo.SetData(model.MediaFiles(mediaFiles))

	for _, mf := range mediaFiles {
		log.Printf("Test: Set up media file: %s (ID: %s, MBID: %s, ArtistID: %s)",
			mf.Title, mf.ID, mf.MbzReleaseTrackID, mf.ArtistID)
	}

	// Create mock datastore
	mockDS := &tests.MockDataStore{
		MockedArtist:    mockArtistRepo,
		MockedMediaFile: mockMediaFileRepo,
	}

	// Create and register a mock agent
	mockAgent := &MockAgent{
		topSongs: []agents.Song{
			{Name: "Song One", MBID: "mbid-1"},
			{Name: "Song Two", MBID: "mbid-2"},
		},
	}

	// Register our mock agent - this is key to making it available
	agents.Register("mock", func(model.DataStore) agents.Interface { return mockAgent })

	// Dump the registered agents for debugging
	log.Printf("Test: Registered agents:")
	for name := range agents.Map {
		log.Printf("  - %s", name)
	}

	// Create the service to test
	log.Printf("Test: Creating ExternalMetadata with conf.Server.Agents=%s", conf.Server.Agents)
	em := NewExternalMetadata(mockDS, agents.GetAgents(mockDS))

	// Test
	log.Printf("Test: Calling TopSongs for artist: %s", "Artist One")
	songs, err := em.TopSongs(ctx, "Artist One", 5)

	log.Printf("Test: Result: error=%v, songs=%v", err, songs)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, songs, 2)
	if len(songs) > 0 {
		assert.Equal(t, "song-1", songs[0].ID)
	}
	if len(songs) > 1 {
		assert.Equal(t, "song-2", songs[1].ID)
	}
}
