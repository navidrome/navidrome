package core

import (
	"context"

	"github.com/deluan/rest"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/navidrome/navidrome/model"
)

type Share interface {
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

func (s *shareService) NewRepository(ctx context.Context) rest.Repository {
	repo := s.ds.Share(ctx)
	wrapper := &shareRepositoryWrapper{
		ShareRepository: repo,
		Repository:      repo.(rest.Repository),
		Persistable:     repo.(rest.Persistable),
	}
	return wrapper
}

type shareRepositoryWrapper struct {
	model.ShareRepository
	rest.Repository
	rest.Persistable
}

func (r *shareRepositoryWrapper) Save(entity interface{}) (string, error) {
	s := entity.(*model.Share)
	id, err := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 9)
	if err != nil {
		return "", err
	}
	s.Name = id
	id, err = r.Persistable.Save(s)
	return id, err
}

func (r *shareRepositoryWrapper) Update(entity interface{}, _ ...string) error {
	return r.Persistable.Update(entity, "description")
}
