package tests

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
)

func CreateMockUserRepo() *MockedUserRepo {
	return &MockedUserRepo{
		Data:          map[string]*model.User{},
		UserLibraries: map[string][]int{},
	}
}

type MockedUserRepo struct {
	model.UserRepository
	Error         error
	Data          map[string]*model.User
	UserLibraries map[string][]int // userID -> libraryIDs
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

func (u *MockedUserRepo) Get(id string) (*model.User, error) {
	if u.Error != nil {
		return nil, u.Error
	}
	for _, usr := range u.Data {
		if usr.ID == id {
			return usr, nil
		}
	}
	return nil, model.ErrNotFound
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

// Library association methods - mock implementations

func (u *MockedUserRepo) GetUserLibraries(userID string) (model.Libraries, error) {
	if u.Error != nil {
		return nil, u.Error
	}
	libraryIDs, exists := u.UserLibraries[userID]
	if !exists {
		return model.Libraries{}, nil
	}

	// Mock: Create libraries based on IDs
	var libraries model.Libraries
	for _, id := range libraryIDs {
		libraries = append(libraries, model.Library{
			ID:   id,
			Name: fmt.Sprintf("Test Library %d", id),
			Path: fmt.Sprintf("/music/library%d", id),
		})
	}
	return libraries, nil
}

func (u *MockedUserRepo) SetUserLibraries(userID string, libraryIDs []int) error {
	if u.Error != nil {
		return u.Error
	}
	if u.UserLibraries == nil {
		u.UserLibraries = make(map[string][]int)
	}
	u.UserLibraries[userID] = libraryIDs
	return nil
}
