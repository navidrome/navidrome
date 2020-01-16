package persistence

import (
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artist Index", func() {
	var repo model.ArtistIndexRepository

	BeforeEach(func() {
		repo = NewArtistIndexRepository()
		err := repo.DeleteAll()
		if err != nil {
			panic(err)
		}
	})

	It("successfully persists data", func() {
		idx1 := model.ArtistIndex{
			ID: "D",
			Artists: model.ArtistInfos{
				{ArtistID: "4", Artist: "Doobie Brothers", AlbumCount: 2},
				{ArtistID: "3", Artist: "The Doors", AlbumCount: 5},
			},
		}
		idx2 := model.ArtistIndex{
			ID: "S",
			Artists: model.ArtistInfos{
				{ArtistID: "1", Artist: "Saara Saara", AlbumCount: 3},
				{ArtistID: "2", Artist: "Seu Jorge", AlbumCount: 1},
			},
		}

		Expect(repo.Put(&idx1)).To(BeNil())
		Expect(repo.Put(&idx2)).To(BeNil())
		Expect(repo.GetAll()).To(Equal(model.ArtistIndexes{idx1, idx2}))
	})
})
