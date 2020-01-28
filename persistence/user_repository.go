package persistence

import (
	"context"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
	"github.com/google/uuid"
)

type userRepository struct {
	sqlRepository
}

func NewUserRepository(ctx context.Context, o orm.Ormer) model.UserRepository {
	r := &userRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "user"
	return r
}

func (r *userRepository) CountAll(qo ...model.QueryOptions) (int64, error) {
	return r.count(Select(), qo...)
}

func (r *userRepository) Get(id string) (*model.User, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.User
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *userRepository) GetAll(options ...model.QueryOptions) (model.Users, error) {
	sel := r.newSelect(options...).Columns("*")
	var res model.Users
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *userRepository) Put(u *model.User) error {
	if u.ID == "" {
		id, _ := uuid.NewRandom()
		u.ID = id.String()
	}
	u.UserName = strings.ToLower(u.UserName)
	values, _ := toSqlArgs(*u)
	update := Update(r.tableName).Where(Eq{"id": u.ID}).SetMap(values)
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"user_name": username})
	var usr model.User
	err := r.queryOne(sel, &usr)
	return &usr, err
}

func (r *userRepository) UpdateLastLoginAt(id string) error {
	upd := Update(r.tableName).Where(Eq{"id": id}).Set("last_login_at", time.Now())
	_, err := r.executeSQL(upd)
	return err
}

func (r *userRepository) UpdateLastAccessAt(id string) error {
	now := time.Now()
	upd := Update(r.tableName).Where(Eq{"id": id}).Set("last_access_at", now)
	_, err := r.executeSQL(upd)
	return err
}

func (r *userRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *userRepository) Read(id string) (interface{}, error) {
	usr, err := r.Get(id)
	if err == model.ErrNotFound {
		return nil, rest.ErrNotFound
	}
	return usr, err
}

func (r *userRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *userRepository) EntityName() string {
	return "user"
}

func (r *userRepository) NewInstance() interface{} {
	return &model.User{}
}

func (r *userRepository) Save(entity interface{}) (string, error) {
	usr := entity.(*model.User)
	err := r.Put(usr)
	if err != nil {
		return "", err
	}
	return usr.ID, err
}

func (r *userRepository) Update(entity interface{}, cols ...string) error {
	usr := entity.(*model.User)
	err := r.Put(usr)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func (r *userRepository) Delete(id string) error {
	err := r.Delete(id)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

var _ model.UserRepository = (*userRepository)(nil)
var _ rest.Repository = (*userRepository)(nil)
var _ rest.Persistable = (*userRepository)(nil)
