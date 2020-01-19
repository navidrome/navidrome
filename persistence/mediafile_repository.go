package persistence

import (
	"os"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type mediaFile struct {
	ID          string    `orm:"pk;column(id)"`
	Path        string    `orm:"index"`
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
	StarredAt   time.Time `orm:"index;null"`
	CreatedAt   time.Time `orm:"null"`
	UpdatedAt   time.Time `orm:"null"`
}

type mediaFileRepository struct {
	searchableRepository
}

func NewMediaFileRepository(o orm.Ormer) model.MediaFileRepository {
	r := &mediaFileRepository{}
	r.ormer = o
	r.tableName = "media_file"
	return r
}

func (r *mediaFileRepository) Put(m *model.MediaFile, overrideAnnotation bool) error {
	tm := mediaFile(*m)
	if !overrideAnnotation {
		// Don't update media annotation fields (playcount, starred, etc..)
		return r.put(m.ID, m.Title, &tm, "path", "title", "album", "artist", "artist_id", "album_artist",
			"album_id", "has_cover_art", "track_number", "disc_number", "year", "size", "suffix", "duration",
			"bit_rate", "genre", "compilation", "updated_at")
	}
	return r.put(m.ID, m.Title, &tm)
}

func (r *mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	tm := mediaFile{ID: id}
	err := r.ormer.Read(&tm)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := model.MediaFile(tm)
	return &a, nil
}

func (r *mediaFileRepository) toMediaFiles(all []mediaFile) model.MediaFiles {
	result := make(model.MediaFiles, len(all))
	for i, m := range all {
		result[i] = model.MediaFile(m)
	}
	return result
}

func (r *mediaFileRepository) FindByAlbum(albumId string) (model.MediaFiles, error) {
	var mfs []mediaFile
	_, err := r.newQuery().Filter("album_id", albumId).OrderBy("disc_number", "track_number").All(&mfs)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(mfs), nil
}

func (r *mediaFileRepository) FindByPath(path string) (model.MediaFiles, error) {
	var mfs []mediaFile
	_, err := r.newQuery().Filter("path__istartswith", path).OrderBy("disc_number", "track_number").All(&mfs)
	if err != nil {
		return nil, err
	}
	var filtered []mediaFile
	path = strings.ToLower(path) + string(os.PathSeparator)
	for _, mf := range mfs {
		filename := strings.TrimPrefix(strings.ToLower(mf.Path), path)
		if len(strings.Split(filename, string(os.PathSeparator))) > 1 {
			continue
		}
		filtered = append(filtered, mf)
	}
	return r.toMediaFiles(filtered), nil
}

func (r *mediaFileRepository) DeleteByPath(path string) error {
	var mfs []mediaFile
	// TODO Paginate this (and all other situations similar)
	_, err := r.newQuery().Filter("path__istartswith", path).OrderBy("disc_number", "track_number").All(&mfs)
	if err != nil {
		return err
	}
	var filtered []string
	path = strings.ToLower(path) + string(os.PathSeparator)
	for _, mf := range mfs {
		filename := strings.TrimPrefix(strings.ToLower(mf.Path), path)
		if len(strings.Split(filename, string(os.PathSeparator))) > 1 {
			continue
		}
		filtered = append(filtered, mf.ID)
	}
	if len(filtered) == 0 {
		return nil
	}
	_, err = r.newQuery().Filter("id__in", filtered).Delete()
	return err
}

func (r *mediaFileRepository) GetStarred(options ...model.QueryOptions) (model.MediaFiles, error) {
	var starred []mediaFile
	_, err := r.newQuery(options...).Filter("starred", true).All(&starred)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(starred), nil
}

func (r *mediaFileRepository) SetStar(starred bool, ids ...string) error {
	if len(ids) == 0 {
		return model.ErrNotFound
	}
	var starredAt time.Time
	if starred {
		starredAt = time.Now()
	}
	_, err := r.newQuery().Filter("id__in", ids).Update(orm.Params{
		"starred":    starred,
		"starred_at": starredAt,
	})
	return err
}

func (r *mediaFileRepository) SetRating(rating int, ids ...string) error {
	if len(ids) == 0 {
		return model.ErrNotFound
	}
	_, err := r.newQuery().Filter("id__in", ids).Update(orm.Params{"rating": rating})
	return err
}

func (r *mediaFileRepository) MarkAsPlayed(id string, playDate time.Time) error {
	_, err := r.newQuery().Filter("id", id).Update(orm.Params{
		"play_count": orm.ColValue(orm.ColAdd, 1),
		"play_date":  playDate,
	})
	return err
}

func (r *mediaFileRepository) PurgeInactive(activeList model.MediaFiles) error {
	_, err := r.purgeInactive(activeList, func(item interface{}) string {
		return item.(model.MediaFile).ID
	})
	return err
}

func (r *mediaFileRepository) Search(q string, offset int, size int) (model.MediaFiles, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []mediaFile
	err := r.doSearch(r.tableName, q, offset, size, &results, "rating desc", "starred desc", "play_count desc", "title")
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(results), nil
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ = model.MediaFile(mediaFile{})
