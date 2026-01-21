package persistence

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Annotation Filters", func() {
	var (
		albumRepo              *albumRepository
		albumWithoutAnnotation model.Album
	)

	BeforeEach(func() {
		ctx := request.WithUser(context.Background(), model.User{ID: "userid", UserName: "johndoe"})
		albumRepo = NewAlbumRepository(ctx, GetDBXBuilder()).(*albumRepository)

		// Create album without any annotation (no star, no rating)
		albumWithoutAnnotation = model.Album{ID: "no-annotation-album", Name: "No Annotation", LibraryID: 1}
		Expect(albumRepo.Put(&albumWithoutAnnotation)).To(Succeed())
	})

	AfterEach(func() {
		_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": albumWithoutAnnotation.ID}))
	})

	Describe("starredFilter", func() {
		It("false includes items without annotations", func() {
			albums, err := albumRepo.GetAll(model.QueryOptions{
				Filters: annotationBoolFilter("starred")("starred", "false"),
			})
			Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, a := range albums {
				if a.ID == albumWithoutAnnotation.ID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Item without annotation should be included in starred=false filter")
		})

		It("true excludes items without annotations", func() {
			albums, err := albumRepo.GetAll(model.QueryOptions{
				Filters: annotationBoolFilter("starred")("starred", "true"),
			})
			Expect(err).ToNot(HaveOccurred())

			for _, a := range albums {
				Expect(a.ID).ToNot(Equal(albumWithoutAnnotation.ID))
			}
		})
	})

	Describe("hasRatingFilter", func() {
		It("false includes items without annotations", func() {
			albums, err := albumRepo.GetAll(model.QueryOptions{
				Filters: annotationBoolFilter("rating")("rating", "false"),
			})
			Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, a := range albums {
				if a.ID == albumWithoutAnnotation.ID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Item without annotation should be included in has_rating=false filter")
		})

		It("true excludes items without annotations", func() {
			albums, err := albumRepo.GetAll(model.QueryOptions{
				Filters: annotationBoolFilter("rating")("rating", "true"),
			})
			Expect(err).ToNot(HaveOccurred())

			for _, a := range albums {
				Expect(a.ID).ToNot(Equal(albumWithoutAnnotation.ID))
			}
		})
	})
})
