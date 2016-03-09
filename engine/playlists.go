package engine

import (
	"github.com/deluan/gosonic/domain"
)

type Playlists interface {
	GetAll() (*domain.Playlists, error)
}

type playlists struct {
	plsRepo domain.PlaylistRepository
}

func NewPlaylists(pr domain.PlaylistRepository) Playlists {
	return playlists{pr}
}

func (p playlists) GetAll() (*domain.Playlists, error) {
	return p.plsRepo.GetAll(domain.QueryOptions{})
}
