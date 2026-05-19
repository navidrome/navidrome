package tests

import (
	"errors"
	"slices"

	"github.com/navidrome/navidrome/model"
)

func CreateMockPlaylistPermissionRepo() *MockPlaylistPermissionRepo {
	return &MockPlaylistPermissionRepo{
		Data: make(map[string]*model.PlaylistPermission),
	}
}

type MockPlaylistPermissionRepo struct {
	model.PlaylistPermissionRepository
	Data map[string]*model.PlaylistPermission // keyed by userID
	Err  bool
}

func (m *MockPlaylistPermissionRepo) SetError(setError bool) {
	m.Err = setError
}

func (m *MockPlaylistPermissionRepo) GetAll() (model.PlaylistPermissions, error) {
	if m.Err {
		return nil, errors.New("error")
	}
	if m.Data != nil {
		perms := make(model.PlaylistPermissions, 0, len(m.Data))
		for i := range m.Data {
			perms = append(perms, *m.Data[i])
		}
		return perms, nil
	}

	return nil, model.ErrNotFound
}

func (m *MockPlaylistPermissionRepo) IsUserAllowed(userID string, permissions []model.Permission) (bool, error) {
	if m.Err {
		return false, errors.New("error")
	}
	if m.Data != nil {
		if perm, ok := m.Data[userID]; ok {
			if slices.Contains(permissions, perm.Permission) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *MockPlaylistPermissionRepo) Put(userID string, permission model.Permission) error {
	if m.Err {
		return errors.New("error")
	}
	if m.Data != nil {
		m.Data[userID] = &model.PlaylistPermission{Permission: permission}
	}
	return nil
}

func (m *MockPlaylistPermissionRepo) Delete(userID string) error {
	if m.Err {
		return errors.New("error")
	}
	if m.Data != nil {
		delete(m.Data, userID)
	}
	return nil
}
