package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenreRepository", func() {
	var repo model.GenreRepository

	BeforeEach(func() {
		repo = NewGenreRepository(orm.NewOrm())
	})

	It("returns all records", func() {
		genres, err := repo.GetAll()
		Expect(err).To(BeNil())
		Expect(genres).To(ContainElement(model.Genre{Name: "Rock", AlbumCount: 2, SongCount: 2}))
		Expect(genres).To(ContainElement(model.Genre{Name: "Electronic", AlbumCount: 1, SongCount: 2}))
	})
})
