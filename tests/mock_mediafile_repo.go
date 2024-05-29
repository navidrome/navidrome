package tests

import (
	"cmp"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

func CreateMockMediaFileRepo() *MockMediaFileRepo {
	return &MockMediaFileRepo{
		data: make(map[string]*model.MediaFile),
	}
}

type MockMediaFileRepo struct {
	model.MediaFileRepository
	data map[string]*model.MediaFile
	err  bool
}

func (m *MockMediaFileRepo) SetError(err bool) {
	m.err = err
}

func (m *MockMediaFileRepo) SetData(mfs model.MediaFiles) {
	m.data = make(map[string]*model.MediaFile)
	for i, mf := range mfs {
		m.data[mf.ID] = &mfs[i]
	}
}

func (m *MockMediaFileRepo) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("error")
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockMediaFileRepo) Get(id string) (*model.MediaFile, error) {
	if m.err {
		return nil, errors.New("error")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) GetAll(...model.QueryOptions) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("error")
	}
	values := slices.Collect(maps.Values(m.data))
	return slice.Map(values, func(p *model.MediaFile) model.MediaFile {
		return *p
	}), nil
}

func (m *MockMediaFileRepo) Put(mf *model.MediaFile) error {
	if m.err {
		return errors.New("error")
	}
	if mf.ID == "" {
		mf.ID = uuid.NewString()
	}
	m.data[mf.ID] = mf
	return nil
}

func (m *MockMediaFileRepo) Delete(id string) error {
	if m.err {
		return errors.New("error")
	}
	if _, ok := m.data[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.data, id)
	return nil
}

func (m *MockMediaFileRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.err {
		return errors.New("error")
	}
	if d, ok := m.data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

func (m *MockMediaFileRepo) FindByAlbum(artistId string) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("error")
	}
	var res = make(model.MediaFiles, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.AlbumID == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}

func (m *MockMediaFileRepo) GetMissingAndMatching(libId int, pagination ...model.QueryOptions) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("error")
	}
	var res model.MediaFiles
	for _, a := range m.data {
		if a.LibraryID == libId && a.Missing {
			res = append(res, *a)
		}
	}

	for _, a := range m.data {
		if a.LibraryID == libId && !(*a).Missing && slices.IndexFunc(res, func(mediaFile model.MediaFile) bool {
			return mediaFile.PID == a.PID
		}) != -1 {
			res = append(res, *a)
		}
	}
	slices.SortFunc(res, func(i, j model.MediaFile) int {
		return cmp.Compare(i.PID, j.PID)
	})

	return res, nil
}

var _ model.MediaFileRepository = (*MockMediaFileRepo)(nil)
