package engine

import (
	"context"
	"errors"

	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browser", func() {
	var repo *mockGenreRepository
	var b Browser

	BeforeEach(func() {
		repo = &mockGenreRepository{data: model.Genres{
			{Name: "Rock", SongCount: 1000, AlbumCount: 100},
			{Name: "", SongCount: 13, AlbumCount: 13},
			{Name: "Electronic", SongCount: 4000, AlbumCount: 40},
		}}
		var ds = &persistence.MockDataStore{MockedGenre: repo}
		b = &browser{ds: ds}
	})

	It("returns sorted data", func() {
		Expect(b.GetGenres(context.TODO())).To(Equal(model.Genres{
			{Name: "<Empty>", SongCount: 13, AlbumCount: 13},
			{Name: "Electronic", SongCount: 4000, AlbumCount: 40},
			{Name: "Rock", SongCount: 1000, AlbumCount: 100},
		}))
	})

	It("bubbles up errors", func() {
		repo.err = errors.New("generic error")
		_, err := b.GetGenres(context.TODO())
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
