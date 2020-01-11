package db_storm

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
		Expect(repo.Get("1")).To(Equal(&domain.Artist{ID: "1", Name: "Saara Saara"}))
	})

	It("returns ErrNotFound when the ID does not exist", func() {
		_, err := repo.Get("999")
		Expect(err).To(MatchError(domain.ErrNotFound))
	})

	Describe("PurgeInactive", func() {
		var data domain.Artists

		BeforeEach(func() {
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
