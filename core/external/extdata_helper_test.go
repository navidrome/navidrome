package external_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/stretchr/testify/mock"
)

// --- Shared Mock Implementations ---

// mockArtistRepo mocks model.ArtistRepository
type mockArtistRepo struct {
	mock.Mock
	model.ArtistRepository
}

func newMockArtistRepo() *mockArtistRepo {
	return &mockArtistRepo{}
}

// SetData sets up basic Get expectations.
func (m *mockArtistRepo) SetData(artists model.Artists) {
	for _, a := range artists {
		artistCopy := a
		m.On("Get", artistCopy.ID).Return(&artistCopy, nil)
	}
}

// Get implements model.ArtistRepository.
func (m *mockArtistRepo) Get(id string) (*model.Artist, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Artist), args.Error(1)
}

// GetAll implements model.ArtistRepository.
func (m *mockArtistRepo) GetAll(options ...model.QueryOptions) (model.Artists, error) {
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

// SetError is a helper to set up a generic error for GetAll.
func (m *mockArtistRepo) SetError(hasError bool) {
	if hasError {
		m.On("GetAll", mock.Anything).Return(nil, errors.New("mock repo error"))
	}
}

// FindByName is a helper to set up a GetAll expectation for finding by name.
func (m *mockArtistRepo) FindByName(name string, artist model.Artist) {
	m.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
		return opt.Filters != nil
	})).Return(model.Artists{artist}, nil).Once()
}

// mockMediaFileRepo mocks model.MediaFileRepository
type mockMediaFileRepo struct {
	mock.Mock
	model.MediaFileRepository
}

func newMockMediaFileRepo() *mockMediaFileRepo {
	return &mockMediaFileRepo{}
}

// SetData sets up basic Get expectations.
func (m *mockMediaFileRepo) SetData(mediaFiles model.MediaFiles) {
	for _, mf := range mediaFiles {
		mfCopy := mf
		m.On("Get", mfCopy.ID).Return(&mfCopy, nil)
	}
}

// Get implements model.MediaFileRepository.
func (m *mockMediaFileRepo) Get(id string) (*model.MediaFile, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MediaFile), args.Error(1)
}

// GetAll implements model.MediaFileRepository.
func (m *mockMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
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

// SetError is a helper to set up a generic error for GetAll.
func (m *mockMediaFileRepo) SetError(hasError bool) {
	if hasError {
		m.On("GetAll", mock.Anything).Return(nil, errors.New("mock repo error"))
	}
}

// FindByMBID is a helper to set up a GetAll expectation for finding by MBID.
func (m *mockMediaFileRepo) FindByMBID(mbid string, mediaFile model.MediaFile) {
	m.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
		return opt.Filters != nil
	})).Return(model.MediaFiles{mediaFile}, nil).Once()
}

// FindByArtistAndTitle is a helper to set up a GetAll expectation for finding by artist/title.
func (m *mockMediaFileRepo) FindByArtistAndTitle(artistID string, title string, mediaFile model.MediaFile) {
	m.On("GetAll", mock.MatchedBy(func(opt model.QueryOptions) bool {
		return opt.Filters != nil
	})).Return(model.MediaFiles{mediaFile}, nil).Once()
}

// mockAlbumRepo mocks model.AlbumRepository
type mockAlbumRepo struct {
	mock.Mock
	model.AlbumRepository
}

func newMockAlbumRepo() *mockAlbumRepo {
	return &mockAlbumRepo{}
}

// Get implements model.AlbumRepository.
func (m *mockAlbumRepo) Get(id string) (*model.Album, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Album), args.Error(1)
}

// GetAll implements model.AlbumRepository.
func (m *mockAlbumRepo) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	argsSlice := make([]interface{}, len(options))
	for i, v := range options {
		argsSlice[i] = v
	}
	args := m.Called(argsSlice...)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(model.Albums), args.Error(1)
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
