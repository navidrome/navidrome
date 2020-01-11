package storm

import (
	"github.com/cloudsonic/sonic-server/domain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlbumRepository", func() {
	var repo domain.AlbumRepository

	BeforeEach(func() {
		repo = NewAlbumRepository()
	})

	Describe("GetAll", func() {
		It("returns all records", func() {
			Expect(repo.GetAll(domain.QueryOptions{})).To(Equal(testAlbums))
		})

		It("returns all records sorted", func() {
			Expect(repo.GetAll(domain.QueryOptions{SortBy: "Name"})).To(Equal(domain.Albums{
				{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
				{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true},
				{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1"},
			}))
		})

		It("returns all records sorted desc", func() {
			Expect(repo.GetAll(domain.QueryOptions{SortBy: "Name", Desc: true})).To(Equal(domain.Albums{
				{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1"},
				{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true},
				{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
			}))
		})

		It("paginates the result", func() {
			Expect(repo.GetAll(domain.QueryOptions{Offset: 1, Size: 1})).To(Equal(domain.Albums{
				{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
			}))
		})
	})

	Describe("GetAllIds", func() {
		It("returns all records", func() {
			Expect(repo.GetAllIds()).To(Equal([]string{"1", "2", "3"}))
		})
	})

	Describe("GetStarred", func() {
		It("returns all starred records", func() {
			Expect(repo.GetStarred(domain.QueryOptions{})).To(Equal(domain.Albums{
				{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true},
			}))
		})
	})

	Describe("FindByArtist", func() {
		It("returns all records from a given ArtistID", func() {
			Expect(repo.FindByArtist("1")).To(Equal(domain.Albums{
				{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1"},
				{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
			}))
		})
	})

})
