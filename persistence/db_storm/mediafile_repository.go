package db_storm

import (
	"time"

	"github.com/asdine/storm/q"
	"github.com/cloudsonic/sonic-server/domain"
)

type _MediaFile struct {
	ID          string    ``
	Path        string    ``
	Title       string    ``
	Album       string    ``
	Artist      string    ``
	ArtistID    string    ``
	AlbumArtist string    ``
	AlbumID     string    `storm:"index"`
	HasCoverArt bool      ``
	TrackNumber int       ``
	DiscNumber  int       ``
	Year        int       ``
	Size        string    ``
	Suffix      string    ``
	Duration    int       ``
	BitRate     int       ``
	Genre       string    ``
	Compilation bool      ``
	PlayCount   int       ``
	PlayDate    time.Time ``
	Rating      int       ``
	Starred     bool      `storm:"index"`
	StarredAt   time.Time ``
	CreatedAt   time.Time ``
	UpdatedAt   time.Time ``
}

type mediaFileRepository struct {
	stormRepository
}

func NewMediaFileRepository() domain.MediaFileRepository {
	r := &mediaFileRepository{}
	r.init(&_MediaFile{})
	return r
}

func (r *mediaFileRepository) Put(m *domain.MediaFile) error {
	tm := _MediaFile(*m)
	return Db().Save(&tm)
}

func (r *mediaFileRepository) Get(id string) (*domain.MediaFile, error) {
	tm := &_MediaFile{}
	err := r.getByID(id, tm)
	if err != nil {
		return nil, err
	}
	a := domain.MediaFile(*tm)
	return &a, nil
}

func (r *mediaFileRepository) toMediaFiles(all []_MediaFile) (domain.MediaFiles, error) {
	result := make(domain.MediaFiles, len(all))
	for i, m := range all {
		result[i] = domain.MediaFile(m)
	}
	return result, nil
}

func (r *mediaFileRepository) FindByAlbum(albumId string) (domain.MediaFiles, error) {
	var mfs []_MediaFile
	err := r.execute(q.Eq("AlbumID", albumId), &mfs)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(mfs)
}

func (r *mediaFileRepository) GetStarred(options domain.QueryOptions) (domain.MediaFiles, error) {
	var starred []_MediaFile
	err := r.execute(q.Eq("Starred", true), &starred, &options)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(starred)
}

func (r *mediaFileRepository) GetAllIds() ([]string, error) {
	var all []_MediaFile
	err := r.getAll(&all, &domain.QueryOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]string, len(all))
	for i, m := range all {
		result[i] = domain.MediaFile(m).ID
	}
	return result, nil
}

func (r *mediaFileRepository) PurgeInactive(activeList domain.MediaFiles) ([]string, error) {
	return r.purgeInactive(activeList)
}

var _ domain.MediaFileRepository = (*mediaFileRepository)(nil)
var _ = domain.MediaFile(_MediaFile{})
