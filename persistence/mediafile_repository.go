package persistence

import (
	"context"
	"fmt"
	"os"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type mediaFileRepository struct {
	sqlRepository
	sqlRestful
}

func NewMediaFileRepository(ctx context.Context, o orm.Ormer) *mediaFileRepository {
	r := &mediaFileRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_file"
	r.sortMappings = map[string]string{
		"artist": "order_artist_name asc, album asc, disc_number asc, track_number asc",
		"album":  "order_album_name asc, disc_number asc, track_number asc",
		"random": "RANDOM()",
	}
	r.filterMappings = map[string]filterFunc{
		"title":   fullTextFilter,
		"starred": booleanFilter,
	}
	return r
}

func (r mediaFileRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.newSelectWithAnnotation("media_file.id"), options...)
}

func (r mediaFileRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r mediaFileRepository) Put(m *model.MediaFile) error {
	m.FullText = getFullText(m.Title, m.Album, m.Artist, m.AlbumArtist,
		m.SortTitle, m.SortAlbumName, m.SortArtistName, m.SortAlbumArtistName, m.DiscSubtitle)
	_, err := r.put(m.ID, m)
	return err
}

func (r mediaFileRepository) selectMediaFile(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation("media_file.id", options...).Columns("media_file.*")
}

func (r mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	sel := r.selectMediaFile().Where(Eq{"id": id})
	var res model.MediaFiles
	if err := r.queryAll(sel, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return &res[0], nil
}

func (r mediaFileRepository) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...)
	res := model.MediaFiles{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r mediaFileRepository) FindByAlbum(albumId string) (model.MediaFiles, error) {
	sel := r.selectMediaFile().Where(Eq{"album_id": albumId}).OrderBy("disc_number", "track_number")
	res := model.MediaFiles{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r mediaFileRepository) FindByPath(path string) (model.MediaFiles, error) {
	// Only return mediafiles that are direct child of requested path
	// Query by path based on https://stackoverflow.com/a/13911906/653632
	sel0 := r.selectMediaFile().Columns(fmt.Sprintf("substr(path, %d) AS item", len(path)+2)).
		Where(Like{"path": path + string(os.PathSeparator) + "%"})
	sel := r.newSelect().Columns("*", "item NOT GLOB '*"+string(os.PathSeparator)+"*' AS isLast").
		Where(Eq{"isLast": 1}).FromSelect(sel0, "sel0")

	res := model.MediaFiles{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r mediaFileRepository) GetStarred(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...).Where("starred = true")
	starred := model.MediaFiles{}
	err := r.queryAll(sq, &starred)
	return starred, err
}

// TODO Keep order when paginating
func (r mediaFileRepository) GetRandom(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...)
	sq = sq.OrderBy("RANDOM()")
	results := model.MediaFiles{}
	err := r.queryAll(sq, &results)
	return results, err
}

func (r mediaFileRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r mediaFileRepository) DeleteByPath(path string) error {
	del := Delete(r.tableName).
		Where(And{Like{"path": path + string(os.PathSeparator) + "%"},
			Eq{fmt.Sprintf("substr(path, %d) glob '*%s*'", len(path)+2, string(os.PathSeparator)): 0}})
	log.Debug(r.ctx, "Deleting mediafiles by path", "path", path)
	_, err := r.executeSQL(del)
	return err
}

func (r mediaFileRepository) Search(q string, offset int, size int) (model.MediaFiles, error) {
	results := model.MediaFiles{}
	err := r.doSearch(q, offset, size, &results, "title")
	return results, err
}

func (r mediaFileRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r mediaFileRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r mediaFileRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r mediaFileRepository) EntityName() string {
	return "mediafile"
}

func (r mediaFileRepository) NewInstance() interface{} {
	return &model.MediaFile{}
}

func (r mediaFileRepository) Save(entity interface{}) (string, error) {
	mf := entity.(*model.MediaFile)
	err := r.Put(mf)
	return mf.ID, err
}

func (r mediaFileRepository) Update(entity interface{}, cols ...string) error {
	mf := entity.(*model.MediaFile)
	return r.Put(mf)
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ model.ResourceRepository = (*mediaFileRepository)(nil)
var _ rest.Persistable = (*mediaFileRepository)(nil)
