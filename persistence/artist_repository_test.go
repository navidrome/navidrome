package persistence

import (
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {
	var repo model.ArtistRepository

	BeforeEach(func() {
		repo = NewArtistRepository()
	})

	It("saves and retrieves data", func() {
		Expect(repo.Get("1")).To(Equal(&model.Artist{ID: "1", Name: "Saara Saara", AlbumCount: 2}))
	})

	It("overrides data if ID already exists", func() {
		Expect(repo.Put(&model.Artist{ID: "1", Name: "Saara Saara is The Best!", AlbumCount: 3})).To(BeNil())
		Expect(repo.Get("1")).To(Equal(&model.Artist{ID: "1", Name: "Saara Saara is The Best!", AlbumCount: 3}))
	})

	It("returns ErrNotFound when the ID does not exist", func() {
		_, err := repo.Get("999")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	Describe("PurgeInactive", func() {
		BeforeEach(func() {
			for _, a := range testArtists {
				repo.Put(&a)
			}
		})

		It("purges inactive records", func() {
			active := model.Artists{{ID: "1"}, {ID: "3"}}

			Expect(repo.PurgeInactive(active)).To(BeNil())

			Expect(repo.CountAll()).To(Equal(int64(2)))
			Expect(repo.Exists("2")).To(BeFalse())
		})

		It("doesn't delete anything if all is active", func() {
			active := model.Artists{{ID: "1"}, {ID: "2"}, {ID: "3"}}

			Expect(repo.PurgeInactive(active)).To(BeNil())

			Expect(repo.CountAll()).To(Equal(int64(3)))
			Expect(repo.Exists("1")).To(BeTrue())
		})
	})
})
