package persistence

import (
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type MediaFile struct {
	ID          string    `orm:"pk;column(id)"`
	Path        string    ``
	Title       string    `orm:"index"`
	Album       string    ``
	Artist      string    ``
	ArtistID    string    `orm:"column(artist_id)"`
	AlbumArtist string    ``
	AlbumID     string    `orm:"column(album_id);index"`
	HasCoverArt bool      ``
	TrackNumber int       ``
	DiscNumber  int       ``
	Year        int       ``
	Size        string    ``
	Suffix      string    ``
	Duration    int       ``
	BitRate     int       ``
	Genre       string    `orm:"index"`
	Compilation bool      ``
	PlayCount   int       `orm:"index"`
	PlayDate    time.Time `orm:"null"`
	Rating      int       `orm:"index"`
	Starred     bool      `orm:"index"`
	StarredAt   time.Time `orm:"null"`
	CreatedAt   time.Time `orm:"null"`
	UpdatedAt   time.Time `orm:"null"`
}

type mediaFileRepository struct {
	searchableRepository
}

func NewMediaFileRepository() model.MediaFileRepository {
	r := &mediaFileRepository{}
	r.tableName = "media_file"
	return r
}

func (r *mediaFileRepository) Put(m *model.MediaFile) error {
	tm := MediaFile(*m)
	return withTx(func(o orm.Ormer) error {
		return r.put(o, m.ID, m.Title, &tm)
	})
}

func (r *mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	tm := MediaFile{ID: id}
	err := Db().Read(&tm)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := model.MediaFile(tm)
	return &a, nil
}

func (r *mediaFileRepository) toMediaFiles(all []MediaFile) model.MediaFiles {
	result := make(model.MediaFiles, len(all))
	for i, m := range all {
		result[i] = model.MediaFile(m)
	}
	return result
}

func (r *mediaFileRepository) FindByAlbum(albumId string) (model.MediaFiles, error) {
	var mfs []MediaFile
	_, err := r.newQuery(Db()).Filter("album_id", albumId).OrderBy("disc_number", "track_number").All(&mfs)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(mfs), nil
}

func (r *mediaFileRepository) GetStarred(options ...model.QueryOptions) (model.MediaFiles, error) {
	var starred []MediaFile
	_, err := r.newQuery(Db(), options...).Filter("starred", true).All(&starred)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(starred), nil
}

func (r *mediaFileRepository) PurgeInactive(activeList model.MediaFiles) error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.purgeInactive(o, activeList, func(item interface{}) string {
			return item.(model.MediaFile).ID
		})
		return err
	})
}

func (r *mediaFileRepository) Search(q string, offset int, size int) (model.MediaFiles, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []MediaFile
	err := r.doSearch(r.tableName, q, offset, size, &results, "rating desc", "starred desc", "play_count desc", "title")
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(results), nil
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ = model.MediaFile(MediaFile{})
