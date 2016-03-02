package repositories

import (
	"github.com/deluan/gosonic/models"
	"errors"
	"sort"
	"github.com/deluan/gosonic/utils"
)

type ArtistIndex interface {
	Put(m *models.ArtistIndex) error
	Get(id string) (*models.ArtistIndex, error)
	GetAll() ([]models.ArtistIndex, error)
}

type artistIndex struct {
	BaseRepository
}

func NewArtistIndexRepository() ArtistIndex {
	r := &artistIndex{}
	r.init("index", &models.ArtistIndex{})
	return r
}

func (r *artistIndex) Put(m *models.ArtistIndex) error {
	if m.Id == "" {
		return errors.New("Id is not set")
	}
	sort.Sort(byArtistName(m.Artists))
	return r.saveOrUpdate(m.Id, m)
}

func (r *artistIndex) Get(id string) (*models.ArtistIndex, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*models.ArtistIndex), err
}

func (r *artistIndex) GetAll() ([]models.ArtistIndex, error) {
	var indices = make([]models.ArtistIndex, 0)
	err := r.loadAll(&indices, "")
	return indices, err
}

type byArtistName []models.ArtistInfo

func (a byArtistName) Len() int {
	return len(a)
}
func (a byArtistName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a byArtistName) Less(i, j int) bool {
	return utils.NoArticle(a[i].Artist) < utils.NoArticle(a[j].Artist)
}
