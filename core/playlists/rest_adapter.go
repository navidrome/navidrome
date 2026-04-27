package playlists

import (
	"context"
	"errors"
	"reflect"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
)

// --- REST adapter (follows Share/Library pattern) ---

func (s *playlists) NewRepository(ctx context.Context) rest.Repository {
	return &playlistRepositoryWrapper{
		ctx:                ctx,
		PlaylistRepository: s.ds.Playlist(ctx),
		service:            s,
	}
}

// playlistRepositoryWrapper wraps the playlist repository as a thin REST-to-service adapter.
// It satisfies rest.Repository through the embedded PlaylistRepository (via ResourceRepository),
// and rest.Persistable by delegating to service methods for all mutations.
type playlistRepositoryWrapper struct {
	model.PlaylistRepository
	ctx     context.Context
	service *playlists
}

func (r *playlistRepositoryWrapper) Save(entity any) (string, error) {
	return r.service.savePlaylist(r.ctx, entity.(*model.Playlist))
}

func (r *playlistRepositoryWrapper) Update(id string, entity any, _ ...string) error {
	return r.service.updatePlaylistEntity(r.ctx, id, entity.(*model.Playlist))
}

func (r *playlistRepositoryWrapper) Delete(id string) error {
	err := r.service.Delete(r.ctx, id)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return rest.ErrNotFound
	case errors.Is(err, model.ErrNotAuthorized):
		return rest.ErrPermissionDenied
	default:
		return err
	}
}

func (s *playlists) TracksRepository(ctx context.Context, playlistId string, refreshSmartPlaylist bool) rest.Repository {
	repo := s.ds.Playlist(ctx)
	tracks := repo.Tracks(playlistId, refreshSmartPlaylist)
	if tracks == nil {
		return nil
	}
	return tracks.(rest.Repository)
}

// savePlaylist creates a new playlist, assigning the owner from context.
// Only Name, Comment, Public, and Rules are user-settable via the REST API.
func (s *playlists) savePlaylist(ctx context.Context, pls *model.Playlist) (string, error) {
	usr, _ := request.UserFrom(ctx)
	pls.OwnerID = usr.ID
	pls.ID = ""               // Force new creation
	pls.Path = ""             // Server-managed (M3U file path)
	pls.Sync = false          // Server-managed (M3U sync flag)
	pls.UploadedImage = ""    // Managed by image upload endpoint
	pls.ExternalImageURL = "" // Managed by M3U import / plugins only
	pls.EvaluatedAt = nil     // Server-managed
	err := s.ds.Playlist(ctx).Put(pls)
	if err != nil {
		return "", err
	}
	return pls.ID, nil
}

// updatePlaylistEntity updates playlist metadata with permission checks.
// Used by the REST API wrapper.
func (s *playlists) updatePlaylistEntity(ctx context.Context, id string, entity *model.Playlist) error {
	current, err := s.checkWritable(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNotFound):
			return rest.ErrNotFound
		case errors.Is(err, model.ErrNotAuthorized):
			return rest.ErrPermissionDenied
		default:
			return err
		}
	}
	usr, _ := request.UserFrom(ctx)
	if !usr.IsAdmin && entity.OwnerID != "" && entity.OwnerID != current.OwnerID {
		return rest.ErrPermissionDenied
	}

	contentChanged := entity.Name != current.Name ||
		entity.Comment != current.Comment ||
		(entity.OwnerID != "" && entity.OwnerID != current.OwnerID) ||
		!rulesEqual(current.Rules, entity.Rules)

	if contentChanged {
		if entity.OwnerID != "" {
			current.OwnerID = entity.OwnerID
		}
		current.Rules = entity.Rules
		if current.Path != "" && current.Sync != entity.Sync {
			current.Sync = entity.Sync
		}
		return s.updateMetadata(ctx, s.ds, current, &entity.Name, &entity.Comment, &entity.Public)
	}

	// Only sync/public changed — skip updatedAt so cover art URLs stay stable
	var cols []string
	if current.Path != "" && current.Sync != entity.Sync {
		current.Sync = entity.Sync
		cols = append(cols, "sync")
	}
	if current.Public != entity.Public {
		current.Public = entity.Public
		cols = append(cols, "public")
	}
	if len(cols) == 0 {
		return nil
	}
	return s.ds.Playlist(ctx).Put(current, cols...)
}

func rulesEqual(a, b *criteria.Criteria) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
