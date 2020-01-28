package persistence

import (
	"context"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
	"github.com/kennygrant/sanitize"
)

type mediaFileRepository struct {
	sqlRepository
}

func NewMediaFileRepository(ctx context.Context, o orm.Ormer) *mediaFileRepository {
	r := &mediaFileRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_file"
	return r
}

func (r mediaFileRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r mediaFileRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r mediaFileRepository) Put(m *model.MediaFile) error {
	values, _ := toSqlArgs(*m)
	update := Update(r.tableName).Where(Eq{"id": m.ID}).SetMap(values)
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r mediaFileRepository) selectMediaFile(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation(model.MediaItemType, "media_file.id", options...).Columns("media_file.*")
}

func (r mediaFileRepository) Get(id string) (*model.MediaFile, error) {
	sel := r.selectMediaFile().Where(Eq{"id": id})
	var res model.MediaFile
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r mediaFileRepository) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...)
	var res model.MediaFiles
	err := r.queryAll(sq, &res)
	return res, err
}

func (r mediaFileRepository) FindByAlbum(albumId string) (model.MediaFiles, error) {
	sel := r.selectMediaFile().Where(Eq{"album_id": albumId})
	var res model.MediaFiles
	err := r.queryAll(sel, &res)
	return res, err
}

func (r mediaFileRepository) FindByPath(path string) (model.MediaFiles, error) {
	sel := r.selectMediaFile().Where(Like{"path": path + "%"})
	var res model.MediaFiles
	err := r.queryAll(sel, &res)
	return res, err
}

func (r mediaFileRepository) GetStarred(userId string, options ...model.QueryOptions) (model.MediaFiles, error) {
	sq := r.selectMediaFile(options...).Where("starred = true")
	var starred model.MediaFiles
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
	sql, args, err := r.toSql(sq)
	if err != nil {
		return nil, err
	}
	var results model.MediaFiles
	_, err = r.ormer.Raw(sql, args...).QueryRows(&results)
	return results, err
}

func (r mediaFileRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r mediaFileRepository) DeleteByPath(path string) error {
	del := Delete(r.tableName).Where(Like{"path": path + "%"})
	_, err := r.executeSQL(del)
	return err
}

func (r mediaFileRepository) Search(q string, offset int, size int) (model.MediaFiles, error) {
	q = strings.TrimSpace(sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*"))))
	if len(q) <= 2 {
		return model.MediaFiles{}, nil
	}
	sq := Select("*").From(r.tableName)
	sq = sq.Limit(uint64(size)).Offset(uint64(offset)).OrderBy("title")
	sq = sq.Join("search").Where("search.id = " + r.tableName + ".id")
	parts := strings.Split(q, " ")
	for _, part := range parts {
		sq = sq.Where(Or{
			Like{"full_text": part + "%"},
			Like{"full_text": "%" + part + "%"},
		})
	}
	sql, args, err := r.toSql(sq)
	if err != nil {
		return nil, err
	}
	var results model.MediaFiles
	_, err = r.ormer.Raw(sql, args...).QueryRows(results)
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
