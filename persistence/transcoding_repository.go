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

func (r *transcodingRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(ctx, options...))
}

func (r *transcodingRepository) Read(ctx context.Context, id string) (*model.Transcoding, error) {
	res, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if !loggedUser(ctx).IsAdmin {
		res.Command = ""
	}
	return res, nil
}

func (r *transcodingRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Transcoding, error) {
	sel := r.newSelect(r.parseRestOptions(ctx, options...)).Columns("*")
	res := model.Transcodings{}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	if !loggedUser(ctx).IsAdmin {
		for i := range res {
			res[i].Command = ""
		}
	}
	return res, nil
}

func (r *transcodingRepository) Save(ctx context.Context, t *model.Transcoding) (string, error) {
	if !loggedUser(ctx).IsAdmin {
		return "", rest.ErrPermissionDenied
	}
	return r.put(t.ID, t)
}

func (r *transcodingRepository) Update(ctx context.Context, id string, entity model.Transcoding, cols ...string) error {
	if !loggedUser(ctx).IsAdmin {
		return rest.ErrPermissionDenied
	}
	t := &entity
	t.ID = id
	_, err := r.put(id, t)
	return err
}

func (r *transcodingRepository) Delete(ctx context.Context, ids ...string) error {
	if !loggedUser(ctx).IsAdmin {
		return rest.ErrPermissionDenied
	}
	for _, id := range ids {
		if err := r.deleteByID(id); err != nil {
			return err
		}
	}
	return nil
}

var _ model.TranscodingRepository = (*transcodingRepository)(nil)
var _ rest.Repository[model.Transcoding] = (*transcodingRepository)(nil)
var _ rest.Persistable[model.Transcoding] = (*transcodingRepository)(nil)
