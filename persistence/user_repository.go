package persistence

import (
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/deluan/rest"
)

type user struct {
	ID           string     `json:"id"             orm:"pk;column(id)"`
	Name         string     `json:"name"           orm:"index"`
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
	c, err := r.CountAll()
	if err != nil {
		return err
	}
	if c == 0 {
		mappedUser := user(*u)
		_, err = r.userResource.Save(&mappedUser)
		return err
	}
	return r.userResource.Update(u, "name", "is_admin", "password")
}

func NewUserRepository(o orm.Ormer) model.UserRepository {
	r := &userRepository{ormer: o}
	r.userResource = NewResource(o, model.User{}, new(user))
	return r
}

var _ = model.User(user{})
var _ model.UserRepository = (*userRepository)(nil)
