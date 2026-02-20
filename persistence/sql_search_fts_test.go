package persistence

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildFTS5Query", func() {
	It("returns empty string for empty input", func() {
		Expect(buildFTS5Query("")).To(BeEmpty())
	})

	It("returns empty string for whitespace-only input", func() {
		Expect(buildFTS5Query("   ")).To(BeEmpty())
	})

	It("passes through a single word", func() {
		Expect(buildFTS5Query("beatles")).To(Equal("beatles"))
	})

	It("joins multiple words with implicit AND", func() {
		Expect(buildFTS5Query("abbey road")).To(Equal("abbey road"))
	})

	It("preserves quoted phrases", func() {
		Expect(buildFTS5Query(`"the beatles"`)).To(Equal(`"the beatles"`))
	})

	It("preserves prefix wildcard", func() {
		Expect(buildFTS5Query("beat*")).To(Equal("beat*"))
	})

	It("strips FTS5 operators to prevent injection", func() {
		Expect(buildFTS5Query("AND OR NOT NEAR")).To(Equal("and or not near"))
	})

	It("strips special FTS5 syntax characters", func() {
		Expect(buildFTS5Query("test^col:val")).To(Equal("test col val"))
	})

	It("handles mixed phrases and words", func() {
		Expect(buildFTS5Query(`"the beatles" abbey`)).To(Equal(`"the beatles" abbey`))
	})

	It("handles prefix with multiple words", func() {
		Expect(buildFTS5Query("beat* abbey")).To(Equal("beat* abbey"))
	})

	It("collapses multiple spaces", func() {
		Expect(buildFTS5Query("abbey   road")).To(Equal("abbey road"))
	})

	It("strips leading * from tokens", func() {
		Expect(buildFTS5Query("*livia")).To(Equal("livia"))
	})

	It("strips leading * but preserves trailing *", func() {
		Expect(buildFTS5Query("*livia oliv*")).To(Equal("livia oliv*"))
	})

	It("strips standalone *", func() {
		Expect(buildFTS5Query("*")).To(BeEmpty())
	})
})

var _ = Describe("FTS5 Integration Search", func() {
	var (
		mr  model.MediaFileRepository
		alr model.AlbumRepository
		arr model.ArtistRepository
	)

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		conn := GetDBXBuilder()
		mr = NewMediaFileRepository(ctx, conn)
		alr = NewAlbumRepository(ctx, conn)
		arr = NewArtistRepository(ctx, conn)
	})

	Describe("MediaFile search", func() {
		It("finds media files by title", func() {
			results, err := mr.Search("Radioactivity", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Radioactivity"))
			Expect(results[0].ID).To(Equal(songRadioactivity.ID))
		})

		It("finds media files by artist name", func() {
			results, err := mr.Search("Beatles", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			for _, r := range results {
				Expect(r.Artist).To(Equal("The Beatles"))
			}
		})
	})

	Describe("Album search", func() {
		It("finds albums by name", func() {
			results, err := alr.Search("Sgt Peppers", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Sgt Peppers"))
			Expect(results[0].ID).To(Equal(albumSgtPeppers.ID))
		})

		It("finds albums with multi-word search", func() {
			results, err := alr.Search("Abbey Road", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Abbey Road"))
			Expect(results[0].ID).To(Equal(albumAbbeyRoad.ID))
		})
	})

	Describe("Artist search", func() {
		It("finds artists by name", func() {
			results, err := arr.Search("Kraftwerk", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Kraftwerk"))
			Expect(results[0].ID).To(Equal(artistKraftwerk.ID))
		})
	})

	Describe("Legacy backend fallback", func() {
		It("returns results using legacy LIKE-based search when configured", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.SearchBackend = "legacy"

			results, err := mr.Search("Radioactivity", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Radioactivity"))
		})
	})
})
