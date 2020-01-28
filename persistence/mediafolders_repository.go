package persistence

import (
	"context"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/model"
)

type mediaFolderRepository struct {
	ctx context.Context
}

func NewMediaFolderRepository(ctx context.Context, o orm.Ormer) model.MediaFolderRepository {
	return &mediaFolderRepository{ctx}
}

func (*mediaFolderRepository) GetAll() (model.MediaFolders, error) {
	mediaFolder := model.MediaFolder{ID: "0", Path: conf.Server.MusicFolder}
	mediaFolder.Name = "Music Library"
	result := make(model.MediaFolders, 1)
	result[0] = mediaFolder
	return result, nil
}

var _ model.MediaFolderRepository = (*mediaFolderRepository)(nil)
