package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
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
		Columns("player.*").
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

// isPermitted authorizes creating a new record, based on the owner declared in the request body.
// This is only safe for inserts: there is no stored row yet, and a non-admin may only create a
// player they own. Updates must not use this (the body owner is attacker-controlled); they go
// through updateOwned, which authorizes against the persisted user_id in the WHERE clause.
func (r *playerRepository) isPermitted(ctx context.Context, p *model.Player) bool {
	u := loggedUser(ctx)
	return u.IsAdmin || p.UserId == u.ID
}

func (r *playerRepository) Save(ctx context.Context, t *model.Player) (string, error) {
	if !r.isPermitted(ctx, t) {
		return "", rest.ErrPermissionDenied
	}
	return r.put(ctx, "", t) // Save only creates; edits go through the owner-scoped Update
}

func (r *playerRepository) Update(ctx context.Context, id string, entity model.Player, cols ...string) error {
	t := &entity
	t.ID = id
	return r.updateOwned(ctx, id, t, cols...)
}

func (r *playerRepository) Delete(ctx context.Context, ids ...string) error {
	return r.deleteOwnedAll(ctx, ids...)
}

var _ model.PlayerRepository = (*playerRepository)(nil)
var _ rest.Repository[model.Player] = (*playerRepository)(nil)
var _ rest.Persistable[model.Player] = (*playerRepository)(nil)
