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
	share, err := repo.Get(id)
	if err != nil {
		return nil, err
	}
	if !share.ExpiresAt.IsZero() && share.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrNotAvailable
	}
	share.LastVisitedAt = time.Now()
	share.VisitCount++

	err = repo.(rest.Persistable).Update(id, share, "last_visited_at", "visit_count")
	if err != nil {
		log.Warn(ctx, "Could not increment visit count for share", "share", share.ID)
	}
	return share, nil
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

	firstId := strings.SplitN(s.ResourceIDs, ",", 2)[0]
	v, err := model.GetEntityByID(r.ctx, r.ds, firstId)
	if err != nil {
		return "", err
	}
	switch v.(type) {
	case *model.Album:
		s.ResourceType = "album"
		s.Contents = r.shareContentsFromAlbums(s.ID, s.ResourceIDs)
	case *model.Playlist:
		s.ResourceType = "playlist"
		s.Contents = r.shareContentsFromPlaylist(s.ID, s.ResourceIDs)
	case *model.MediaFile:
		s.ResourceType = "media_file"
	default:
		log.Error(r.ctx, "Invalid Resource ID", "id", firstId)
		return "", model.ErrNotFound
	}

	id, err = r.Persistable.Save(s)
	return id, err
}

func (r *shareRepositoryWrapper) Update(id string, entity interface{}, _ ...string) error {
	cols := []string{"description"}

	// TODO Better handling of Share expiration
	if !entity.(*model.Share).ExpiresAt.IsZero() {
		cols = append(cols, "expires_at")
	}
	return r.Persistable.Update(id, entity, cols...)
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
func (r *shareRepositoryWrapper) shareContentsFromPlaylist(shareID string, id string) string {
	pls, err := r.ds.Playlist(r.ctx).Get(id)
	if err != nil {
		log.Error(r.ctx, "Error retrieving album names for share", "share", shareID, err)
		return ""
	}
	content := pls.Name
	if len(content) > 30 {
		content = content[:26] + "..."
	}
	return content
}
