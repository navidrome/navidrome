package scanner

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mapping", func() {
	Describe("mediaFileMapper", func() {
		var mapper *mediaFileMapper
		Describe("mapTrackTitle", func() {
			BeforeEach(func() {
				mapper = newMediaFileMapper("/music", nil)
			})
			It("returns the Title when it is available", func() {
				md := metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{"title": []string{"This is not a love song"}})
				Expect(mapper.mapTrackTitle(md)).To(Equal("This is not a love song"))
			})
			It("returns the filename if Title is not set", func() {
				md := metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{})
				Expect(mapper.mapTrackTitle(md)).To(Equal("artist/album01/Song"))
			})
		})

		Describe("mapGenres", func() {
			var gr model.GenreRepository
			var ctx context.Context

			BeforeEach(func() {
				ctx = context.Background()
				ds := &tests.MockDataStore{}
				gr = ds.Genre(ctx)
				gr = newCachedGenreRepository(ctx, gr)
				mapper = newMediaFileMapper("/", gr)
			})

			It("returns empty if no genres are available", func() {
				g, gs := mapper.mapGenres(nil)
				Expect(g).To(BeEmpty())
				Expect(gs).To(BeEmpty())
			})

			It("returns genres", func() {
				g, gs := mapper.mapGenres([]string{"Rock", "Electronic"})
				Expect(g).To(Equal("Rock"))
				Expect(gs).To(HaveLen(2))
				Expect(gs[0].Name).To(Equal("Rock"))
				Expect(gs[1].Name).To(Equal("Electronic"))
			})

			It("parses multi-valued genres", func() {
				g, gs := mapper.mapGenres([]string{"Rock;Dance", "Electronic", "Rock"})
				Expect(g).To(Equal("Rock"))
				Expect(gs).To(HaveLen(3))
				Expect(gs[0].Name).To(Equal("Rock"))
				Expect(gs[1].Name).To(Equal("Dance"))
				Expect(gs[2].Name).To(Equal("Electronic"))
			})
			It("trims genres names", func() {
				_, gs := mapper.mapGenres([]string{"Rock ;  Dance", " Electronic "})
				Expect(gs).To(HaveLen(3))
				Expect(gs[0].Name).To(Equal("Rock"))
				Expect(gs[1].Name).To(Equal("Dance"))
				Expect(gs[2].Name).To(Equal("Electronic"))
			})
			It("does not break on spaces", func() {
				_, gs := mapper.mapGenres([]string{"New Wave"})
				Expect(gs).To(HaveLen(1))
				Expect(gs[0].Name).To(Equal("New Wave"))
			})
		})

		Describe("mapDates", func() {
			var md metadata.Tags
			BeforeEach(func() {
				mapper = newMediaFileMapper("/", nil)
			})
			Context("when all date fields are provided", func() {
				BeforeEach(func() {
					md = metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{
						"date":         []string{"2023-03-01"},
						"originaldate": []string{"2022-05-10"},
						"releasedate":  []string{"2023-01-15"},
					})
				})

				It("should map all date fields correctly", func() {
					year, date, originalYear, originalDate, releaseYear, releaseDate := mapper.mapDates(md)
					Expect(year).To(Equal(2023))
					Expect(date).To(Equal("2023-03-01"))
					Expect(originalYear).To(Equal(2022))
					Expect(originalDate).To(Equal("2022-05-10"))
					Expect(releaseYear).To(Equal(2023))
					Expect(releaseDate).To(Equal("2023-01-15"))
				})
			})

			Context("when date field is missing", func() {
				BeforeEach(func() {
					md = metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{
						"originaldate": []string{"2022-05-10"},
						"releasedate":  []string{"2023-01-15"},
					})
				})

				It("should fallback to original date if date is missing", func() {
					year, date, _, _, _, _ := mapper.mapDates(md)
					Expect(year).To(Equal(2022))
					Expect(date).To(Equal("2022-05-10"))
				})
			})

			Context("when original and release dates are missing", func() {
				BeforeEach(func() {
					md = metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{
						"date": []string{"2023-03-01"},
					})
				})

				It("should only map the date field", func() {
					year, date, originalYear, originalDate, releaseYear, releaseDate := mapper.mapDates(md)
					Expect(year).To(Equal(2023))
					Expect(date).To(Equal("2023-03-01"))
					Expect(originalYear).To(BeZero())
					Expect(originalDate).To(BeEmpty())
					Expect(releaseYear).To(BeZero())
					Expect(releaseDate).To(BeEmpty())
				})
			})

			Context("when date fields are in an incorrect format", func() {
				BeforeEach(func() {
					md = metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{
						"date": []string{"invalid-date"},
					})
				})

				It("should handle invalid date formats gracefully", func() {
					year, date, _, _, _, _ := mapper.mapDates(md)
					Expect(year).To(BeZero())
					Expect(date).To(BeEmpty())
				})
			})

			Context("when all date fields are missing", func() {
				It("should return zero values for all date fields", func() {
					year, date, originalYear, originalDate, releaseYear, releaseDate := mapper.mapDates(md)
					Expect(year).To(BeZero())
					Expect(date).To(BeEmpty())
					Expect(originalYear).To(BeZero())
					Expect(originalDate).To(BeEmpty())
					Expect(releaseYear).To(BeZero())
					Expect(releaseDate).To(BeEmpty())
				})
			})
		})
	})

	Describe("sanitizeFieldForSorting", func() {
		BeforeEach(func() {
			conf.Server.IgnoredArticles = "The O"
		})
		It("sanitize accents", func() {
			Expect(sanitizeFieldForSorting("Céu")).To(Equal("Ceu"))
		})
		It("removes articles", func() {
			Expect(sanitizeFieldForSorting("The Beatles")).To(Equal("Beatles"))
		})
		It("removes accented articles", func() {
			Expect(sanitizeFieldForSorting("Õ Blésq Blom")).To(Equal("Blesq Blom"))
		})
	})
})
