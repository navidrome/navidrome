package engine

import (
	"errors"

	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browser", func() {
	var repo *mockGenreRepository
	var b Browser

	BeforeSuite(func() {
		repo = &mockGenreRepository{data: model.Genres{
			{Name: "Rock", SongCount: 1000, AlbumCount: 100},
			{Name: "", SongCount: 13, AlbumCount: 13},
			{Name: "Electronic", SongCount: 4000, AlbumCount: 40},
		}}
		b = &browser{genreRepo: repo}
	})

	It("returns sorted data", func() {
		Expect(b.GetGenres()).To(Equal(model.Genres{
			{Name: "<Empty>", SongCount: 13, AlbumCount: 13},
			{Name: "Electronic", SongCount: 4000, AlbumCount: 40},
			{Name: "Rock", SongCount: 1000, AlbumCount: 100},
		}))
	})

	It("bubbles up errors", func() {
		repo.err = errors.New("generic error")
		_, err := b.GetGenres()
		Expect(err).ToNot(BeNil())
	})
})

type mockGenreRepository struct {
	data model.Genres
	err  error
}

func (r *mockGenreRepository) GetAll() (model.Genres, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.data, nil
}
