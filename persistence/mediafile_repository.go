package persistence

import (
	"errors"
	"sort"

	"github.com/deluan/gosonic/domain"
)

type mediaFileRepository struct {
	ledisRepository
}

func NewMediaFileRepository() domain.MediaFileRepository {
	r := &mediaFileRepository{}
	r.init("mediafile", &domain.MediaFile{})
	return r
}

func (r *mediaFileRepository) Put(m *domain.MediaFile) error {
	if m.Id == "" {
		return errors.New("mediaFile Id is not set")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *mediaFileRepository) Get(id string) (*domain.MediaFile, error) {
	m, err := r.readEntity(id)
	if err != nil {
		return nil, err
	}
	mf := m.(*domain.MediaFile)
	if mf.Id != id {
		return nil, nil
	}
	return mf, nil
}

func (r *mediaFileRepository) FindByAlbum(albumId string) (domain.MediaFiles, error) {
	var mfs = make(domain.MediaFiles, 0)
	err := r.loadChildren("album", albumId, &mfs, domain.QueryOptions{SortBy: "TrackNumber"})
	sort.Sort(mfs)
	return mfs, err
}

func (r *mediaFileRepository) PurgeInactive(active domain.MediaFiles) ([]string, error) {
	return r.purgeInactive(active, func(e interface{}) string {
		return e.(domain.MediaFile).Id
	})
}

var _ domain.MediaFileRepository = (*mediaFileRepository)(nil)
