package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type playerRepository struct {
	sqlRepository
}

func NewPlayerRepository(ctx context.Context, db dbx.Builder) model.PlayerRepository {
	r := &playerRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Player{}, map[string]filterFunc{
		"name": containsFilter("player.name"),
	})
	r.setSortMappings(map[string]string{
		"user_name": "username", //TODO rename all user_name and userName to username
	})
	return r
}

func (r *playerRepository) Put(p *model.Player) error {
	_, err := r.put(p.ID, p)
	return err
}

func (r *playerRepository) selectPlayer(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).
		Columns("player.*", "player.api_key_hash is not null as has_api_key").
		Join("user ON player.user_id = user.id").
		Columns("user.user_name username")
}

func (r *playerRepository) Get(id string) (*model.Player, error) {
	sel := r.selectPlayer().Where(Eq{"player.id": id})
	var res model.Player
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *playerRepository) FindMatch(userId, client, userAgent string) (*model.Player, error) {
	sel := r.selectPlayer().Where(And{
		Eq{"client": client},
		Eq{"user_agent": userAgent},
		Eq{"user_id": userId},
	})
	var res model.Player
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *playerRepository) newRestSelect(options ...model.QueryOptions) SelectBuilder {
	s := r.selectPlayer(options...)
	return s.Where(r.addRestriction())
}

func (r *playerRepository) CountByClient(options ...model.QueryOptions) (map[string]int64, error) {
	sel := r.newSelect(options...).
		Columns(
			"case when client = 'NavidromeUI' then name else client end as player",
			"count(*) as count",
		).GroupBy("client")
	var res []struct {
		Player string
		Count  int64
	}
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(res))
	for _, c := range res {
		counts[c.Player] = c.Count
	}
	return counts, nil
}

func (r *playerRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.newRestSelect(), options...)
}

func (r *playerRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *playerRepository) Read(id string) (any, error) {
	sel := r.newRestSelect().Where(Eq{"player.id": id})
	var res model.Player
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *playerRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	sel := r.newRestSelect(r.parseRestOptions(r.ctx, options...))
	res := model.Players{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *playerRepository) EntityName() string {
	return "player"
}

func (r *playerRepository) NewInstance() any {
	return &model.Player{}
}

// isPermitted authorizes creating a new record, based on the owner declared in the request body.
// This is only safe for inserts: there is no stored row yet, and a non-admin may only create a
// player they own. Updates must not use this (the body owner is attacker-controlled); they go
// through updateOwned, which authorizes against the persisted user_id in the WHERE clause.
func (r *playerRepository) isPermitted(p *model.Player) bool {
	u := loggedUser(r.ctx)
	return u.IsAdmin || p.UserId == u.ID
}

func (r *playerRepository) Save(entity any) (string, error) {
	t := entity.(*model.Player)
	if u := loggedUser(r.ctx); t.UserId == "" && u.ID != invalidUserId {
		t.UserId = u.ID
	}
	if !r.isPermitted(t) {
		return "", rest.ErrPermissionDenied
	}
	return r.put("", t) // Save only creates; edits go through the owner-scoped Update
}

func (r *playerRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.Player)
	t.ID = id
	return r.updateOwned(id, t, cols...)
}

func (r *playerRepository) Delete(id string) error {
	return r.deleteOwned(id)
}

// Keys are 128-bit random values, so a fast unsalted hash is enough and allows an indexed lookup.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (r *playerRepository) FindByAPIKey(key string) (*model.Player, error) {
	if key == "" {
		return nil, model.ErrNotFound
	}
	sel := r.selectPlayer().Where(Eq{"player.api_key_hash": hashAPIKey(key)})
	var res model.Player
	if err := r.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GenerateAPIKey is owner-only, even for admins, so nobody can mint a login for someone else.
func (r *playerRepository) GenerateAPIKey(playerID string) (string, error) {
	key := consts.APIKeyPrefix + id.NewRandom()
	upd := Update(r.tableName).Set("api_key_hash", hashAPIKey(key)).
		Where(Eq{"id": playerID, "user_id": loggedUser(r.ctx).ID})
	count, err := r.executeSQL(upd)
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", r.classifyOwnedWriteMiss(playerID)
	}
	return key, nil
}

func (r *playerRepository) RevokeAPIKey(playerID string) error {
	upd := Update(r.tableName).Set("api_key_hash", nil).Where(r.addRestriction(Eq{"id": playerID}))
	count, err := r.executeSQL(upd)
	if err != nil {
		return err
	}
	if count == 0 {
		return r.classifyOwnedWriteMiss(playerID)
	}
	return nil
}

var _ model.PlayerRepository = (*playerRepository)(nil)
var _ rest.Repository = (*playerRepository)(nil)
var _ rest.Persistable = (*playerRepository)(nil)
