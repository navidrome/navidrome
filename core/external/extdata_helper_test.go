package external_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/stretchr/testify/mock"
)

var adminUser = model.User{ID: "admin-1", UserName: "admin-1", IsAdmin: true}

// credited returns participants crediting the artists in the main artist role.
func credited(artists ...model.Artist) model.Participants {
	p := model.Participants{}
	p.Add(model.RoleArtist, artists...)
	return p
}

var errDBDown = errors.New("db down")

// failingArtists serves an artist repository whose lookups fail with err.
func failingArtists(ds model.DataStore, err error) model.DataStore {
	return persistencetest.Stub(ds, persistencetest.Repos{Artist: func(r model.ArtistRepository) model.ArtistRepository {
		return failingArtistRepo{r, err}
	}})
}

type failingArtistRepo struct {
	model.ArtistRepository
	err error
}

func (r failingArtistRepo) Get(context.Context, string) (*model.Artist, error) {
	return nil, r.err
}

func (r failingArtistRepo) GetAll(context.Context, ...model.QueryOptions) (model.Artists, error) {
	return nil, r.err
}

// mockSimilarArtistAgent mocks agents implementing ArtistTopSongsRetriever and ArtistSimilarRetriever
type mockSimilarArtistAgent struct {
	mock.Mock
	agents.Interface // Embed to satisfy methods not explicitly mocked
}

func (m *mockSimilarArtistAgent) AgentName() string {
	return "mockSimilar"
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

// mockAgents mocks the main Agents interface used by Provider
type mockAgents struct {
	mock.Mock      // Embed testify mock
	topSongsAgent  agents.ArtistTopSongsRetriever
	similarAgent   agents.ArtistSimilarRetriever
	imageAgent     agents.ArtistImageRetriever
	albumInfoAgent interface {
		agents.AlbumInfoRetriever
		agents.AlbumImageRetriever
	}
	bioAgent  agents.ArtistBiographyRetriever
	mbidAgent agents.ArtistMBIDRetriever
	urlAgent  agents.ArtistURLRetriever
	agents.Interface
}

func (m *mockAgents) AgentName() string {
	return "mockCombined"
}

func (m *mockAgents) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	if m.similarAgent != nil {
		return m.similarAgent.GetSimilarArtists(ctx, id, name, mbid, limit)
	}
	args := m.Called(ctx, id, name, mbid, limit)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Artist), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	if m.topSongsAgent != nil {
		return m.topSongsAgent.GetArtistTopSongs(ctx, id, artistName, mbid, count)
	}
	args := m.Called(ctx, id, artistName, mbid, count)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Song), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	if m.albumInfoAgent != nil {
		return m.albumInfoAgent.GetAlbumInfo(ctx, name, artist, mbid)
	}
	args := m.Called(ctx, name, artist, mbid)
	if args.Get(0) != nil {
		return args.Get(0).(*agents.AlbumInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	if m.mbidAgent != nil {
		return m.mbidAgent.GetArtistMBID(ctx, id, name)
	}
	args := m.Called(ctx, id, name)
	return args.String(0), args.Error(1)
}

func (m *mockAgents) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	if m.urlAgent != nil {
		return m.urlAgent.GetArtistURL(ctx, id, name, mbid)
	}
	args := m.Called(ctx, id, name, mbid)
	return args.String(0), args.Error(1)
}

func (m *mockAgents) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	if m.bioAgent != nil {
		return m.bioAgent.GetArtistBiography(ctx, id, name, mbid)
	}
	args := m.Called(ctx, id, name, mbid)
	return args.String(0), args.Error(1)
}

func (m *mockAgents) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	if m.imageAgent != nil {
		return m.imageAgent.GetArtistImages(ctx, id, name, mbid)
	}
	args := m.Called(ctx, id, name, mbid)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.ExternalImage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	if m.albumInfoAgent != nil {
		return m.albumInfoAgent.GetAlbumImages(ctx, name, artist, mbid)
	}
	args := m.Called(ctx, name, artist, mbid)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.ExternalImage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetSimilarSongsByTrack(ctx context.Context, id, name, artist, mbid string, count int) ([]agents.Song, error) {
	args := m.Called(ctx, id, name, artist, mbid, count)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Song), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetSimilarSongsByAlbum(ctx context.Context, id, name, artist, mbid string, count int) ([]agents.Song, error) {
	args := m.Called(ctx, id, name, artist, mbid, count)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Song), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgents) GetSimilarSongsByArtist(ctx context.Context, id, name, mbid string, count int) ([]agents.Song, error) {
	args := m.Called(ctx, id, name, mbid, count)
	if args.Get(0) != nil {
		return args.Get(0).([]agents.Song), args.Error(1)
	}
	return nil, args.Error(1)
}

func ids(mfs model.MediaFiles) []string {
	return slice.Map(mfs, func(mf model.MediaFile) string { return mf.ID })
}
