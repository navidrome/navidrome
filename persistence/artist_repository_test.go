package persistence

import (
	"context"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {
	var repo model.ArtistRepository

	BeforeEach(func() {
		repo = NewArtistRepository(context.Background(), orm.NewOrm())
	})

	Describe("Count", func() {
		It("returns the number of artists in the DB", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})
	})

	Describe("Exist", func() {
		It("returns true for an artist that is in the DB", func() {
			Expect(repo.Exists("3")).To(BeTrue())
		})
		It("returns false for an artist that is in the DB", func() {
			Expect(repo.Exists("666")).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("saves and retrieves data", func() {
			Expect(repo.Get("2")).To(Equal(&artistKraftwerk))
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
			}))
		})
	})
})
