package persistence

import (
	"github.com/deluan/gosonic/domain"
)

type MediaFile struct {
	BaseRepository
}

func NewMediaFileRepository() *MediaFile {
	r := &MediaFile{}
	r.init("mediafile", &domain.MediaFile{})
	return r
}

func (r *MediaFile) Put(m *domain.MediaFile) error {
	return r.saveOrUpdate(m.Id, m)
}