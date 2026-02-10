package playlists

import (
	"context"
	"errors"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
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

func (r *playlistRepositoryWrapper) Update(id string, entity any, cols ...string) error {
	return r.service.updatePlaylistEntity(r.ctx, id, entity.(*model.Playlist), cols...)
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
func (s *playlists) savePlaylist(ctx context.Context, pls *model.Playlist) (string, error) {
	usr, _ := request.UserFrom(ctx)
	pls.OwnerID = usr.ID
	pls.ID = "" // Force new creation
	err := s.ds.Playlist(ctx).Put(pls)
	if err != nil {
		return "", err
	}
	return pls.ID, nil
}

// updatePlaylistEntity updates playlist metadata with permission checks.
// Used by the REST API wrapper.
func (s *playlists) updatePlaylistEntity(ctx context.Context, id string, entity *model.Playlist, cols ...string) error {
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
	// Apply ownership change (admin only)
	if entity.OwnerID != "" {
		current.OwnerID = entity.OwnerID
	}
	return s.updateMetadata(ctx, s.ds, current, &entity.Name, &entity.Comment, &entity.Public)
}
