package persistence

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
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

	Describe("annotationBoolFilter", func() {
		DescribeTable("creates correct SQL expressions",
			func(field, value string, expectedSQL string, expectedArgs []interface{}) {
				sqlizer := annotationBoolFilter(field)(field, value)
				sql, args, err := sqlizer.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSQL))
				Expect(args).To(Equal(expectedArgs))
			},
			Entry("starred=true", "starred", "true", "COALESCE(starred, 0) > 0", []interface{}(nil)),
			Entry("starred=false", "starred", "false", "COALESCE(starred, 0) = 0", []interface{}(nil)),
			Entry("starred=True (case insensitive)", "starred", "True", "COALESCE(starred, 0) > 0", []interface{}(nil)),
			Entry("rating=true", "rating", "true", "COALESCE(rating, 0) > 0", []interface{}(nil)),
		)

		It("returns nil if value is not a string", func() {
			sqlizer := annotationBoolFilter("starred")("starred", 123)
			Expect(sqlizer).To(BeNil())
		})
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

		It("true includes items with rating > 0", func() {
			// Create album with rating 1
			ratedAlbum := model.Album{ID: "rated-album", Name: "Rated Album", LibraryID: 1}
			Expect(albumRepo.Put(&ratedAlbum)).To(Succeed())
			Expect(albumRepo.SetRating(1, ratedAlbum.ID)).To(Succeed())
			defer func() {
				_, _ = albumRepo.executeSQL(squirrel.Delete("annotation").Where(squirrel.Eq{"item_id": ratedAlbum.ID}))
				_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": ratedAlbum.ID}))
			}()

			albums, err := albumRepo.GetAll(model.QueryOptions{
				Filters: annotationBoolFilter("rating")("rating", "true"),
			})
			Expect(err).ToNot(HaveOccurred())

			var found bool
			for _, a := range albums {
				if a.ID == ratedAlbum.ID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Album with rating 5 should be included in has_rating=true filter")
		})
	})

	It("ignores invalid filter values (not strings)", func() {
		res, err := albumRepo.ReadAll(rest.QueryOptions{
			Filters: map[string]any{"starred": 123},
		})
		Expect(err).ToNot(HaveOccurred())
		albums := res.(model.Albums)

		var found bool
		for _, a := range albums {
			if a.ID == albumWithoutAnnotation.ID {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "Item without annotation should be included when filter is ignored")
	})
})
