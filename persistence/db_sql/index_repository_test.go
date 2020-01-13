package db_sql

import (
	"github.com/cloudsonic/sonic-server/domain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artist Index", func() {
	var repo domain.ArtistIndexRepository

	BeforeEach(func() {
		repo = NewArtistIndexRepository()
		err := repo.DeleteAll()
		if err != nil {
			panic(err)
		}
	})

	It("successfully persists data", func() {
		idx1 := domain.ArtistIndex{
			ID: "D",
			Artists: domain.ArtistInfos{
				{ArtistID: "4", Artist: "Doobie Brothers", AlbumCount: 2},
				{ArtistID: "3", Artist: "The Doors", AlbumCount: 5},
			},
		}
		idx2 := domain.ArtistIndex{
			ID: "S",
			Artists: domain.ArtistInfos{
				{ArtistID: "1", Artist: "Saara Saara", AlbumCount: 3},
				{ArtistID: "2", Artist: "Seu Jorge", AlbumCount: 1},
			},
		}

		Expect(repo.Put(&idx1)).To(BeNil())
		Expect(repo.Put(&idx2)).To(BeNil())
		Expect(repo.Get("D")).To(Equal(&idx1))
		Expect(repo.Get("S")).To(Equal(&idx2))
		Expect(repo.GetAll()).To(Equal(domain.ArtistIndexes{idx1, idx2}))
		Expect(repo.CountAll()).To(Equal(int64(2)))
		Expect(repo.DeleteAll()).To(BeNil())
		Expect(repo.CountAll()).To(Equal(int64(0)))
	})
})
