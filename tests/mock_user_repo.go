package tests

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
)

func CreateMockUserRepo() *MockedUserRepo {
	return &MockedUserRepo{
		Data: map[string]*model.User{},
	}
}

type MockedUserRepo struct {
	model.UserRepository
	Error error
	Data  map[string]*model.User
}

func (u *MockedUserRepo) CountAll(qo ...model.QueryOptions) (int64, error) {
	if u.Error != nil {
		return 0, u.Error
	}
	return int64(len(u.Data)), nil
}

func (u *MockedUserRepo) Put(usr *model.User) error {
	if u.Error != nil {
		return u.Error
	}
	if usr.ID == "" {
		usr.ID = base64.StdEncoding.EncodeToString([]byte(usr.UserName))
	}
	usr.Password = usr.NewPassword
	u.Data[strings.ToLower(usr.UserName)] = usr
	return nil
}

func (u *MockedUserRepo) FindByUsername(username string) (*model.User, error) {
	if u.Error != nil {
		return nil, u.Error
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
	for _, usr := range u.Data {
		if usr.ID == id {
			usr.LastLoginAt = gg.P(time.Now())
			return nil
		}
	}
	return u.Error
}

func (u *MockedUserRepo) UpdateLastAccessAt(id string) error {
	for _, usr := range u.Data {
		if usr.ID == id {
			usr.LastAccessAt = gg.P(time.Now())
			return nil
		}
	}
	return u.Error
}
