package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type playerRepository struct {
	sqlRepository
}

func NewPlayerRepository(ctx context.Context, o orm.Ormer) model.PlayerRepository {
	r := &playerRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "player"
	return r
}

func (r *playerRepository) Put(p *model.Player) error {
	_, err := r.put(p.ID, p)
	return err
}

func (r *playerRepository) Get(id string) (*model.Player, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Player
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *playerRepository) FindByName(client, userName string) (*model.Player, error) {
	sel := r.newSelect().Columns("*").Where(And{Eq{"client": client}, Eq{"user_name": userName}})
	var res model.Player
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *playerRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(options...))
}

func (r *playerRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *playerRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).Columns("*")
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

func (r *playerRepository) Save(entity interface{}) (string, error) {
	t := entity.(*model.Player)
	id, err := r.put(t.ID, t)
	if err == model.ErrNotFound {
		return "", rest.ErrNotFound
	}
	return id, err
}

func (r *playerRepository) Update(entity interface{}, cols ...string) error {
	t := entity.(*model.Player)
	_, err := r.put(t.ID, t)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func (r *playerRepository) Delete(id string) error {
	err := r.delete(Eq{"id": id})
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

var _ model.PlayerRepository = (*playerRepository)(nil)
var _ rest.Repository = (*playerRepository)(nil)
var _ rest.Persistable = (*playerRepository)(nil)
