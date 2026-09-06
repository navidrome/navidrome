package tests

import (
	"errors"
	"sync"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

func CreateMockAlbumRepo() *MockAlbumRepo {
	return &MockAlbumRepo{
		Data: make(map[string]*model.Album),
	}
}

type MockAlbumRepo struct {
	model.AlbumRepository
	mu                      sync.RWMutex
	Data                    map[string]*model.Album
	All                     model.Albums
	Err                     bool
	Options                 model.QueryOptions
	ReassignAnnotationCalls map[string]string // prevID -> newID
	CopyAttributesCalls     map[string]string // fromID -> toID
}

func (m *MockAlbumRepo) SetError(err bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Err = err
}

func (m *MockAlbumRepo) SetData(albums model.Albums) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.All = append(model.Albums(nil), albums...)
	m.Data = make(map[string]*model.Album, len(m.All))
	for i := range m.All {
		m.Data[m.All[i].ID] = &m.All[i]
	}
}

func (m *MockAlbumRepo) Exists(id string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Err {
		return false, errors.New("unexpected error")
	}
	_, found := m.Data[id]
	return found, nil
}

func (m *MockAlbumRepo) Get(id string) (*model.Album, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	if d, ok := m.Data[id]; ok {
		copied := *d
		return &copied, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockAlbumRepo) Put(al *model.Album) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err {
		return errors.New("unexpected error")
	}
	if al.ID == "" {
		al.ID = id.NewRandom()
	}
	// Keep the caller's pointer so IncPlayCount is visible on the original value.
	m.Data[al.ID] = al
	found := false
	for i := range m.All {
		if m.All[i].ID == al.ID {
			m.All[i] = *al
			found = true
			break
		}
	}
	if !found {
		m.All = append(m.All, *al)
	}
	return nil
}

func (m *MockAlbumRepo) GetAll(qo ...model.QueryOptions) (model.Albums, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.Err {
		return nil, errors.New("unexpected error")
	}
	out := make(model.Albums, 0, len(m.Data))
	for _, album := range m.Data {
		out = append(out, *album)
	}
	return out, nil
}

func (m *MockAlbumRepo) GetCursor(qo ...model.QueryOptions) (model.AlbumCursor, error) {
	all, err := m.GetAll(qo...)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.Album, error) bool) {
		for _, al := range all {
			if !yield(al, nil) {
				return
			}
		}
	}, nil
}

func (m *MockAlbumRepo) IncPlayCount(id string, timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err {
		return errors.New("unexpected error")
	}
	if d, ok := m.Data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}
func (m *MockAlbumRepo) CountAll(...model.QueryOptions) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.Data)), nil
}

func (m *MockAlbumRepo) GetTouchedAlbums(libID int) (model.AlbumCursor, error) {
	m.mu.RLock()
	if m.Err {
		m.mu.RUnlock()
		return nil, errors.New("unexpected error")
	}
	albums := make(model.Albums, 0, len(m.Data))
	for _, a := range m.Data {
		albums = append(albums, *a)
	}
	m.mu.RUnlock()
	return func(yield func(model.Album, error) bool) {
		for _, a := range albums {
			if a.ID == "error" {
				if !yield(a, errors.New("error")) {
					break
				}
				continue
			}
			if a.LibraryID != libID {
				continue
			}
			if !yield(a, nil) {
				break
			}
		}
	}, nil
}

func (m *MockAlbumRepo) UpdateExternalInfo(album *model.Album) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err {
		return errors.New("unexpected error")
	}
	if album == nil {
		return nil
	}
	if m.Data == nil {
		m.Data = make(map[string]*model.Album)
	}
	if d, ok := m.Data[album.ID]; ok {
		*d = *album
		for i := range m.All {
			if m.All[i].ID == album.ID {
				m.All[i] = *album
				break
			}
		}
		return nil
	}
	copied := *album
	m.All = append(m.All, copied)
	m.Data[album.ID] = &m.All[len(m.All)-1]
	return nil
}

func (m *MockAlbumRepo) Search(q string, options ...model.QueryOptions) (model.Albums, error) {
	return m.GetAll(options...)
}

// ReassignAnnotation reassigns annotations from one album to another
func (m *MockAlbumRepo) ReassignAnnotation(prevID string, newID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err {
		return errors.New("unexpected error")
	}
	if m.ReassignAnnotationCalls == nil {
		m.ReassignAnnotationCalls = make(map[string]string)
	}
	m.ReassignAnnotationCalls[prevID] = newID
	return nil
}

// CopyAttributes copies attributes from one album to another
func (m *MockAlbumRepo) CopyAttributes(fromID, toID string, columns ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err {
		return errors.New("unexpected error")
	}
	from, ok := m.Data[fromID]
	if !ok {
		return model.ErrNotFound
	}
	to, ok := m.Data[toID]
	if !ok {
		return model.ErrNotFound
	}
	for _, col := range columns {
		switch col {
		case "created_at":
			to.CreatedAt = from.CreatedAt
		}
	}
	if m.CopyAttributesCalls == nil {
		m.CopyAttributesCalls = make(map[string]string)
	}
	m.CopyAttributesCalls[fromID] = toID
	return nil
}

// SetRating sets the rating for an album
func (m *MockAlbumRepo) SetRating(rating int, itemID string) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	return nil
}

// SetStar sets the starred status for albums
func (m *MockAlbumRepo) SetStar(starred bool, itemIDs ...string) error {
	if m.Err {
		return errors.New("unexpected error")
	}
	return nil
}

var _ model.AlbumRepository = (*MockAlbumRepo)(nil)
