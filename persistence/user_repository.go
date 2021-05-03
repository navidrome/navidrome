package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/rest"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
)

type userRepository struct {
	sqlRepository
	sqlRestful
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
	res := model.Users{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *userRepository) Put(u *model.User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	u.UpdatedAt = time.Now()
	values, _ := toSqlArgs(*u)
	delete(values, "current_password")
	update := Update(r.tableName).Where(Eq{"id": u.ID}).SetMap(values)
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	values["created_at"] = time.Now()
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r *userRepository) FindFirstAdmin() (*model.User, error) {
	sel := r.newSelect(model.QueryOptions{Sort: "updated_at", Max: 1}).Columns("*").Where(Eq{"is_admin": true})
	var usr model.User
	err := r.queryOne(sel, &usr)
	return &usr, err
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	sel := r.newSelect().Columns("*").Where(Like{"user_name": username})
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
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin {
		return 0, rest.ErrPermissionDenied
	}
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *userRepository) Read(id string) (interface{}, error) {
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin && usr.ID != id {
		return nil, rest.ErrPermissionDenied
	}
	usr, err := r.Get(id)
	if err == model.ErrNotFound {
		return nil, rest.ErrNotFound
	}
	return usr, err
}

func (r *userRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin {
		return nil, rest.ErrPermissionDenied
	}
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *userRepository) EntityName() string {
	return "user"
}

func (r *userRepository) NewInstance() interface{} {
	return &model.User{}
}

func (r *userRepository) Save(entity interface{}) (string, error) {
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin {
		return "", rest.ErrPermissionDenied
	}
	u := entity.(*model.User)
	err := r.Put(u)
	if err != nil {
		return "", err
	}
	return u.ID, err
}

func (r *userRepository) Update(entity interface{}, cols ...string) error {
	u := entity.(*model.User)
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin && usr.ID != u.ID {
		return rest.ErrPermissionDenied
	}
	if !usr.IsAdmin {
		if !conf.Server.EnableUserEditing {
			return rest.ErrPermissionDenied
		}
		u.IsAdmin = false
		u.UserName = usr.UserName
	}
	if err := validatePasswordChange(u, usr); err != nil {
		return err
	}
	err := r.Put(u)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func validatePasswordChange(newUser *model.User, logged *model.User) error {
	err := &rest.ValidationError{Errors: map[string]string{}}
	if logged.IsAdmin && newUser.ID != logged.ID {
		return nil
	}
	if newUser.NewPassword != "" && newUser.CurrentPassword == "" {
		err.Errors["currentPassword"] = "ra.validation.required"
	}
	if newUser.CurrentPassword != "" {
		if newUser.NewPassword == "" {
			err.Errors["password"] = "ra.validation.required"
		}
		if newUser.CurrentPassword != logged.Password {
			err.Errors["currentPassword"] = "ra.validation.passwordDoesNotMatch"
		}
	}
	if len(err.Errors) > 0 {
		return err
	}
	return nil
}

func (r *userRepository) Delete(id string) error {
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin {
		return rest.ErrPermissionDenied
	}
	err := r.delete(Eq{"id": id})
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

var _ model.UserRepository = (*userRepository)(nil)
var _ rest.Repository = (*userRepository)(nil)
var _ rest.Persistable = (*userRepository)(nil)
