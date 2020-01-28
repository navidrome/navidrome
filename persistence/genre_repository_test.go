package persistence_test

import (
	"context"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenreRepository", func() {
	var repo model.GenreRepository

	BeforeEach(func() {
		repo = persistence.NewGenreRepository(context.Background(), orm.NewOrm())
	})

	It("returns all records", func() {
		genres, err := repo.GetAll()
		Expect(err).To(BeNil())
		Expect(genres).To(ContainElement(model.Genre{Name: "Rock", AlbumCount: 2, SongCount: 2}))
		Expect(genres).To(ContainElement(model.Genre{Name: "Electronic", AlbumCount: 1, SongCount: 2}))
	})
})
