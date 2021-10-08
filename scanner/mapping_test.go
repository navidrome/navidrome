package scanner

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("mapping", func() {
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

	Describe("mapGenres", func() {
		var mapper *mediaFileMapper
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
})

var _ = Describe("isSynced", func() {
	It("returns false if lyrics contain no timestamps", func() {
		Expect(isSynced(`Just in case my car goes off the highway`)).To(Equal(false))
		Expect(isSynced(`[02.50] Just in case my car goes off the highway`)).To(Equal(false))
	})
	It("returns false if lyrics is an empty string", func() {
		Expect(isSynced(``)).To(Equal(false))
	})
	It("returns true if lyrics contain timestamps", func() {
		Expect(isSynced(`NF Real Music
		[00:00] ksdjjs
		[00:00.85] JUST LIKE YOU
		[00:00.85] Just in case my car goes off the highway`)).To(Equal(true))
		Expect(isSynced(`[04:02:50.85] Never gonna give you up`)).To(Equal(true))
		Expect(isSynced(`[02:50.85] Never gonna give you up`)).To(Equal(true))
		Expect(isSynced(`[02:50] Never gonna give you up`)).To(Equal(true))
	})

})
