package tests

import (
	"encoding/base64"
	"strings"

	"github.com/navidrome/navidrome/model"
)

func CreateMockUserRepo() *MockedUserRepo {
	return &MockedUserRepo{
		Data: map[string]*model.User{},
	}
}

type MockedUserRepo struct {
	model.UserRepository
	Err  error
	Data map[string]*model.User
}

func (u *MockedUserRepo) CountAll(qo ...model.QueryOptions) (int64, error) {
	if u.Err != nil {
		return 0, u.Err
	}
	return int64(len(u.Data)), nil
}

func (u *MockedUserRepo) Put(usr *model.User) error {
	if u.Err != nil {
		return u.Err
	}
	if usr.ID == "" {
		usr.ID = base64.StdEncoding.EncodeToString([]byte(usr.UserName))
	}
	usr.Password = usr.NewPassword
	u.Data[strings.ToLower(usr.UserName)] = usr
	return nil
}

func (u *MockedUserRepo) FindByUsername(username string) (*model.User, error) {
	if u.Err != nil {
		return nil, u.Err
	}
	usr, ok := u.Data[strings.ToLower(username)]
	if !ok {
		return nil, model.ErrNotFound
	}
	return usr, nil
}

func (u *MockedUserRepo) FindByUsernameWithPassword(username string) (*model.User, error) {
	return u.FindByUsername(username)
}

func (u *MockedUserRepo) UpdateLastLoginAt(id string) error {
	return u.Err
}
