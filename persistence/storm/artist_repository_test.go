package storm

import (
	"github.com/cloudsonic/sonic-server/domain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {
	var repo domain.ArtistRepository

	BeforeEach(func() {
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

	It("purges inactive records", func() {
		data := domain.Artists{
			{ID: "1", Name: "Saara Saara"},
			{ID: "2", Name: "Kraftwerk"},
			{ID: "3", Name: "The Beatles"},
		}
		active := domain.Artists{
			{ID: "1"}, {ID: "3"},
		}
		for _, a := range data {
			repo.Put(&a)
		}
		Expect(repo.PurgeInactive(active)).To(Equal([]string{"2"}))
	})
})
