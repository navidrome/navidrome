package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type transcodingRepository struct {
	sqlRepository
}

func NewTranscodingRepository(ctx context.Context, db dbx.Builder) model.TranscodingRepository {
	r := &transcodingRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Transcoding{}, nil)
	return r
}

func (r *transcodingRepository) Get(id string) (*model.Transcoding, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Transcoding
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *transcodingRepository) CountAll(qo ...model.QueryOptions) (int64, error) {
	return r.count(Select(), qo...)
}

func (r *transcodingRepository) FindByFormat(format string) (*model.Transcoding, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"target_format": format})
	var res model.Transcoding
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *transcodingRepository) Put(t *model.Transcoding) error {
	if !loggedUser(r.ctx).IsAdmin {
		return rest.ErrPermissionDenied
	}
	_, err := r.put(t.ID, t)
	return err
}

func (r *transcodingRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(r.ctx, options...))
}

func (r *transcodingRepository) Read(id string) (any, error) {
	res, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if !loggedUser(r.ctx).IsAdmin {
		res.Command = ""
	}
	return res, nil
}

func (r *transcodingRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	sel := r.newSelect(r.parseRestOptions(r.ctx, options...)).Columns("*")
	res := model.Transcodings{}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	if !loggedUser(r.ctx).IsAdmin {
		for i := range res {
			res[i].Command = ""
		}
	}
	return res, nil
}

func (r *transcodingRepository) EntityName() string {
	return "transcoding"
}

func (r *transcodingRepository) NewInstance() any {
	return &model.Transcoding{}
}

func (r *transcodingRepository) Save(entity any) (string, error) {
	if !loggedUser(r.ctx).IsAdmin {
		return "", rest.ErrPermissionDenied
	}
	t := entity.(*model.Transcoding)
	return r.put(t.ID, t)
}

func (r *transcodingRepository) Update(id string, entity any, cols ...string) error {
	if !loggedUser(r.ctx).IsAdmin {
		return rest.ErrPermissionDenied
	}
	t := entity.(*model.Transcoding)
	t.ID = id
	_, err := r.put(id, t)
	return err
}

func (r *transcodingRepository) Delete(id string) error {
	if !loggedUser(r.ctx).IsAdmin {
		return rest.ErrPermissionDenied
	}
	return r.deleteByID(id)
}

var _ model.TranscodingRepository = (*transcodingRepository)(nil)
var _ rest.Repository = (*transcodingRepository)(nil)
var _ rest.Persistable = (*transcodingRepository)(nil)
