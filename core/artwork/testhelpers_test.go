package artwork

import (
	"github.com/navidrome/navidrome/model"
)

// fakeFolderRepo covers the three FolderRepository methods the resolvers reach for; only the
// folder listing varies per spec, so the other two answer as an unremarkable library does.
type fakeFolderRepo struct {
	model.FolderRepository
	result []model.Folder
	err    error
}

func (f *fakeFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) {
	return f.result, f.err
}

func (f *fakeFolderRepo) HasAudioOutsideFolders(model.Folder, []string) (bool, error) {
	return false, nil
}

func (f *fakeFolderRepo) Get(string) (*model.Folder, error) {
	return nil, model.ErrNotFound
}
