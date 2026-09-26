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

func NewPlayerRepository(db dbx.Builder) model.PlayerRepository {
	r := &playerRepository{}
	r.db = db
	r.registerModel(&model.Player{}, map[string]filterFunc{
		"name": containsFilter("player.name"),
	})
	r.setSortMappings(map[string]string{
		"user_name": "username", //TODO rename all user_name and userName to username
	})
	return r
}

func (r *playerRepository) Put(ctx context.Context, p *model.Player) error {
	_, err := r.put(ctx, p.ID, p)
	return err
}

func (r *playerRepository) selectPlayer(ctx context.Context, options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(ctx, options...).
		Columns("player.*", "player.api_key_hash is not null as has_api_key").
		Join("user ON player.user_id = user.id").
		Columns("user.user_name username")
}

func (r *playerRepository) Get(ctx context.Context, id string) (*model.Player, error) {
	sel := r.selectPlayer(ctx).Where(Eq{"player.id": id})
	var res model.Player
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

func (r *playerRepository) FindMatch(ctx context.Context, userId, client, userAgent string) (*model.Player, error) {
	sel := r.selectPlayer(ctx).Where(And{
		Eq{"client": client},
		Eq{"user_agent": userAgent},
		Eq{"user_id": userId},
	})
	var res model.Player
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

func (r *playerRepository) newRestSelect(ctx context.Context, options ...model.QueryOptions) SelectBuilder {
	s := r.selectPlayer(ctx, options...)
	return s.Where(r.addRestriction(ctx))
}

func (r *playerRepository) CountByClient(ctx context.Context, options ...model.QueryOptions) (map[string]int64, error) {
	sel := r.newSelect(ctx, options...).
		Columns(
			"case when client = 'NavidromeUI' then name else client end as player",
			"count(*) as count",
		).GroupBy("client")
	var res []struct {
		Player string
		Count  int64
	}
	err := r.queryAll(ctx, sel, &res)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(res))
	for _, c := range res {
		counts[c.Player] = c.Count
	}
	return counts, nil
}

func (r *playerRepository) CountAll(ctx context.Context, options ...model.QueryOptions) (int64, error) {
	return r.count(ctx, r.newRestSelect(ctx), options...)
}

func (r *playerRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *playerRepository) Read(ctx context.Context, id string) (*model.Player, error) {
	sel := r.newRestSelect(ctx).Where(Eq{"player.id": id})
	var res model.Player
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

func (r *playerRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Player, error) {
	sel := r.newRestSelect(ctx, r.parseRestOptions(ctx, options...))
	res := model.Players{}
	err := r.queryAll(ctx, sel, &res)
	return res, err
}

var apiKeyFormat = regexp.MustCompile(`^` + consts.APIKeyPrefix + `[0-9A-Za-z]{22}$`)

func apiKeyValidationError(msg string) error {
	return &rest.ValidationError{Errors: map[string]string{"apiKey": msg}}
}

func validateAPIKey(key string) error {
	if !apiKeyFormat.MatchString(key) {
		return apiKeyValidationError("resources.player.validation.apiKeyFormat")
	}
	return nil
}

func (r *playerRepository) Save(ctx context.Context, t *model.Player) (string, error) {
	u := loggedUser(ctx)
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
	if err := validateAPIKey(*t.APIKey); err != nil {
		return "", err
	}
	values, err := toSQLArgs(t)
	if err != nil {
		return "", err
	}
	// Save only creates, so the key hash goes in the same INSERT and the unique index settles races
	values["id"] = id.NewRandom()
	values["api_key_hash"] = hashAPIKey(*t.APIKey)
	_, err = r.executeSQL(ctx, Insert(r.tableName).SetMap(values))
	if isUniqueViolation(err) {
		return "", apiKeyValidationError("ra.validation.unique")
	}
	if err != nil {
		return "", err
	}
	return values["id"].(string), nil
}

func (r *playerRepository) Update(ctx context.Context, id string, entity model.Player, cols ...string) error {
	t := &entity
	t.ID = id
	if t.APIKey == nil {
		return r.updateOwned(ctx, id, t, cols...)
	}
	// The key and the other columns are two writes; commit both or neither
	return r.inTx(func(tx *playerRepository) error {
		if err := tx.SetAPIKey(ctx, id, *t.APIKey); err != nil {
			return err
		}
		return tx.updateOwned(ctx, id, t, cols...)
	})
}

func (r *playerRepository) inTx(block func(tx *playerRepository) error) error {
	conn, ok := r.db.(*dbx.DB)
	if !ok {
		return block(r) // already inside a transaction
	}
	return conn.Transactional(func(tx *dbx.Tx) error {
		return block(NewPlayerRepository(tx).(*playerRepository))
	})
}

func (r *playerRepository) Delete(ctx context.Context, ids ...string) error {
	return r.deleteOwnedAll(ctx, ids...)
}

// Keys are long random strings, not user-chosen passwords, so a fast unsalted hash is enough and keeps lookups indexed.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (r *playerRepository) FindByAPIKey(ctx context.Context, key string) (*model.Player, error) {
	sel := r.selectPlayer(ctx).Where(Eq{"player.api_key_hash": hashAPIKey(key)})
	var res model.Player
	if err := r.queryOne(ctx, sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SetAPIKey stores the key's hash, or revokes it when key is empty. Setting is owner-only, even for
// admins, so nobody can mint a login for someone else.
func (r *playerRepository) SetAPIKey(ctx context.Context, playerID, key string) error {
	if key == "" {
		return r.updateOwnedRow(ctx, playerID, ownerOrAdmin, map[string]any{"api_key_hash": nil})
	}
	if err := validateAPIKey(key); err != nil {
		return err
	}
	err := r.updateOwnedRow(ctx, playerID, ownerOnly, map[string]any{"api_key_hash": hashAPIKey(key)})
	if isUniqueViolation(err) {
		return apiKeyValidationError("ra.validation.unique")
	}
	return err
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

var _ model.PlayerRepository = (*playerRepository)(nil)
var _ rest.Repository[model.Player] = (*playerRepository)(nil)
var _ rest.Persistable[model.Player] = (*playerRepository)(nil)
