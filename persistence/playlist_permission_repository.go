package persistence

import (
	. "github.com/Masterminds/squirrel"

	"github.com/navidrome/navidrome/model"
)

type playlistPermissionRepository struct {
	sqlRepository
	playlistId string
}

func (r *playlistRepository) Permissions(playlistId string) model.PlaylistPermissionRepository {
	p := &playlistPermissionRepository{}
	p.playlistId = playlistId
	p.ctx = r.ctx
	p.db = r.db
	p.tableName = "playlist_permissions"
	p.registerModel(&model.PlaylistPermission{}, nil)
	return p
}

func (r *playlistPermissionRepository) selectPermissions(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).
		Columns(r.tableName + ".*")
}

func (r *playlistPermissionRepository) GetAll() (model.PlaylistPermissions, error) {
	sel := r.selectPermissions().
		Where(Eq{"playlist_id": r.playlistId})

	var perms model.PlaylistPermissions
	if err := r.queryAll(sel, &perms); err != nil {
		return nil, err
	}

	return perms, nil
}

func (r *playlistPermissionRepository) IsUserAllowed(userID string, permissions []model.Permission) (bool, error) {
	permsOr := Or{}
	for _, perm := range permissions {
		permsOr = append(permsOr, Eq{"permission": perm})
	}

	existsQuery := Select("count(*) as exist").From(r.tableName).
		Where(And{
			Eq{"playlist_id": r.playlistId},
			Eq{"user_id": userID},
			permsOr,
		})
	var res struct{ Exist int64 }
	err := r.queryOne(existsQuery, &res)
	return res.Exist > 0, err
}

func (r *playlistPermissionRepository) Put(userID string, permission model.Permission) error {
	insert := Insert(r.tableName).
		Columns("playlist_id", "user_id", "permission").
		Values(r.playlistId, userID, permission).
		Suffix(`ON CONFLICT(playlist_id, user_id) DO UPDATE SET permission = ?`, permission)
	_, err := r.executeSQL(insert)
	return err
}

func (r *playlistPermissionRepository) Delete(userID string) error {
	return r.delete(And{Eq{"playlist_id": r.playlistId, "user_id": userID}})
}
