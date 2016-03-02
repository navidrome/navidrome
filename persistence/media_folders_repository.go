package persistence

import (
	"github.com/deluan/gosonic/domain"
	"github.com/astaxie/beego"
)

type mediaFolderRepository struct {
	domain.MediaFolderRepository
}

func NewMediaFolderRepository() domain.MediaFolderRepository {
	return &mediaFolderRepository{}
}


func (*mediaFolderRepository) GetAll() ([]domain.MediaFolder, error) {
	mediaFolder := domain.MediaFolder{Id: "0", Name: "iTunes Library", Path: beego.AppConfig.String("musicFolder")}
	result := make([]domain.MediaFolder, 1)
	result[0] = mediaFolder
	return result, nil
}