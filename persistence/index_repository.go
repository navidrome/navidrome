package persistence

import (
	"errors"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"sort"
)

type artistIndexRepository struct {
	baseRepository
}

func NewArtistIndexRepository() domain.ArtistIndexRepository {
	r := &artistIndexRepository{}
	r.init("index", &domain.ArtistIndex{})
	return r
}

func (r *artistIndexRepository) Put(m *domain.ArtistIndex) error {
	if m.Id == "" {
		return errors.New("Id is not set")
	}
	sort.Sort(byArtistName(m.Artists))
	return r.saveOrUpdate(m.Id, m)
}

func (r *artistIndexRepository) Get(id string) (*domain.ArtistIndex, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.ArtistIndex), err
}

func (r *artistIndexRepository) GetAll() ([]domain.ArtistIndex, error) {
	var indices = make([]domain.ArtistIndex, 0)
	err := r.loadAll(&indices, "", true)
	return indices, err
}

type byArtistName []domain.ArtistInfo

func (a byArtistName) Len() int {
	return len(a)
}
func (a byArtistName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a byArtistName) Less(i, j int) bool {
	return utils.NoArticle(a[i].Artist) < utils.NoArticle(a[j].Artist)
}

var _ domain.ArtistIndexRepository = (*artistIndexRepository)(nil)