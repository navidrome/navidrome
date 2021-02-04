package core

import (
	"context"
	"image"
	"io/ioutil"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork", func() {
	var artwork Artwork
	var ds model.DataStore
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepository{}}
		ds.Album(ctx).(*tests.MockAlbum).SetData(model.Albums{
			{ID: "222", CoverArtId: "123", CoverArtPath: "tests/fixtures/test.mp3"},
			{ID: "333", CoverArtId: ""},
			{ID: "444", CoverArtId: "444", CoverArtPath: "tests/fixtures/cover.jpg"},
		})
		ds.MediaFile(ctx).(*tests.MockMediaFile).SetData(model.MediaFiles{
			{ID: "123", AlbumID: "222", Path: "tests/fixtures/test.mp3", HasCoverArt: true},
			{ID: "456", AlbumID: "222", Path: "tests/fixtures/test.ogg", HasCoverArt: false},
		})
	})

	Context("Cache is configured", func() {
		BeforeEach(func() {
			conf.Server.DataFolder, _ = ioutil.TempDir("", "file_caches")
			conf.Server.ImageCacheSize = "100MB"
			cache := GetImageCache()
			Eventually(func() bool { return cache.Ready(context.TODO()) }).Should(BeTrue())
			artwork = NewArtwork(ds, cache)
		})

		It("retrieves the external artwork art for an album", func() {
			r, err := artwork.Get(ctx, "al-444", 0)
			Expect(err).To(BeNil())

			_, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(r.Close()).To(BeNil())
		})

		It("retrieves the embedded artwork art for an album", func() {
			r, err := artwork.Get(ctx, "al-222", 0)
			Expect(err).To(BeNil())

			_, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(r.Close()).To(BeNil())
		})

		It("returns the default artwork if album does not have artwork", func() {
			r, err := artwork.Get(ctx, "al-333", 0)
			Expect(err).To(BeNil())

			_, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("png"))
			Expect(r.Close()).To(BeNil())
		})

		It("returns the default artwork if album is not found", func() {
			r, err := artwork.Get(ctx, "al-0101", 0)
			Expect(err).To(BeNil())

			_, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("png"))
			Expect(r.Close()).To(BeNil())
		})

		It("retrieves the original artwork art from a media_file", func() {
			r, err := artwork.Get(ctx, "123", 0)
			Expect(err).To(BeNil())

			img, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(img.Bounds().Size().X).To(Equal(600))
			Expect(img.Bounds().Size().Y).To(Equal(600))
			Expect(r.Close()).To(BeNil())
		})

		It("retrieves the album artwork art if media_file does not have one", func() {
			r, err := artwork.Get(ctx, "456", 0)
			Expect(err).To(BeNil())

			_, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(r.Close()).To(BeNil())
		})

		It("retrieves the album artwork by album id", func() {
			r, err := artwork.Get(ctx, "222", 0)
			Expect(err).To(BeNil())

			_, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(r.Close()).To(BeNil())
		})

		It("resized artwork art as requested", func() {
			r, err := artwork.Get(ctx, "123", 200)
			Expect(err).To(BeNil())

			img, format, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(img.Bounds().Size().X).To(Equal(200))
			Expect(img.Bounds().Size().Y).To(Equal(200))
			Expect(r.Close()).To(BeNil())
		})

		Context("Errors", func() {
			It("returns err if gets error from album table", func() {
				ds.Album(ctx).(*tests.MockAlbum).SetError(true)
				_, err := artwork.Get(ctx, "al-222", 0)
				Expect(err).To(MatchError("Error!"))
			})

			It("returns err if gets error from media_file table", func() {
				ds.MediaFile(ctx).(*tests.MockMediaFile).SetError(true)
				_, err := artwork.Get(ctx, "123", 0)
				Expect(err).To(MatchError("Error!"))
			})
		})
	})
})
