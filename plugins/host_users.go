package plugins

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/slice"
)

type usersServiceImpl struct {
	ds    model.DataStore
	users userAccess
}

func newUsersService(ds model.DataStore, users userAccess) host.UsersService {
	return &usersServiceImpl{ds: ds, users: users}
}

func (s *usersServiceImpl) GetUsers(ctx context.Context) ([]host.User, error) {
	users, err := s.ds.User().GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var result []host.User
	for _, u := range users {
		if s.users.allows(u.ID) {
			result = append(result, host.User{
				UserName: u.UserName,
				Name:     u.Name,
				IsAdmin:  u.IsAdmin,
			})
		}
	}

	return result, nil
}

func (s *usersServiceImpl) GetAdmins(ctx context.Context) ([]host.User, error) {
	users, err := s.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	return slice.Filter(users, func(u host.User) bool {
		return u.IsAdmin
	}), nil
}

var _ host.UsersService = (*usersServiceImpl)(nil)
