package artwork

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ItemName", func() {
	var ds *tests.MockDataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		albumRepo := tests.CreateMockAlbumRepo()
		albumRepo.SetData(model.Albums{
			{ID: "al-1", Name: "Kid A"},
			{ID: "al-2", Name: "Sandinista!", Discs: model.Discs{2: "Side Three"}},
		})
		ds = &tests.MockDataStore{MockedAlbum: albumRepo}
		Expect(ds.Artist(ctx).(*tests.MockArtistRepo).Put(&model.Artist{ID: "ar-1", Name: "Radiohead"})).To(Succeed())
	})

	It("returns the album name", func() {
		Expect(ItemName(ctx, ds, model.KindAlbumArtwork, "al-1")).To(Equal("Kid A"))
	})

	It("returns the artist name", func() {
		Expect(ItemName(ctx, ds, model.KindArtistArtwork, "ar-1")).To(Equal("Radiohead"))
	})

	It("errors for an unknown album", func() {
		_, err := ItemName(ctx, ds, model.KindAlbumArtwork, "nope")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("errors for an unsupported kind", func() {
		// model.Kind is a struct with unexported fields, so the zero value is the only
		// unsupported Kind constructible from outside package model.
		_, err := ItemName(ctx, ds, model.Kind{}, "al-1")
		Expect(err).To(HaveOccurred())
	})

	Context("disc artwork", func() {
		It("names the album, the disc and its subtitle", func() {
			Expect(ItemName(ctx, ds, model.KindDiscArtwork, "al-2:2")).
				To(Equal("Sandinista! (disc 2): Side Three"))
		})

		It("omits the subtitle when the disc has none", func() {
			Expect(ItemName(ctx, ds, model.KindDiscArtwork, "al-2:1")).
				To(Equal("Sandinista! (disc 1)"))
		})

		It("rejects an id that is not <albumID>:<disc>", func() {
			_, err := ItemName(ctx, ds, model.KindDiscArtwork, "al-2")
			Expect(err).To(HaveOccurred())
		})
	})
})
