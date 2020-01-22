package persistence

import (
	"os"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type mediaFile struct {
	ID          string    `json:"id"            orm:"pk;column(id)"`
	Path        string    `json:"path"          orm:"index"`
	Title       string    `json:"title"         orm:"index"`
	Album       string    `json:"album"`
	Artist      string    `json:"artist"`
	ArtistID    string    `json:"artistId"      orm:"column(artist_id)"`
	AlbumArtist string    `json:"albumArtist"`
	AlbumID     string    `json:"albumId"       orm:"column(album_id);index"`
	HasCoverArt bool      `json:"-"`
	TrackNumber int       `json:"trackNumber"`
	DiscNumber  int       `json:"discNumber"`
	Year        int       `json:"year"`
	Size        int       `json:"size"`
	Suffix      string    `json:"suffix"`
	Duration    int       `json:"duration"`
	BitRate     int       `json:"bitRate"`
	Genre       string    `json:"genre"         orm:"index"`
	Compilation bool      `json:"compilation"`
	CreatedAt   time.Time `json:"createdAt"     orm:"null"`
	UpdatedAt   time.Time `json:"updatedAt"     orm:"null"`
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

func (r *mediaFileRepository) Put(m *model.MediaFile) error {
	tm := mediaFile(*m)
	// Don't update media annotation fields (playcount, starred, etc..)
	// TODO Validate if this is still necessary, now that we don't have annotations in the mediafile model
	return r.put(m.ID, m.Title, &tm, "path", "title", "album", "artist", "artist_id", "album_artist",
		"album_id", "has_cover_art", "track_number", "disc_number", "year", "size", "suffix", "duration",
		"bit_rate", "genre", "compilation", "updated_at")
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

func (r *mediaFileRepository) GetRandom(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.newRawQuery(options...)
	switch r.ormer.Driver().Type() {
	case orm.DRMySQL:
		sq = sq.OrderBy("RAND()")
	default:
		sq = sq.OrderBy("RANDOM()")
	}
	sql, args, err := sq.ToSql()
	if err != nil {
		return nil, err
	}
	var results []mediaFile
	_, err = r.ormer.Raw(sql, args...).QueryRows(&results)
	return r.toMediaFiles(results), err
}

func (r *mediaFileRepository) GetStarred(userId string, options ...model.QueryOptions) (model.MediaFiles, error) {
	var starred []mediaFile
	sq := r.newRawQuery(options...).Join("annotation").Where("annotation.item_id = " + r.tableName + ".id")
	sq = sq.Where(squirrel.And{
		squirrel.Eq{"annotation.user_id": userId},
		squirrel.Eq{"annotation.starred": true},
	})
	sql, args, err := sq.ToSql()
	if err != nil {
		return nil, err
	}
	_, err = r.ormer.Raw(sql, args...).QueryRows(&starred)
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(starred), nil
}

func (r *mediaFileRepository) Search(q string, offset int, size int) (model.MediaFiles, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []mediaFile
	err := r.doSearch(r.tableName, q, offset, size, &results, "title")
	if err != nil {
		return nil, err
	}
	return r.toMediaFiles(results), nil
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ = model.MediaFile(mediaFile{})
