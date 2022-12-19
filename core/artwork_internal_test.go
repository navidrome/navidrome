package core

import (
	"context"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Artwork", func() {
	var aw *artwork
	var ds model.DataStore
	ctx := log.NewContext(context.TODO())
	var alOnlyEmbed, alEmbedNotFound model.Album

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		alOnlyEmbed = model.Album{ID: "222", Name: "Only embed", EmbedArtPath: "tests/fixtures/test.mp3"}
		alEmbedNotFound = model.Album{ID: "333", Name: "Embed not found", EmbedArtPath: "tests/fixtures/NON_EXISTENT.mp3"}
		//	{ID: "666", Name: "All options", EmbedArtPath: "tests/fixtures/test.mp3",
		//		ImageFiles: "tests/fixtures/cover.jpg:tests/fixtures/front.png"},
		//})
		aw = NewArtwork(ds).(*artwork)
	})

	When("cover art is not found", func() {
		BeforeEach(func() {
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				alOnlyEmbed,
			})
		})
		It("returns placeholder if album is not in the DB", func() {
			_, path, err := aw.get(context.Background(), "al-999-0", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(consts.PlaceholderAlbumArt))
		})
	})
	When("album has only embed images", func() {
		BeforeEach(func() {
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				alOnlyEmbed,
				alEmbedNotFound,
			})
		})
		It("returns embed cover", func() {
			_, path, err := aw.get(context.Background(), alOnlyEmbed.CoverArtID().String(), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("tests/fixtures/test.mp3"))
		})
		It("returns placeholder if embed path is not available", func() {
			_, path, err := aw.get(context.Background(), alEmbedNotFound.CoverArtID().String(), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(consts.PlaceholderAlbumArt))
		})
	})
})
