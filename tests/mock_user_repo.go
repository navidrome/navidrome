package tests

import (
	"encoding/base64"
	"strings"

	"github.com/navidrome/navidrome/model"
)

type mockedUserRepo struct {
	model.UserRepository
	data map[string]*model.User
}

func (u *mockedUserRepo) CountAll(qo ...model.QueryOptions) (int64, error) {
	return int64(len(u.data)), nil
}

func (u *mockedUserRepo) Put(usr *model.User) error {
	if u.data == nil {
		u.data = make(map[string]*model.User)
	}
	if usr.ID == "" {
		usr.ID = base64.StdEncoding.EncodeToString([]byte(usr.UserName))
	}
	usr.Password = usr.NewPassword
	u.data[strings.ToLower(usr.UserName)] = usr
	return nil
}

func (u *mockedUserRepo) FindByUsername(username string) (*model.User, error) {
	usr, ok := u.data[strings.ToLower(username)]
	if !ok {
		return nil, model.ErrNotFound
	}
	return usr, nil
}

func (u *mockedUserRepo) UpdateLastLoginAt(id string) error {
	return nil
}
