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
			func(field, value string, expectedSQL string, expectedArgs []any) {
				sqlizer := annotationBoolFilter(field)(field, value)
				sql, args, err := sqlizer.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSQL))
				Expect(args).To(Equal(expectedArgs))
			},
			Entry("starred=true", "starred", "true", "COALESCE(starred, 0) > 0", []any(nil)),
			Entry("starred=false", "starred", "false", "COALESCE(starred, 0) = 0", []any(nil)),
			Entry("starred=True (case insensitive)", "starred", "True", "COALESCE(starred, 0) > 0", []any(nil)),
			Entry("rating=true", "rating", "true", "COALESCE(rating, 0) > 0", []any(nil)),
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

	Describe("annotationColumns", func() {
		It("derives the annotation join columns from model.Annotations, excluding average_rating", func() {
			cols := annotationColumns()
			Expect(cols).To(HaveKey("starred"))
			Expect(cols).To(HaveKey("starred_at"))
			Expect(cols).To(HaveKey("rating"))
			Expect(cols).To(HaveKey("rated_at"))
			Expect(cols).To(HaveKey("play_count"))
			Expect(cols).To(HaveKey("play_date"))
			Expect(cols).To(HaveLen(6), "expected exactly the 6 annotation-join columns")
			Expect(cols).ToNot(HaveKey("average_rating"), "average_rating lives on the base table, not the annotation join")
		})
	})

	Describe("filtersNeedAnnotation", func() {
		It("is true when the query references an annotation column", func() {
			q := squirrel.Select("count(1)").From("media_file").Where(squirrel.Eq{"starred": true})
			Expect(filtersNeedAnnotation(q)).To(BeTrue())
		})

		It("is true for a raw expression referencing an annotation column", func() {
			q := squirrel.Select("count(1)").From("media_file").Where(squirrel.Expr("rating > 0"))
			Expect(filtersNeedAnnotation(q)).To(BeTrue())
		})

		It("is false for a query that references no annotation column", func() {
			q := squirrel.Select("count(1)").From("media_file").Where(squirrel.Eq{"missing": false})
			Expect(filtersNeedAnnotation(q)).To(BeFalse())
		})

		It("is false for a filter on average_rating (base-table column, not the annotation rating)", func() {
			// Regression: average_rating must not match the annotation column "rating".
			q := squirrel.Select("count(1)").From("media_file").Where(squirrel.Gt{"average_rating": 3})
			Expect(filtersNeedAnnotation(q)).To(BeFalse())
		})

		It("is true when both average_rating and a real annotation column are referenced", func() {
			q := squirrel.Select("count(1)").From("media_file").
				Where(squirrel.Gt{"average_rating": 3}).
				Where(squirrel.Expr("COALESCE(rating, 0) > 0"))
			Expect(filtersNeedAnnotation(q)).To(BeTrue())
		})

		It("is true for uppercase/mixed-case annotation columns (SQLite is case-insensitive)", func() {
			q := squirrel.Select("count(1)").From("media_file").Where(squirrel.Expr("RATING > 0"))
			Expect(filtersNeedAnnotation(q)).To(BeTrue())
		})

		It("is false for uppercase average_rating (still excluded case-insensitively)", func() {
			q := squirrel.Select("count(1)").From("media_file").Where(squirrel.Expr("AVERAGE_RATING > 3"))
			Expect(filtersNeedAnnotation(q)).To(BeFalse())
		})
	})

	Describe("CountAll annotation-join gating", func() {
		It("counts all items unfiltered (join dropped)", func() {
			total, err := albumRepo.CountAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(BeNumerically(">=", int64(1)))

			filtered, err := albumRepo.CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.id": albumWithoutAnnotation.ID},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(filtered).To(Equal(int64(1)))
		})

		It("counts starred items correctly (named annotation filter keeps the join)", func() {
			starredAlbum := model.Album{ID: "counted-starred-album", Name: "Counted Starred", LibraryID: 1}
			Expect(albumRepo.Put(&starredAlbum)).To(Succeed())
			Expect(albumRepo.SetStar(true, starredAlbum.ID)).To(Succeed())
			defer func() {
				_, _ = albumRepo.executeSQL(squirrel.Delete("annotation").Where(squirrel.Eq{"item_id": starredAlbum.ID}))
				_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": starredAlbum.ID}))
			}()

			// Exactly two albums are starred for this user: the one created above and
			// albumRadioactivity (id 103) from the seed data.
			count, err := albumRepo.CountAll(model.QueryOptions{
				Filters: annotationBoolFilter("starred")("starred", "true"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(2)))
		})

		It("counts via a raw annotation filter without a 'no such column' error", func() {
			count, err := albumRepo.CountAll(model.QueryOptions{
				Filters: squirrel.Expr("COALESCE(rating, 0) > 0"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeNumerically(">=", int64(0)))
		})
	})
})
