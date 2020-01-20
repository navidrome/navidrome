package persistence

import (
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/deluan/rest"
)

type user struct {
	ID           string     `json:"id"             orm:"pk;column(id)"`
	UserName     string     `json:"userName"       orm:"index;unique"`
	Name         string     `json:"name"`
	Email        string     `json:"email"          orm:"unique"`
	Password     string     `json:"password"`
	IsAdmin      bool       `json:"isAdmin"`
	LastLoginAt  *time.Time `json:"lastLoginAt"    orm:"null"`
	LastAccessAt *time.Time `json:"lastAccessAt"   orm:"null"`
	CreatedAt    time.Time  `json:"createdAt"      orm:"auto_now_add;type(datetime)"`
	UpdatedAt    time.Time  `json:"updatedAt"      orm:"auto_now;type(datetime)"`
}

type userRepository struct {
	ormer        orm.Ormer
	userResource model.ResourceRepository
}

func NewUserRepository(o orm.Ormer) model.UserRepository {
	r := &userRepository{ormer: o}
	r.userResource = NewResource(o, model.User{}, new(user))
	return r
}

func (r *userRepository) CountAll(qo ...model.QueryOptions) (int64, error) {
	if len(qo) > 0 {
		return r.userResource.Count(rest.QueryOptions(qo[0]))
	}
	return r.userResource.Count()
}

func (r *userRepository) Get(id string) (*model.User, error) {
	u, err := r.userResource.Read(id)
	if err != nil {
		return nil, err
	}
	res := model.User(u.(user))
	return &res, nil
}

func (r *userRepository) Put(u *model.User) error {
	tu := user(*u)
	c, err := r.CountAll()
	if err != nil {
		return err
	}
	if c == 0 {
		_, err = r.userResource.Save(&tu)
		return err
	}
	return r.userResource.Update(&tu, "user_name", "is_admin", "password")
}

func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	tu := user{}
	err := r.ormer.QueryTable(user{}).Filter("user_name", username).One(&tu)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u := model.User(tu)
	return &u, err
}

func (r *userRepository) UpdateLastLoginAt(id string) error {
	now := time.Now()
	tu := user{ID: id, LastLoginAt: &now}
	_, err := r.ormer.Update(&tu, "last_login_at")
	return err
}

var _ = model.User(user{})
var _ model.UserRepository = (*userRepository)(nil)
