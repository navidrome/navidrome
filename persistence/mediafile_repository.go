package persistence

import (
	"context"
	"os"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type mediaFileRepository struct {
	sqlRepository
}

func NewMediaFileRepository(ctx context.Context, o orm.Ormer) *mediaFileRepository {
	r := &mediaFileRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_file"
	r.sortMappings = map[string]string{
		"artist": "artist asc, album asc, disc_number asc, track_number asc",
		"album":  "album asc, disc_number asc, track_number asc",
	}
	return r
}

func (r mediaFileRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r mediaFileRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r mediaFileRepository) Put(m *model.MediaFile) error {
	_, err := r.put(m.ID, m)
	if err != nil {
		return err
	}
	return r.index(m.ID, m.Title)
}

func (r mediaFileRepository) selectMediaFile(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation("media_file.id", options...).Columns("media_file.*")
}

func (r mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	sel := r.selectMediaFile().Where(Eq{"id": id})
	var res model.MediaFile
	err := r.queryOne(sel, &res)
	return &res, err
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
	sel := r.selectMediaFile().Where(Like{"path": path + "%"})
	res := model.MediaFiles{}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}

	// Only return mediafiles that are direct child of requested path
	filtered := model.MediaFiles{}
	path = strings.ToLower(path) + string(os.PathSeparator)
	for _, mf := range res {
		filename := strings.TrimPrefix(strings.ToLower(mf.Path), path)
		if len(strings.Split(filename, string(os.PathSeparator))) > 1 {
			continue
		}
		filtered = append(filtered, mf)
	}
	return filtered, nil
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
	switch r.ormer.Driver().Type() {
	case orm.DRMySQL:
		sq = sq.OrderBy("RAND()")
	default:
		sq = sq.OrderBy("RANDOM()")
	}
	results := model.MediaFiles{}
	err := r.queryAll(sq, &results)
	return results, err
}

func (r mediaFileRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r mediaFileRepository) DeleteByPath(path string) error {
	filtered, err := r.FindByPath(path)
	if err != nil {
		return err
	}
	if len(filtered) == 0 {
		return nil
	}
	ids := make([]string, len(filtered))
	for i, mf := range filtered {
		ids[i] = mf.ID
	}
	log.Debug(r.ctx, "Deleting mediafiles by path", "path", path, "totalDeleted", len(ids))
	del := Delete(r.tableName).Where(Eq{"id": ids})
	_, err = r.executeSQL(del)
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
	return model.MediaFile{}
}

var _ model.MediaFileRepository = (*mediaFileRepository)(nil)
var _ model.ResourceRepository = (*mediaFileRepository)(nil)
