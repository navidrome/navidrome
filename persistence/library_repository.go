package persistence

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type libraryRepository struct {
	ctx context.Context
}

func NewLibraryRepository(ctx context.Context, _ dbx.Builder) model.LibraryRepository {
	return &libraryRepository{ctx}
}

func (r *libraryRepository) Get(int32) (*model.Library, error) {
	library := hardCoded()
	return &library, nil
}

func (*libraryRepository) GetAll() (model.Libraries, error) {
	return model.Libraries{hardCoded()}, nil
}

func hardCoded() model.Library {
	library := model.Library{ID: 0, Path: conf.Server.MusicFolder}
	library.Name = "Music Library"
	return library
}

var _ model.LibraryRepository = (*libraryRepository)(nil)
