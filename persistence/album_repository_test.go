package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlbumRepository", func() {
	var repo model.AlbumRepository

	BeforeEach(func() {
		repo = NewAlbumRepository(orm.NewOrm())
	})

	Describe("GetAll", func() {
		It("returns all records", func() {
			Expect(repo.GetAll()).To(Equal(testAlbums))
		})

		It("returns all records sorted", func() {
			Expect(repo.GetAll(model.QueryOptions{SortBy: "Name"})).To(Equal(model.Albums{
				albumAbbeyRoad,
				albumRadioactivity,
				albumSgtPeppers,
			}))
		})

		It("returns all records sorted desc", func() {
			Expect(repo.GetAll(model.QueryOptions{SortBy: "Name", Desc: true})).To(Equal(model.Albums{
				albumSgtPeppers,
				albumRadioactivity,
				albumAbbeyRoad,
			}))
		})

		It("paginates the result", func() {
			Expect(repo.GetAll(model.QueryOptions{Offset: 1, Size: 1})).To(Equal(model.Albums{
				albumAbbeyRoad,
			}))
		})
	})

	Describe("GetAllIds", func() {
		It("returns all records", func() {
			Expect(repo.GetAllIds()).To(ConsistOf("1", "2", "3"))
		})
	})

	Describe("GetStarred", func() {
		It("returns all starred records", func() {
			Expect(repo.GetStarred(model.QueryOptions{})).To(Equal(model.Albums{
				albumRadioactivity,
			}))
		})
	})

	Describe("FindByArtist", func() {
		It("returns all records from a given ArtistID", func() {
			Expect(repo.FindByArtist("1")).To(Equal(model.Albums{
				albumAbbeyRoad,
				albumSgtPeppers,
			}))
		})
	})

})
