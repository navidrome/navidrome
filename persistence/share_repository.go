package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/pocketbase/dbx"
)

type shareRepository struct {
	sqlRepository
}

func NewShareRepository(db dbx.Builder) model.ShareRepository {
	r := &shareRepository{}
	r.db = db
	r.registerModel(&model.Share{}, nil)
	r.setSortMappings(map[string]string{
		"username": "username",
	})
	return r
}

func (r *shareRepository) Delete(ctx context.Context, ids ...string) error {
	for _, id := range ids {
		if err := r.deleteOwned(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *shareRepository) selectShare(ctx context.Context, options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(ctx, options...).Join("user u on u.id = share.user_id").
		Columns("share.*", "user_name as username").
		Where(r.addRestriction(ctx))
}

func (r *shareRepository) Exists(ctx context.Context, id string) (bool, error) {
	return r.exists(ctx, r.addRestriction(ctx, And{Eq{"id": id}}))
}

func (r *shareRepository) Get(ctx context.Context, id string) (*model.Share, error) {
	sel := r.selectShare(ctx).Where(Eq{"share.id": id})
	var res model.Share
	err := r.queryOne(ctx, sel, &res)
	if err != nil {
		return nil, err
	}
	err = r.loadMedia(ctx, &res)
	return &res, err
}

func (r *shareRepository) GetAll(ctx context.Context, options ...model.QueryOptions) (model.Shares, error) {
	sq := r.selectShare(ctx, options...)
	res := model.Shares{}
	err := r.queryAll(ctx, sq, &res)
	if err != nil {
		return nil, err
	}
	for i := range res {
		err = r.loadMedia(ctx, &res[i])
		if err != nil {
			return nil, fmt.Errorf("error loading media for share %s: %w", res[i].ID, err)
		}
	}
	return res, err
}

func (r *shareRepository) loadMedia(ctx context.Context, share *model.Share) error {
	ids := strings.Split(share.ResourceIDs, ",")
	if len(ids) == 0 {
		return nil
	}
	noMissing := func(cond Sqlizer) Sqlizer {
		return And{cond, Eq{"missing": false}}
	}
	// Load as the share owner so their library access is applied, whoever renders the share.
	ownerCtx, err := r.ownerContext(ctx, share)
	if err != nil {
		return err
	}
	switch share.ResourceType {
	case "artist":
		// Match by album-artist participation, not the deprecated album_artist_id
		// column (first album artist only), so co-album-artists are included too.
		albumRepo := NewAlbumRepository(ownerCtx, r.db)
		share.Albums, err = albumRepo.GetAll(model.QueryOptions{Filters: noMissing(ParticipantIDFilter("album", ids, model.RoleAlbumArtist)), Sort: "artist"})
		if err != nil {
			return err
		}
		mfRepo := NewMediaFileRepository(ownerCtx, r.db)
		share.Tracks, err = mfRepo.GetAll(model.QueryOptions{Filters: noMissing(ParticipantIDFilter("media_file", ids, model.RoleAlbumArtist)), Sort: "artist"})
		return err
	case "album":
		albumRepo := NewAlbumRepository(ownerCtx, r.db)
		share.Albums, err = albumRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"album.id": ids})})
		if err != nil {
			return err
		}
		mfRepo := NewMediaFileRepository(ownerCtx, r.db)
		share.Tracks, err = mfRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"album_id": ids}), Sort: "album"})
		return err
	case "playlist":
		plsRepo := NewPlaylistRepository(ownerCtx, r.db)
		// Tracks returns nil when the playlist is no longer visible to the owner
		// (e.g. it was made private after the share was created); leave the share
		// with no tracks rather than exposing it.
		trackRepo := plsRepo.Tracks(ids[0], true)
		if trackRepo == nil {
			return nil
		}
		tracks, err := trackRepo.GetAll(model.QueryOptions{Sort: "id", Filters: noMissing(Eq{})})
		if err != nil {
			return err
		}
		share.Tracks = tracks.MediaFiles()
		return nil
	case "media_file":
		mfRepo := NewMediaFileRepository(ownerCtx, r.db)
		tracks, err := mfRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"media_file.id": ids})})
		share.Tracks = sortByIdPosition(tracks, ids)
		return err
	}
	log.Warn(ctx, "Unsupported Share ResourceType", "share", share.ID, "resourceType", share.ResourceType)
	return nil
}

// ownerContext returns a context scoped to the share owner, so repository
// queries apply the owner's library access when a public share is rendered.
func (r *shareRepository) ownerContext(ctx context.Context, share *model.Share) (context.Context, error) {
	owner, err := NewUserRepository(r.db).Get(ctx, share.UserID)
	if err != nil {
		return nil, fmt.Errorf("loading share owner %q: %w", share.UserID, err)
	}
	if owner == nil {
		return nil, fmt.Errorf("share owner %q not found", share.UserID)
	}
	return request.WithUser(ctx, *owner), nil
}

func sortByIdPosition(mfs model.MediaFiles, ids []string) model.MediaFiles {
	m := map[string]int{}
	for i, mf := range mfs {
		m[mf.ID] = i
	}
	var sorted model.MediaFiles
	for _, id := range ids {
		if idx, ok := m[id]; ok {
			sorted = append(sorted, mfs[idx])
		}
	}
	return sorted
}

func (r *shareRepository) Update(ctx context.Context, id string, entity model.Share, cols ...string) error {
	s := &entity
	s.ID = id
	s.UpdatedAt = time.Now()
	if len(cols) > 0 {
		cols = append(cols, "updated_at")
	}
	return r.updateOwned(ctx, id, s, cols...)
}

func (r *shareRepository) Save(ctx context.Context, s *model.Share) (string, error) {
	// TODO Validate record
	// Owner is server-managed: for an authenticated request, never trust a
	// client-supplied UserID, as it drives the share's library-access context.
	u := loggedUser(ctx)
	if u.ID != invalidUserId || s.UserID == "" {
		s.UserID = u.ID
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	return r.put(ctx, s.ID, s)
}

func (r *shareRepository) CountAll(ctx context.Context, options ...model.QueryOptions) (int64, error) {
	return r.count(ctx, r.selectShare(ctx), options...)
}

func (r *shareRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *shareRepository) Read(ctx context.Context, id string) (*model.Share, error) {
	sel := r.selectShare(ctx).Where(Eq{"share.id": id})
	var res model.Share
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

func (r *shareRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Share, error) {
	sq := r.selectShare(ctx, r.parseRestOptions(ctx, options...))
	res := model.Shares{}
	err := r.queryAll(ctx, sq, &res)
	return res, err
}

var _ model.ShareRepository = (*shareRepository)(nil)
var _ rest.Repository[model.Share] = (*shareRepository)(nil)
var _ rest.Persistable[model.Share] = (*shareRepository)(nil)
