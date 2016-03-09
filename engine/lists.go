package engine

import (
	"github.com/deluan/gosonic/domain"
)

type ListGenerator interface {
	GetNewest(offset int, size int) (*domain.Albums, error)
	GetRecent(offset int, size int) (*domain.Albums, error)
	GetFrequent(offset int, size int) (*domain.Albums, error)
	GetHighest(offset int, size int) (*domain.Albums, error)
}

func NewListGenerator(alr domain.AlbumRepository) ListGenerator {
	return listGenerator{alr}
}

type listGenerator struct {
	albumRepo domain.AlbumRepository
}

func (g listGenerator) query(qo domain.QueryOptions, offset int, size int) (*domain.Albums, error) {
	qo.Offset = offset
	qo.Size = size
	als, err := g.albumRepo.GetAll(qo)
	return &als, err
}

func (g listGenerator) GetNewest(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "CreatedAt", Desc: true, Alpha: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetRecent(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "PlayDate", Desc: true, Alpha: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetFrequent(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "PlayCount", Desc: true}
	return g.query(qo, offset, size)
}

func (g listGenerator) GetHighest(offset int, size int) (*domain.Albums, error) {
	qo := domain.QueryOptions{SortBy: "Rating", Desc: true}
	return g.query(qo, offset, size)
}
