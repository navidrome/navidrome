package playlists

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/slice"
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
//
// cols names the fields the client actually sent in the JSON body (extracted by
// rest.Put). When non-empty, fields outside cols are not considered changed and
// are left untouched — this prevents partial requests like bulk "Make Public"
// (body: {"public": true}) from wiping fields that just happen to be zero in
// the deserialized entity (see issue #5541). An empty cols means "treat the
// entity as a complete record" — preserved for callers that don't use the REST
// wrapper.
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

	sent := sentFields(cols)

	usr, _ := request.UserFrom(ctx)
	ownerChanged := sent("ownerId") && entity.OwnerID != "" && entity.OwnerID != current.OwnerID
	if !usr.IsAdmin && ownerChanged {
		return rest.ErrPermissionDenied
	}

	nameChanged := sent("name") && entity.Name != current.Name
	commentChanged := sent("comment") && entity.Comment != current.Comment
	rulesChanged := sent("rules") && !rulesEqual(current.Rules, entity.Rules)

	if nameChanged || commentChanged || ownerChanged || rulesChanged {
		return s.applyContentUpdate(ctx, current, entity, sent,
			nameChanged, commentChanged, ownerChanged, rulesChanged)
	}
	return s.applyFlagsOnly(ctx, current, entity, sent)
}

// applyContentUpdate handles updates that change at least one of name/comment/
// owner/rules. It goes through updateMetadata, which always bumps updatedAt
// (invalidating cached cover-art URLs). namePtr/commentPtr are nil when the
// field is absent from the request OR present-but-unchanged (so updateMetadata
// skips them); publicPtr is nil only when public is absent from the request
// (an idempotent public value is still forwarded).
func (s *playlists) applyContentUpdate(ctx context.Context, current, entity *model.Playlist,
	sent func(string) bool, nameChanged, commentChanged, ownerChanged, rulesChanged bool,
) error {
	if ownerChanged {
		current.OwnerID = entity.OwnerID
	}
	if rulesChanged {
		current.Rules = entity.Rules
	}
	if sent("sync") && current.Path != "" && current.Sync != entity.Sync {
		current.Sync = entity.Sync
	}
	var namePtr, commentPtr *string
	var publicPtr *bool
	if nameChanged {
		namePtr = &entity.Name
	}
	if commentChanged {
		commentPtr = &entity.Comment
	}
	if sent("public") {
		publicPtr = &entity.Public
	}
	return s.updateMetadata(ctx, s.ds, current, namePtr, commentPtr, publicPtr)
}

// applyFlagsOnly handles updates that only toggle sync/public — skips
// updatedAt so cover art URLs stay stable.
func (s *playlists) applyFlagsOnly(ctx context.Context, current, entity *model.Playlist,
	sent func(string) bool,
) error {
	var updateCols []string
	if sent("sync") && current.Path != "" && current.Sync != entity.Sync {
		current.Sync = entity.Sync
		updateCols = append(updateCols, "sync")
	}
	if sent("public") && current.Public != entity.Public {
		current.Public = entity.Public
		updateCols = append(updateCols, "public")
	}
	if len(updateCols) == 0 {
		return nil
	}
	return s.ds.Playlist(ctx).Put(current, updateCols...)
}

// sentFields returns a predicate that reports whether a JSON field was present
// in the request body. Matching is case-insensitive to mirror Go's json
// decoder, which populates struct fields from case-variant keys like
// {"Name":"x"} or {"OWNERID":"y"}. An empty cols list means "treat the entity
// as a full record" — every field is considered sent.
func sentFields(cols []string) func(string) bool {
	if len(cols) == 0 {
		return func(string) bool { return true }
	}
	set := slice.ToMap(cols, func(c string) (string, struct{}) { return strings.ToLower(c), struct{}{} })
	return func(field string) bool {
		_, ok := set[strings.ToLower(field)]
		return ok
	}
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
