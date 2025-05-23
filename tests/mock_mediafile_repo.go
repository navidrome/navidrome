package tests

import (
	"cmp"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils/slice"
)

func CreateMockMediaFileRepo() *MockMediaFileRepo {
	return &MockMediaFileRepo{
		Data: make(map[string]*model.MediaFile),
	}
}

type MockMediaFileRepo struct {
	model.MediaFileRepository
	Data map[string]*model.MediaFile
	Err  bool
	// Add fields and methods for controlling CountAll and DeleteAllMissing in tests
	CountAllValue         int64
	CountAllOptions       model.QueryOptions
	DeleteAllMissingValue int64
}

func (m *MockMediaFileRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockMediaFileRepo) SetData(mfs model.MediaFiles) {
	m.Data = make(map[string]*model.MediaFile)
	for i, mf := range mfs {
		m.Data[mf.ID] = &mfs[i]
	}
}

func (m *MockMediaFileRepo) Exists(id string) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	_, found := m.Data[id]
	return found, nil
}

func (m *MockMediaFileRepo) Get(id string) (*model.MediaFile, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		// Intentionally clone the file and remove participants. This should
		// catch any caller that actually means to call GetWithParticipants
		res := *d
		res.Participants = model.Participants{}
		return &res, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) GetWithParticipants(id string) (*model.MediaFile, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) GetAll(...model.QueryOptions) (model.MediaFiles, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	values := slices.Collect(maps.Values(m.Data))
	return slice.Map(values, func(p *model.MediaFile) model.MediaFile {
		return *p
	}), nil
}

func (m *MockMediaFileRepo) Put(mf *model.MediaFile) error {
	if m.Err {
		return errors.New("error")
	}
	if mf.ID == "" {
		mf.ID = id.NewRandom()
	}
	m.Data[mf.ID] = mf
	return nil
}

func (m *MockMediaFileRepo) Delete(id string) error {
	if m.Err {
		return errors.New("error")
	}
	if _, ok := m.Data[id]; !ok {
		return model.ErrNotFound
	}
	delete(m.Data, id)
	return nil
}

func (m *MockMediaFileRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.Err {
		return errors.New("error")
	}
	if d, ok := m.Data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

func (m *MockMediaFileRepo) FindByAlbum(artistId string) (model.MediaFiles, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res = make(model.MediaFiles, len(m.Data))
	i := 0
	for _, a := range m.Data {
		if a.AlbumID == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}

func (m *MockMediaFileRepo) GetMissingAndMatching(libId int) (model.MediaFileCursor, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	var res model.MediaFiles
	for _, a := range m.Data {
		if a.LibraryID == libId && a.Missing {
			res = append(res, *a)
		}
	}

	for _, a := range m.Data {
		if a.LibraryID == libId && !(*a).Missing && slices.IndexFunc(res, func(mediaFile model.MediaFile) bool {
			return mediaFile.PID == a.PID
		}) != -1 {
			res = append(res, *a)
		}
	}
	slices.SortFunc(res, func(i, j model.MediaFile) int {
		return cmp.Or(
			cmp.Compare(i.PID, j.PID),
			cmp.Compare(i.ID, j.ID),
		)
	})

	return func(yield func(model.MediaFile, error) bool) {
		for _, a := range res {
			if !yield(a, nil) {
				break
			}
		}
	}, nil
}

func (m *MockMediaFileRepo) CountAll(opts ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	if m.CountAllValue != 0 {
		if len(opts) > 0 {
			m.CountAllOptions = opts[0]
		}
		return m.CountAllValue, nil
	}
	return int64(len(m.Data)), nil
}

func (m *MockMediaFileRepo) DeleteAllMissing() (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	if m.DeleteAllMissingValue != 0 {
		return m.DeleteAllMissingValue, nil
	}
	// Remove all missing files from Data
	var count int64
	for id, mf := range m.Data {
		if mf.Missing {
			delete(m.Data, id)
			count++
		}
	}
	return count, nil
}

var _ model.MediaFileRepository = (*MockMediaFileRepo)(nil)
