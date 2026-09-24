package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

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

var apiKeyFormat = regexp.MustCompile(`^` + consts.APIKeyPrefix + `[0-9A-Za-z]{22}$`)

func apiKeyValidationError(msg string) error {
	return &rest.ValidationError{Errors: map[string]string{"apiKey": msg}}
}

func (r *playerRepository) Save(entity any) (string, error) {
	t := entity.(*model.Player)
	u := loggedUser(r.ctx)
	if t.UserId == "" && u.ID != invalidUserId {
		t.UserId = u.ID
	}
	if t.UserId != u.ID {
		return "", rest.ErrPermissionDenied
	}
	// Hand-made players are only reachable through a key, so one is required
	if t.APIKey == nil || *t.APIKey == "" {
		return "", apiKeyValidationError("ra.validation.required")
	}
	if !apiKeyFormat.MatchString(*t.APIKey) {
		return "", apiKeyValidationError("resources.player.validation.apiKeyFormat")
	}
	values, err := toSQLArgs(t)
	if err != nil {
		return "", err
	}
	// Save only creates, so the key hash goes in the same INSERT and the unique index settles races
	values["id"] = id.NewRandom()
	values["api_key_hash"] = hashAPIKey(*t.APIKey)
	_, err = r.executeSQL(Insert(r.tableName).SetMap(values))
	if isUniqueViolation(err) {
		return "", apiKeyValidationError("ra.validation.unique")
	}
	if err != nil {
		return "", err
	}
	return values["id"].(string), nil
}

func (r *playerRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.Player)
	t.ID = id
	if t.APIKey != nil {
		if err := r.SetAPIKey(id, *t.APIKey); err != nil {
			return err
		}
	}
	return r.updateOwned(id, t, cols...)
}

func (r *playerRepository) Delete(id string) error {
	return r.deleteOwned(id)
}

// Keys are long random strings, not user-chosen passwords, so a fast unsalted hash is enough and keeps lookups indexed.
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

// SetAPIKey stores the key's hash, or revokes it when key is empty. Setting is owner-only, even for
// admins, so nobody can mint a login for someone else.
func (r *playerRepository) SetAPIKey(playerID, key string) error {
	if key == "" {
		return r.updateOwnedRow(playerID, ownerOrAdmin, map[string]any{"api_key_hash": nil})
	}
	if !apiKeyFormat.MatchString(key) {
		return apiKeyValidationError("resources.player.validation.apiKeyFormat")
	}
	err := r.updateOwnedRow(playerID, ownerOnly, map[string]any{"api_key_hash": hashAPIKey(key)})
	if isUniqueViolation(err) {
		return apiKeyValidationError("ra.validation.unique")
	}
	return err
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

var _ model.PlayerRepository = (*playerRepository)(nil)
var _ rest.Repository = (*playerRepository)(nil)
var _ rest.Persistable = (*playerRepository)(nil)
