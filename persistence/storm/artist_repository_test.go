package storm

import (
	"github.com/cloudsonic/sonic-server/domain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {
	var repo domain.ArtistRepository

	BeforeEach(func() {
		Db().Drop(&_Artist{})
		repo = NewArtistRepository()
	})

	It("saves and retrieves data", func() {
		artist := &domain.Artist{
			ID:         "1",
			Name:       "Saara Saara",
			AlbumCount: 2,
		}
		Expect(repo.Put(artist)).To(BeNil())

		Expect(repo.Get("1")).To(Equal(artist))
	})

	Describe("PurgeInactive", func() {
		var data domain.Artists

		BeforeEach(func() {
			data = domain.Artists{
				{ID: "1", Name: "Saara Saara"},
				{ID: "2", Name: "Kraftwerk"},
				{ID: "3", Name: "The Beatles"},
			}
			for _, a := range data {
				repo.Put(&a)
			}
		})

		It("purges inactive records", func() {
			active := domain.Artists{{ID: "1"}, {ID: "3"}}
			Expect(repo.PurgeInactive(active)).To(Equal([]string{"2"}))
		})

		It("doesn't delete anything if all is active", func() {
			active := domain.Artists{{ID: "1"}, {ID: "2"}, {ID: "3"}}
			Expect(repo.PurgeInactive(active)).To(BeEmpty())
		})
	})

})
