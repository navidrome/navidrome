package core

import (
	"context"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

type Share interface {
	Load(ctx context.Context, id string) (*model.Share, error)
	NewRepository(ctx context.Context) rest.Repository
}

func NewShare(ds model.DataStore) Share {
	return &shareService{
		ds: ds,
	}
}

type shareService struct {
	ds model.DataStore
}

func (s *shareService) Load(ctx context.Context, id string) (*model.Share, error) {
	repo := s.ds.Share(ctx)
	entity, err := repo.(rest.Repository).Read(id)
	if err != nil {
		return nil, err
	}
	share := entity.(*model.Share)
	share.LastVisitedAt = time.Now()
	share.VisitCount++

	err = repo.(rest.Persistable).Update(id, share, "last_visited_at", "visit_count")
	if err != nil {
		log.Warn(ctx, "Could not increment visit count for share", "share", share.ID)
	}

	idList := strings.Split(share.ResourceIDs, ",")
	switch share.ResourceType {
	case "album":
		share.Tracks, err = s.loadMediafiles(ctx, squirrel.Eq{"album_id": idList}, "album")
	}
	if err != nil {
		return nil, err
	}
	return entity.(*model.Share), nil
}

func (s *shareService) loadMediafiles(ctx context.Context, filter squirrel.Eq, sort string) ([]model.ShareTrack, error) {
	all, err := s.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: filter, Sort: sort})
	if err != nil {
		return nil, err
	}
	return slice.Map(all, func(mf model.MediaFile) model.ShareTrack {
		return model.ShareTrack{
			ID:        mf.ID,
			Title:     mf.Title,
			Artist:    mf.Artist,
			Album:     mf.Album,
			Duration:  mf.Duration,
			UpdatedAt: mf.UpdatedAt,
		}
	}), nil
}

func (s *shareService) NewRepository(ctx context.Context) rest.Repository {
	repo := s.ds.Share(ctx)
	wrapper := &shareRepositoryWrapper{
		ctx:             ctx,
		ShareRepository: repo,
		Repository:      repo.(rest.Repository),
		Persistable:     repo.(rest.Persistable),
		ds:              s.ds,
	}
	return wrapper
}

type shareRepositoryWrapper struct {
	model.ShareRepository
	rest.Repository
	rest.Persistable
	ctx context.Context
	ds  model.DataStore
}

func (r *shareRepositoryWrapper) newId() (string, error) {
	for {
		id, err := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 10)
		if err != nil {
			return "", err
		}
		exists, err := r.Exists(id)
		if err != nil {
			return "", err
		}
		if !exists {
			return id, nil
		}
	}
}

func (r *shareRepositoryWrapper) Save(entity interface{}) (string, error) {
	s := entity.(*model.Share)
	id, err := r.newId()
	if err != nil {
		return "", err
	}
	s.ID = id
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt = time.Now().Add(365 * 24 * time.Hour)
	}
	if s.ResourceType == "album" {
		s.Contents = r.shareContentsFromAlbums(s.ID, s.ResourceIDs)
	}
	id, err = r.Persistable.Save(s)
	return id, err
}

func (r *shareRepositoryWrapper) Update(id string, entity interface{}, _ ...string) error {
	return r.Persistable.Update(id, entity, "description", "expires_at")
}

func (r *shareRepositoryWrapper) shareContentsFromAlbums(shareID string, ids string) string {
	all, err := r.ds.Album(r.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"id": ids}})
	if err != nil {
		log.Error(r.ctx, "Error retrieving album names for share", "share", shareID, err)
		return ""
	}
	names := slice.Map(all, func(a model.Album) string { return a.Name })
	content := strings.Join(names, ", ")
	if len(content) > 30 {
		content = content[:26] + "..."
	}
	return content
}
