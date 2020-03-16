package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type transcodingRepository struct {
	sqlRepository
}

func NewTranscodingRepository(ctx context.Context, o orm.Ormer) model.TranscodingRepository {
	r := &transcodingRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "transcoding"
	return r
}

func (r *transcodingRepository) Get(id string) (*model.Transcoding, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Transcoding
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *transcodingRepository) Put(t *model.Transcoding) error {
	_, err := r.put(t.ID, t)
	return err
}

func (r *transcodingRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(options...))
}

func (r *transcodingRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *transcodingRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).Columns("*")
	res := model.Transcodings{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *transcodingRepository) EntityName() string {
	return "transcoding"
}

func (r *transcodingRepository) NewInstance() interface{} {
	return &model.Transcoding{}
}

func (r *transcodingRepository) Save(entity interface{}) (string, error) {
	t := entity.(*model.Transcoding)
	id, err := r.put(t.ID, t)
	if err == model.ErrNotFound {
		return "", rest.ErrNotFound
	}
	return id, err
}

func (r *transcodingRepository) Update(entity interface{}, cols ...string) error {
	t := entity.(*model.Transcoding)
	_, err := r.put(t.ID, t)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func (r *transcodingRepository) Delete(id string) error {
	err := r.delete(Eq{"id": id})
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

var _ model.TranscodingRepository = (*transcodingRepository)(nil)
var _ rest.Repository = (*transcodingRepository)(nil)
var _ rest.Persistable = (*transcodingRepository)(nil)
