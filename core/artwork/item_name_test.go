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
		ds = &tests.MockDataStore{}
		Expect(ds.Album(ctx).(*tests.MockAlbumRepo).Put(&model.Album{ID: "al-1", Name: "Kid A"})).To(Succeed())
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
})
