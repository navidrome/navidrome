package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/model"
)

type shareRepository struct {
	sqlRepository
}

func NewShareRepository(ctx context.Context, o orm.Ormer) model.ShareRepository {
	r := &shareRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "share"
	return r
}

func (r *shareRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r *shareRepository) selectShare(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation("share.id", options...).Columns("*")
}

func (r *shareRepository) GetAll(options ...model.QueryOptions) (model.Shares, error) {
	sq := r.selectShare(options...)
	res := model.Shares{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *shareRepository) Put(s *model.Share) (*model.Share, error) {
	s.CreatedAt = time.Now()
	id, err := r.put(s.ID, s)
	if err != nil {
		return nil, err
	}
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Share
	err = r.queryOne(sel, &res)
	return &res, err
}

func (r *shareRepository) Update(s *model.Share) error {
	_, err := r.put(s.ID, s)
	return err
}
