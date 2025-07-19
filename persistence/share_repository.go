package persistence

import (
	"context"
	"errors"
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

func NewShareRepository(ctx context.Context, db dbx.Builder) model.ShareRepository {
	r := &shareRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Share{}, nil)
	r.setSortMappings(map[string]string{
		"username": "username",
	})
	return r
}

func (r *shareRepository) Delete(id string) error {
	err := r.delete(Eq{"id": id})
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

func (r *shareRepository) selectShare(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).Join("user u on u.id = share.user_id").
		Columns("share.*", "user_name as username")
}

func (r *shareRepository) Exists(id string) (bool, error) {
	return r.exists(Eq{"id": id})
}

func (r *shareRepository) Get(id string) (*model.Share, error) {
	sel := r.selectShare().Where(Eq{"share.id": id})
	var res model.Share
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	err = r.loadMedia(&res)
	return &res, err
}

func (r *shareRepository) GetAll(options ...model.QueryOptions) (model.Shares, error) {
	sq := r.selectShare(options...)
	res := model.Shares{}
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	for i := range res {
		err = r.loadMedia(&res[i])
		if err != nil {
			return nil, fmt.Errorf("error loading media for share %s: %w", res[i].ID, err)
		}
	}
	return res, err
}

func (r *shareRepository) loadMedia(share *model.Share) error {
	var err error
	ids := strings.Split(share.ResourceIDs, ",")
	if len(ids) == 0 {
		return nil
	}
	noMissing := func(cond Sqlizer) Sqlizer {
		return And{cond, Eq{"missing": false}}
	}
	switch share.ResourceType {
	case "artist":
		albumRepo := NewAlbumRepository(r.ctx, r.db)
		share.Albums, err = albumRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"album_artist_id": ids}), Sort: "artist"})
		if err != nil {
			return err
		}
		mfRepo := NewMediaFileRepository(r.ctx, r.db)
		share.Tracks, err = mfRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"album_artist_id": ids}), Sort: "artist"})
		return err
	case "album":
		albumRepo := NewAlbumRepository(r.ctx, r.db)
		share.Albums, err = albumRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"album.id": ids})})
		if err != nil {
			return err
		}
		mfRepo := NewMediaFileRepository(r.ctx, r.db)
		share.Tracks, err = mfRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"album_id": ids}), Sort: "album"})
		return err
	case "playlist":
		// Create a context with a fake admin user, to be able to access all playlists
		ctx := request.WithUser(r.ctx, model.User{IsAdmin: true})
		plsRepo := NewPlaylistRepository(ctx, r.db)
		tracks, err := plsRepo.Tracks(ids[0], true).GetAll(model.QueryOptions{Sort: "id", Filters: noMissing(Eq{})})
		if err != nil {
			return err
		}
		if len(tracks) >= 0 {
			share.Tracks = tracks.MediaFiles()
		}
		return nil
	case "media_file":
		mfRepo := NewMediaFileRepository(r.ctx, r.db)
		tracks, err := mfRepo.GetAll(model.QueryOptions{Filters: noMissing(Eq{"media_file.id": ids})})
		share.Tracks = sortByIdPosition(tracks, ids)
		return err
	}
	log.Warn(r.ctx, "Unsupported Share ResourceType", "share", share.ID, "resourceType", share.ResourceType)
	return nil
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

func (r *shareRepository) Update(id string, entity interface{}, cols ...string) error {
	s := entity.(*model.Share)
	// TODO Validate record
	s.ID = id
	s.UpdatedAt = time.Now()
	cols = append(cols, "updated_at")
	_, err := r.put(id, s, cols...)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

func (r *shareRepository) Save(entity interface{}) (string, error) {
	s := entity.(*model.Share)
	// TODO Validate record
	u := loggedUser(r.ctx)
	if s.UserID == "" {
		s.UserID = u.ID
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	id, err := r.put(s.ID, s)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return id, err
}

func (r *shareRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.selectShare(), options...)
}

func (r *shareRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *shareRepository) EntityName() string {
	return "share"
}

func (r *shareRepository) NewInstance() interface{} {
	return &model.Share{}
}

func (r *shareRepository) Read(id string) (interface{}, error) {
	sel := r.selectShare().Where(Eq{"share.id": id})
	var res model.Share
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *shareRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sq := r.selectShare(r.parseRestOptions(r.ctx, options...))
	res := model.Shares{}
	err := r.queryAll(sq, &res)
	return res, err
}

var _ model.ShareRepository = (*shareRepository)(nil)
var _ rest.Repository = (*shareRepository)(nil)
var _ rest.Persistable = (*shareRepository)(nil)
