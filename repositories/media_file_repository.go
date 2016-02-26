package repositories

import (
	"github.com/deluan/gosonic/models"
	"fmt"
	"crypto/md5"
)

type MediaFile struct {
	BaseRepository
}

func NewMediaFileRepository() *MediaFile {
	r := &MediaFile{}
	r.col = createCollection("MediaFiles")
	return r
}

func (r *MediaFile) Add(m *models.MediaFile) error {
	if m.Id == "" {
		m.Id = fmt.Sprintf("%x", md5.Sum([]byte(m.Path)))
	}
	return r.saveOrUpdate(m)
}