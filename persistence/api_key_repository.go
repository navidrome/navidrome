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
	return Eq{"p.user_id": user.ID}
}

func (r *apiKeyRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sq := r.selectAPIKey(options...).Where(r.userFilter())
	return r.count(sq, options...)
}

func (r *apiKeyRepository) Get(id string) (*model.APIKey, error) {
	sel := r.selectAPIKey().Where(And{Eq{"ak.id": id}})
	var res model.APIKey
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return &res, err
}

func (r *apiKeyRepository) selectAPIKey(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).
		From("api_key ak").
		LeftJoin("player p ON ak.player_id = p.id").
		Columns("ak.*")
}

func (r *apiKeyRepository) GetAll(options ...model.QueryOptions) (model.APIKeys, error) {
	sel := r.selectAPIKey().Where(r.userFilter())
	res := model.APIKeys{}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	return res, err
}

func (r *apiKeyRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *apiKeyRepository) Read(id string) (interface{}, error) {
	apiKey, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if err := r.VerifyPlayerAccess(apiKey.PlayerID); err != nil {
		return nil, err
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
	if err := r.VerifyPlayerAccess(ak.PlayerID); err != nil {
		return "", err
	}

	if ak.ID == "" {
		ak.ID = id.NewRandom()
	}
	ak.Key = generateAPIKey()
	ak.CreatedAt = time.Now()
	values, err := toSQLArgs(*ak)
	if err != nil {
		return "", err
	}

	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return ak.ID, err
}

func (r *apiKeyRepository) Update(id string, entity interface{}, _ ...string) error {
	ak := entity.(*model.APIKey)
	current, err := r.Get(id)
	if err != nil {
		return err
	}

	if err := r.VerifyPlayerAccess(current.PlayerID); err != nil {
		return err
	}

	update := Update(r.tableName).
		Set("name", ak.Name).
		Where(Eq{"id": id})
	_, err = r.executeSQL(update)
	return err
}

func (r *apiKeyRepository) Delete(id string) error {
	apiKey, err := r.Get(id)
	if err != nil {
		return err
	}

	if err := r.VerifyPlayerAccess(apiKey.PlayerID); err != nil {
		return err
	}

	return r.delete(Eq{"id": id})
}

func (r *apiKeyRepository) FindByKey(key string) (*model.APIKey, error) {
	sel := r.selectAPIKey().Where(And{Eq{"ak.key": key}})
	var res model.APIKey
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return &res, err
}

func (r *apiKeyRepository) RefreshKey(id string) (string, error) {
	apiKey, err := r.Get(id)
	if err != nil {
		return "", err
	}
	if err := r.VerifyPlayerAccess(apiKey.PlayerID); err != nil {
		return "", err
	}

	newKey := generateAPIKey()
	update := Update(r.tableName).
		Set("key", newKey).
		Where(Eq{"id": id})
	_, err = r.executeSQL(update)

	if err != nil {
		return "", err
	}

	return newKey, nil
}

func (r *apiKeyRepository) VerifyPlayerAccess(playerID string) error {
	if playerID == "" {
		return model.ErrNotFound
	}

	playerRepo := NewPlayerRepository(r.ctx, r.db)
	player, err := playerRepo.Get(playerID)
	if err != nil {
		return err
	}

	user := loggedUser(r.ctx)
	if !user.IsAdmin && player.UserId != user.ID {
		return rest.ErrPermissionDenied
	}

	return nil
}

func generateAPIKey() string {
	return "nav_" + id.NewRandom()
}

var _ model.APIKeyRepository = (*apiKeyRepository)(nil)
var _ rest.Repository = (*apiKeyRepository)(nil)
var _ rest.Persistable = (*apiKeyRepository)(nil)
