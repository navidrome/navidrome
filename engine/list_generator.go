package engine

import (
	"math/rand"

	"github.com/deluan/gosonic/domain"
)

// TODO Use Entries instead of Albums
type ListGenerator interface {
	GetNewest(offset int, size int) (*domain.Albums, error)
	GetRecent(offset int, size int) (*domain.Albums, error)
	GetFrequent(offset int, size int) (*domain.Albums, error)
	GetHighest(offset int, size int) (*domain.Albums, error)
	GetRandom(offset int, size int) (*domain.Albums, error)
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
	return g.albumRepo.GetAll(qo)
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

func (g listGenerator) GetRandom(offset int, size int) (*domain.Albums, error) {
	ids, err := g.albumRepo.GetAllIds()
	if err != nil {
		return nil, err
	}
	r := make(domain.Albums, len(*ids))
	perm := rand.Perm(len(*ids))

	for i, v := range perm {
		al, err := g.albumRepo.Get((*ids)[v])
		if err != nil {
			return nil, err
		}
		r[i] = *al
	}
	return &r, nil
}
