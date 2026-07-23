package artwork

import (
	"github.com/navidrome/navidrome/model"
)

type fakeFolderRepo struct {
	model.FolderRepository
	result       []model.Folder
	parentResult *model.Folder
	getErr       error
	getCallCount int
	err          error
	// hasOtherAudio is returned by HasAudioOutsideFolders (the album-root
	// check). False means the parent qualifies as an album root.
	hasOtherAudio bool
	otherAudioErr error
}

func (f *fakeFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) {
	return f.result, f.err
}

func (f *fakeFolderRepo) HasAudioOutsideFolders(model.Folder, []string) (bool, error) {
	return f.hasOtherAudio, f.otherAudioErr
}

func (f *fakeFolderRepo) Get(id string) (*model.Folder, error) {
	f.getCallCount++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.parentResult != nil {
		return f.parentResult, nil
	}
	return nil, model.ErrNotFound
}
