package tests

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type MockPlaylistTrackRepo struct {
	model.PlaylistTrackRepository
	Data       model.PlaylistTracks
	Options    model.QueryOptions
	AddedIds   []string
	DeletedIds []string
	Reordered  bool
	AddCount   int
	InsertPos  int
	Err        error
	AlbumIDs   []string // stubbed result for GetAlbumIDs, ignoring options
}

func (m *MockPlaylistTrackRepo) SetData(tracks model.PlaylistTracks) {
	m.Data = tracks
}

// page applies Max/Offset as the real repository's SQL would.
func (m *MockPlaylistTrackRepo) page(options ...model.QueryOptions) model.PlaylistTracks {
	var opts model.QueryOptions
	if len(options) > 0 {
		opts = options[0]
		m.Options = opts
	}
	tracks := m.Data
	if opts.Offset >= len(tracks) {
		return nil
	}
	tracks = tracks[opts.Offset:]
	if opts.Max > 0 && opts.Max < len(tracks) {
		tracks = tracks[:opts.Max]
	}
	return tracks
}

func (m *MockPlaylistTrackRepo) CountAll(_ context.Context, _ ...model.QueryOptions) (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return int64(len(m.Data)), nil
}

func (m *MockPlaylistTrackRepo) GetAll(_ context.Context, options ...model.QueryOptions) (model.PlaylistTracks, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.page(options...), nil
}

func (m *MockPlaylistTrackRepo) GetCursor(_ context.Context, options ...model.QueryOptions) (model.PlaylistTrackCursor, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	tracks := m.page(options...)
	return func(yield func(model.PlaylistTrack, error) bool) {
		for _, t := range tracks {
			if !yield(t, nil) {
				return
			}
		}
	}, nil
}

func (m *MockPlaylistTrackRepo) GetAlbumIDs(context.Context, ...model.QueryOptions) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.AlbumIDs, nil
}

func (m *MockPlaylistTrackRepo) GetMediaFileIDs(_ context.Context, options ...model.QueryOptions) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return slice.Map(m.page(options...), func(t model.PlaylistTrack) string { return t.MediaFileID }), nil
}

func (m *MockPlaylistTrackRepo) Add(_ context.Context, ids []string) (int, error) {
	m.AddedIds = append(m.AddedIds, ids...)
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) Insert(ctx context.Context, ids []string, pos int) (int, error) {
	m.InsertPos = pos
	return m.Add(ctx, ids)
}

func (m *MockPlaylistTrackRepo) AddAlbums(_ context.Context, _ []string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddArtists(_ context.Context, _ []string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddDiscs(_ context.Context, _ []model.DiscID) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) Delete(_ context.Context, ids ...string) error {
	m.DeletedIds = append(m.DeletedIds, ids...)
	return m.Err
}

func (m *MockPlaylistTrackRepo) Reorder(_ context.Context, _, _ int) error {
	m.Reordered = true
	return m.Err
}

var _ model.PlaylistTrackRepository = (*MockPlaylistTrackRepo)(nil)
