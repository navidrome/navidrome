package persistence

import (
	"context"
	"errors"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
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
		Columns("player.*").
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

func (r *playerRepository) FindByAPIKey(key string) (*model.Player, error) {
	if key == "" {
		return nil, model.ErrNotFound
	}
	sel := r.selectPlayer().Where(Eq{"api_key": key})
	var res model.Player
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *playerRepository) newRestSelect(options ...model.QueryOptions) SelectBuilder {
	s := r.selectPlayer(options...)
	return s.Where(r.addRestriction())
}

func (r *playerRepository) addRestriction(sql ...Sqlizer) Sqlizer {
	s := And{}
	if len(sql) > 0 {
		s = append(s, sql[0])
	}
	u := loggedUser(r.ctx)
	if u.IsAdmin {
		return s
	}
	return append(s, Eq{"user_id": u.ID})
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

func (r *playerRepository) Read(id string) (interface{}, error) {
	sel := r.newRestSelect().Where(Eq{"player.id": id})
	var res model.Player
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *playerRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newRestSelect(r.parseRestOptions(r.ctx, options...))
	res := model.Players{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *playerRepository) EntityName() string {
	return "player"
}

func (r *playerRepository) NewInstance() interface{} {
	return &model.Player{}
}

func (r *playerRepository) isPermitted(p *model.Player) bool {
	u := loggedUser(r.ctx)
	return u.IsAdmin || p.UserId == u.ID
}

func (r *playerRepository) preparePlayerForSave(p *model.Player) *model.Player {
	playerCopy := *p
	if playerCopy.UserId == "" {
		user := loggedUser(r.ctx)
		playerCopy.UserId = user.ID
	}
	playerCopy.APIKey = ""
	return &playerCopy
}

func (r *playerRepository) Save(entity interface{}) (string, error) {
	t := entity.(*model.Player)
	playerToSave := r.preparePlayerForSave(t)
	if !r.isPermitted(playerToSave) {
		return "", rest.ErrPermissionDenied
	}
	playerId, err := r.put(playerToSave.ID, playerToSave)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return playerId, err
}

func (r *playerRepository) Update(id string, entity interface{}, cols ...string) error {
	t := entity.(*model.Player)
	t.ID = id
	playerToUpdate := r.preparePlayerForSave(t)
	if !r.isPermitted(playerToUpdate) {
		return rest.ErrPermissionDenied
	}
	_, err := r.put(id, playerToUpdate, cols...)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

func (r *playerRepository) Delete(id string) error {
	filter := r.addRestriction(And{Eq{"player.id": id}})
	err := r.delete(filter)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

// GenerateAPIKey generates a new API key for the specified player
func (r *playerRepository) GenerateAPIKey(playerID string) (string, error) {
	player, err := r.Get(playerID)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	if err != nil {
		return "", err
	}

	if !r.isPermitted(player) {
		return "", rest.ErrPermissionDenied
	}

	player.APIKey = generateAPIKey()
	err = r.Put(player)
	if err != nil {
		return "", err
	}
	return player.APIKey, nil
}

// generateAPIKey creates a new random API key
func generateAPIKey() string {
	return "nav_" + id.NewRandom()
}

var _ model.PlayerRepository = (*playerRepository)(nil)
var _ rest.Repository = (*playerRepository)(nil)
var _ rest.Persistable = (*playerRepository)(nil)
