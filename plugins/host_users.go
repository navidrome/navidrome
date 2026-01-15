package plugins

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/utils/slice"
)

type usersServiceImpl struct {
	ds           model.DataStore
	allowedUsers []string // User IDs this plugin can access
	allUsers     bool     // If true, plugin can access all users
}

func newUsersService(ds model.DataStore, allowedUsers []string, allUsers bool) host.UsersService {
	return &usersServiceImpl{
		ds:           ds,
		allowedUsers: allowedUsers,
		allUsers:     allUsers,
	}
}

func (s *usersServiceImpl) GetUsers(ctx context.Context) ([]host.User, error) {
	users, err := s.ds.User(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	// Build allowed users map for efficient lookup
	allowedMap := make(map[string]bool, len(s.allowedUsers))
	for _, id := range s.allowedUsers {
		allowedMap[id] = true
	}

	var result []host.User
	for _, u := range users {
		// If allUsers is true, include all users
		// Otherwise, only include users in the allowed list
		if s.allUsers || allowedMap[u.ID] {
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
