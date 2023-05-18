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
		BeforeEach(func() {
			mapper = newMediaFileMapper("/music", nil, nil)
		})
		Describe("mapTrackTitle", func() {
			It("returns the Title when it is available", func() {
				md := metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{"title": []string{"This is not a love song"}})
				Expect(mapper.mapTrackTitle(md)).To(Equal("This is not a love song"))
			})
			It("returns the filename if Title is not set", func() {
				md := metadata.NewTag("/music/artist/album01/Song.mp3", nil, metadata.ParsedTags{})
				Expect(mapper.mapTrackTitle(md)).To(Equal("artist/album01/Song"))
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

	Describe("mapGenres", func() {
		var mapper *mediaFileMapper
		var gr model.GenreRepository
		var ctx context.Context
		BeforeEach(func() {
			ctx = context.Background()
			ds := &tests.MockDataStore{}
			gr = ds.Genre(ctx)
			gr = newCachedGenreRepository(ctx, gr)
			mapper = newMediaFileMapper("/", gr, nil)
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

	Describe("mapPublishers", func() {
		var mapper *mediaFileMapper
		var pr model.PublisherRepository
		var ctx context.Context
		BeforeEach(func() {
			ctx = context.Background()
			ds := &tests.MockDataStore{}
			pr = ds.Publisher(ctx)
			pr = newCachedPublisherRepository(ctx, pr)
			mapper = newMediaFileMapper("/", nil, pr)
		})

		It("returns empty if no publishers are available", func() {
			p, ps := mapper.mapPublishers(nil)
			Expect(p).To(BeEmpty())
			Expect(ps).To(BeEmpty())
		})

		It("returns publishers", func() {
			p, ps := mapper.mapPublishers([]string{"Virgin", "Universal"})
			Expect(p).To(Equal("Virgin"))
			Expect(ps).To(HaveLen(2))
			Expect(ps[0].Name).To(Equal("Virgin"))
			Expect(ps[1].Name).To(Equal("Universal"))
		})

		It("parses multi-valued publishers", func() {
			p, ps := mapper.mapPublishers([]string{"Virgin;Dance", "Universal", "Virgin"})
			Expect(p).To(Equal("Virgin"))
			Expect(ps).To(HaveLen(3))
			Expect(ps[0].Name).To(Equal("Virgin"))
			Expect(ps[1].Name).To(Equal("Dance"))
			Expect(ps[2].Name).To(Equal("Universal"))
		})
		It("trims publishers names", func() {
			_, ps := mapper.mapPublishers([]string{"Virgin ;  Parlophone", " Universal "})
			Expect(ps).To(HaveLen(3))
			Expect(ps[0].Name).To(Equal("Virgin"))
			Expect(ps[1].Name).To(Equal("Parlophone"))
			Expect(ps[2].Name).To(Equal("Universal"))
		})
		It("does not break on spaces", func() {
			_, ps := mapper.mapPublishers([]string{"Capitol Records"})
			Expect(ps).To(HaveLen(1))
			Expect(ps[0].Name).To(Equal("Capitol Records"))
		})
	})
})
