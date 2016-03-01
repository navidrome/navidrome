package repositories

import (
	"github.com/deluan/gosonic/models"
)

type MediaFile struct {
	BaseRepository
}

func NewMediaFileRepository() *MediaFile {
	r := &MediaFile{}
	r.init("mediafile", &models.MediaFile{})
	return r
}

func (r *MediaFile) Put(m *models.MediaFile) error {
	return r.saveOrUpdate(m.Id, m)
}