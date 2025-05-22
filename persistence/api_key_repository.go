package persistence

import (
	"context"
	"time"

	"github.com/deluan/rest"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type apiKeyRepository struct {
	sqlRepository
}

func NewAPIKeyRepository(ctx context.Context, db dbx.Builder) model.APIKeyRepository {
	r := &apiKeyRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.APIKey{}, nil)
	return r
}

func (r *apiKeyRepository) userFilter() Sqlizer {
	user := loggedUser(r.ctx)
	if user.IsAdmin {
		return And{}
	}
	return Eq{"user_id": user.ID}
}

func (r *apiKeyRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sq := Select().From(r.tableName).Where(r.userFilter())
	return r.count(sq, options...)
}

func (r *apiKeyRepository) Get(id string) (*model.APIKey, error) {
	sel := r.newSelect().Columns("*").Where(And{Eq{"id": id}})
	var res model.APIKey
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return &res, err
}

func (r *apiKeyRepository) GetAll(options ...model.QueryOptions) (model.APIKeys, error) {
	sel := r.newSelect(options...).Columns("*").Where(r.userFilter())
	res := model.APIKeys{}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	return res, err
}

func (r *apiKeyRepository) Put(ak *model.APIKey) error {
	if ak.ID == "" {
		ak.ID = id.NewRandom()
	}
	ak.CreatedAt = time.Now()
	values, err := toSQLArgs(*ak)
	if err != nil {
		return err
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r *apiKeyRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *apiKeyRepository) Read(id string) (interface{}, error) {
	user := loggedUser(r.ctx)
	apiKey, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if !user.IsAdmin && apiKey.UserID != user.ID {
		return nil, rest.ErrPermissionDenied
	}
	return apiKey, err
}

func (r *apiKeyRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *apiKeyRepository) EntityName() string {
	return "apikey"
}

func (r *apiKeyRepository) NewInstance() interface{} {
	return &model.APIKey{}
}

func (r *apiKeyRepository) Save(entity interface{}) (string, error) {
	ak := entity.(*model.APIKey)
	user := loggedUser(r.ctx)
	ak.UserID = user.ID
	// prefix API keys with nav_
	ak.Key = "nav_" + id.NewRandom()
	err := r.Put(ak)
	if err != nil {
		return "", err
	}
	return ak.ID, err
}

func (r *apiKeyRepository) Update(id string, entity interface{}, _ ...string) error {
	ak := entity.(*model.APIKey)
	current, err := r.Get(id)
	if err != nil {
		return err
	}
	user := loggedUser(r.ctx)
	if !user.IsAdmin && current.UserID != user.ID {
		return rest.ErrPermissionDenied
	}

	// Only allow updating name
	update := Update(r.tableName).
		Set("name", ak.Name).
		Where(Eq{"id": id})
	_, err = r.executeSQL(update)
	return err
}

func (r *apiKeyRepository) Delete(id string) error {
	user := loggedUser(r.ctx)
	apiKey, err := r.Get(id)
	if err != nil {
		return err
	}
	if !user.IsAdmin && apiKey.UserID != user.ID {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

func (r *apiKeyRepository) FindByKey(key string) (*model.APIKey, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"key": key})
	var res model.APIKey
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return &res, err
}

var _ model.APIKeyRepository = (*apiKeyRepository)(nil)
var _ rest.Repository = (*apiKeyRepository)(nil)
var _ rest.Persistable = (*apiKeyRepository)(nil)
