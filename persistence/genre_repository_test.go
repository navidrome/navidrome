package persistence_test

import (
	"context"

	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenreRepository", func() {
	var repo model.GenreRepository

	BeforeEach(func() {
		repo = persistence.NewGenreRepository(log.NewContext(context.TODO()), orm.NewOrm())
	})

	It("returns all records", func() {
		genres, err := repo.GetAll()
		Expect(err).To(BeNil())
		Expect(genres).To(ConsistOf(
			model.Genre{ID: "gn-1", Name: "Electronic", AlbumCount: 1, SongCount: 2},
			model.Genre{ID: "gn-2", Name: "Rock", AlbumCount: 3, SongCount: 3},
		))
	})
})
