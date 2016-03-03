package persistence

import (
	"github.com/deluan/gosonic/domain"
)

type mediaFileRepository struct {
	baseRepository
}

func NewMediaFileRepository() domain.MediaFileRepository {
	r := &mediaFileRepository{}
	r.init("mediafile", &domain.MediaFile{})
	return r
}

func (r *mediaFileRepository) Put(m *domain.MediaFile) error {
	return r.saveOrUpdate(m.Id, m)
}

func (r *mediaFileRepository) FindByAlbum(albumId string) ([]domain.MediaFile, error) {
	var mfs = make([]domain.MediaFile, 0)
	err := r.loadChildren("album", albumId, &mfs, "Track", false)
	return mfs, err
}
