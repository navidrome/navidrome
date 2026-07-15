package tests

import (
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
	Err        error
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

func (m *MockPlaylistTrackRepo) CountAll(_ ...model.QueryOptions) (int64, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return int64(len(m.Data)), nil
}

func (m *MockPlaylistTrackRepo) GetAll(options ...model.QueryOptions) (model.PlaylistTracks, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.page(options...), nil
}

func (m *MockPlaylistTrackRepo) GetCursor(options ...model.QueryOptions) (model.PlaylistTrackCursor, error) {
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

func (m *MockPlaylistTrackRepo) GetMediaFileIDs(options ...model.QueryOptions) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return slice.Map(m.page(options...), func(t model.PlaylistTrack) string { return t.MediaFileID }), nil
}

func (m *MockPlaylistTrackRepo) Add(ids []string) (int, error) {
	m.AddedIds = append(m.AddedIds, ids...)
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddAlbums(_ []string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddArtists(_ []string) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) AddDiscs(_ []model.DiscID) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.AddCount, nil
}

func (m *MockPlaylistTrackRepo) Delete(ids ...string) error {
	m.DeletedIds = append(m.DeletedIds, ids...)
	return m.Err
}

func (m *MockPlaylistTrackRepo) Reorder(_, _ int) error {
	m.Reordered = true
	return m.Err
}

var _ model.PlaylistTrackRepository = (*MockPlaylistTrackRepo)(nil)
