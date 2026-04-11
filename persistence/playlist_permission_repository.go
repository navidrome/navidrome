package persistence

import (
	. "github.com/Masterminds/squirrel"

	"github.com/navidrome/navidrome/log"
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

	_, err := r.Get(playlistId)
	if err != nil {
		log.Warn(r.ctx, "Error getting playlist's tracks", "playlistId", playlistId, err)
		return nil
	}

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

// TODO: consider dropping this and replacing it with a call to `GetForPlaylist` (adjust it to accept `model.QueryOptions` and then pass a Filter with userID and permission)
// TODO: is there actually a usecase to accept a slice of permissions?
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
	// Remove existing permission
	if err := r.Delete(userID); err != nil {
		return err
	}

	// Add new permission
	insert := Insert(r.tableName).
		Columns("playlist_id", "user_id", "permission").
		Values(r.playlistId, userID, permission)
	_, err := r.executeSQL(insert)
	return err
}

func (r *playlistPermissionRepository) Delete(userID string) error {
	return r.delete(And{Eq{"playlist_id": r.playlistId, "user_id": userID}})
}
