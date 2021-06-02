package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

type shareRepository struct {
	sqlRepository
	sqlRestful
}

func NewShareRepository(ctx context.Context, o orm.Ormer) model.ShareRepository {
	r := &shareRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "share"
	return r
}

func (r *shareRepository) Delete(id string) error {
	err := r.delete(Eq{"id": id})
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func (r *shareRepository) selectShare(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).Columns("*")
}

func (r *shareRepository) GetAll(options ...model.QueryOptions) (model.Shares, error) {
	sq := r.selectShare(options...)
	res := model.Shares{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *shareRepository) Put(s *model.Share) error {
	_, err := r.put(s.ID, s)
	return err
}

func (r *shareRepository) Update(entity interface{}, cols ...string) error {
	s := entity.(*model.Share)
	_, err := r.put(s.ID, s, cols...)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func (r *shareRepository) Save(entity interface{}) (string, error) {
	s := entity.(*model.Share)
	id, err := r.put(s.ID, s)
	if err == model.ErrNotFound {
		return "", rest.ErrNotFound
	}
	return id, err
}

func (r *shareRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.selectShare(), options...)
}

func (r *shareRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *shareRepository) EntityName() string {
	return "share"
}

func (r *shareRepository) NewInstance() interface{} {
	return &model.Share{}
}

func (r *shareRepository) Get(id string) (*model.Share, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Share
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *shareRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *shareRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

var _ model.ShareRepository = (*shareRepository)(nil)
var _ rest.Repository = (*shareRepository)(nil)
var _ rest.Persistable = (*shareRepository)(nil)
