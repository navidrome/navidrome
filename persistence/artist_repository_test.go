package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {
	var repo model.ArtistRepository

	BeforeEach(func() {
		repo = NewArtistRepository(orm.NewOrm())
	})

	Describe("Put/Get", func() {
		It("saves and retrieves data", func() {
			Expect(repo.Get("1")).To(Equal(&artistSaaraSaara))
		})

		It("overrides data if ID already exists", func() {
			Expect(repo.Put(&model.Artist{ID: "1", Name: "Saara Saara is The Best!", AlbumCount: 3})).To(BeNil())
			Expect(repo.Get("1")).To(Equal(&model.Artist{ID: "1", Name: "Saara Saara is The Best!", AlbumCount: 3}))
		})

		It("returns ErrNotFound when the ID does not exist", func() {
			_, err := repo.Get("999")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetIndex", func() {
		It("returns the index", func() {
			idx, err := repo.GetIndex()
			Expect(err).To(BeNil())
			Expect(idx).To(Equal(model.ArtistIndexes{
				{
					ID: "B",
					Artists: model.Artists{
						artistBeatles,
					},
				},
				{
					ID: "K",
					Artists: model.Artists{
						artistKraftwerk,
					},
				},
				{
					ID: "S",
					Artists: model.Artists{
						{ID: "1", Name: "Saara Saara is The Best!", AlbumCount: 3},
					},
				},
			}))
		})
	})
})
