package tests

import (
	"github.com/navidrome/navidrome/model"
)

type MockFolderRepo struct {
	Error error
	Data  map[string]model.Folder
}

func (r *MockFolderRepo) init() {
	if r.Data == nil {
		r.Data = make(map[string]model.Folder)
	}
}

func (r *MockFolderRepo) Get(id string) (*model.Folder, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	r.init()
	if f, ok := r.Data[id]; ok {
		return &f, nil
	}
	return nil, model.ErrNotFound
}

func (r *MockFolderRepo) GetByPath(lib model.Library, path string) (*model.Folder, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	r.init()
	for _, f := range r.Data {
		if f.LibraryID == lib.ID && f.Path == path {
			return &f, nil
		}
	}
	return nil, model.ErrNotFound
}

func (r *MockFolderRepo) GetAll(...model.QueryOptions) (model.Folders, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	r.init()
	var all model.Folders
	for _, f := range r.Data {
		all = append(all, f)
	}
	return all, nil
}

func (r *MockFolderRepo) CountAll(...model.QueryOptions) (int64, error) {
	if r.Error != nil {
		return 0, r.Error
	}
	r.init()
	return int64(len(r.Data)), nil
}

func (r *MockFolderRepo) GetFolderUpdateInfo(lib model.Library, targetPaths ...string) (map[string]model.FolderUpdateInfo, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	return make(map[string]model.FolderUpdateInfo), nil
}

func (r *MockFolderRepo) Put(f *model.Folder) error {
	if r.Error != nil {
		return r.Error
	}
	r.init()
	r.Data[f.ID] = *f
	return nil
}

func (r *MockFolderRepo) MarkMissing(missing bool, ids ...string) error {
	if r.Error != nil {
		return r.Error
	}
	r.init()
	for _, id := range ids {
		if f, ok := r.Data[id]; ok {
			f.Missing = missing
			r.Data[id] = f
		}
	}
	return nil
}

func (r *MockFolderRepo) GetTouchedWithPlaylists() (model.FolderCursor, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	return func(yield func(model.Folder, error) bool) {}, nil
}

var _ model.FolderRepository = (*MockFolderRepo)(nil)
