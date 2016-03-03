package persistence

import (
	"github.com/deluan/gosonic/domain"
	"sort"
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
	err := r.loadChildren("album", albumId, &mfs, "", false)
	sort.Sort(byTrackNumber(mfs))
	return mfs, err
}

type byTrackNumber []domain.MediaFile

func (a byTrackNumber) Len() int {
	return len(a)
}
func (a byTrackNumber) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a byTrackNumber) Less(i, j int) bool {
	return (a[i].DiscNumber*1000 + a[i].TrackNumber) < (a[j].DiscNumber*1000 + a[j].TrackNumber)
}

var _ domain.MediaFileRepository = (*mediaFileRepository)(nil)