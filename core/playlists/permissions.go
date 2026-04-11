package playlists

import (
	"context"
	"fmt"
	"slices"

	"github.com/navidrome/navidrome/model"
)

func (s *playlists) GetPermissionsForPlaylist(ctx context.Context, playlistID string) (model.PlaylistPermissions, error) {
	if _, err := s.checkWritable(ctx, playlistID); err != nil {
		return nil, err
	}

	playlistPermissions, err := s.ds.Playlist(ctx).Permissions(playlistID).GetAll()
	if err != nil {
		return nil, err
	}

	return playlistPermissions, nil
}

var userPermissions = []model.Permission{model.PermissionEditor, model.PermissionViewer}

// AddPermission grants the user the provided permission on the playlist.
// If the user is already granted a permission on the playlist, consecutive calls override the current permission.
func (s *playlists) AddPermission(ctx context.Context, playlistID string, userID string, permission model.Permission) error {
	playlist, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return err
	}

	if userID == playlist.OwnerID {
		return fmt.Errorf("%w: playlist owner must not be added to the playlist", model.ErrValidation)

	}

	if !slices.Contains(userPermissions, permission) {
		return fmt.Errorf("%w: permission %q not supported, possible values %q", model.ErrValidation, permission, userPermissions)
	}

	if _, err := s.ds.User(ctx).Get(userID); err != nil {
		return fmt.Errorf("failed validating existence of user with ID %q: %w", userID, err)
	}

	return s.ds.Playlist(ctx).Permissions(playlistID).Put(userID, permission)
}

// RemovePermission removes the granted permission of the user from the playlist.
func (s *playlists) RemovePermission(ctx context.Context, playlistID string, userID string) error {
	_, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return err
	}

	if _, err := s.ds.User(ctx).Get(userID); err != nil {
		return fmt.Errorf("validating existence of user with ID %q: %w", userID, err)
	}

	// TODO: check if the Delete actually did something (might not be needed as the api response already tells the users if the set of permissions changed)
	return s.ds.Playlist(ctx).Permissions(playlistID).Delete(userID)
}
