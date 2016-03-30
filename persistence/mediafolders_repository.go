package persistence

import (
	"github.com/deluan/gosonic/conf"
	"github.com/deluan/gosonic/domain"
)

type mediaFolderRepository struct {
	domain.MediaFolderRepository
}

func NewMediaFolderRepository() domain.MediaFolderRepository {
	return &mediaFolderRepository{}
}

func (*mediaFolderRepository) GetAll() (domain.MediaFolders, error) {
	mediaFolder := domain.MediaFolder{Id: "0", Name: "iTunes Library", Path: conf.GoSonic.MusicFolder}
	result := make(domain.MediaFolders, 1)
	result[0] = mediaFolder
	return result, nil
}

var _ domain.MediaFolderRepository = (*mediaFolderRepository)(nil)
