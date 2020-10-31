package core

import (
	"bytes"
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
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "al-444", 0, buf)).To(BeNil())

			_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
		})

		It("retrieves the embedded artwork art for an album", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "al-222", 0, buf)).To(BeNil())

			_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
		})

		It("returns the default artwork if album does not have artwork", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "al-333", 0, buf)).To(BeNil())

			_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("png"))
		})

		It("returns the default artwork if album is not found", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "al-0101", 0, buf)).To(BeNil())

			_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("png"))
		})

		It("retrieves the original artwork art from a media_file", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "123", 0, buf)).To(BeNil())

			img, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(img.Bounds().Size().X).To(Equal(600))
			Expect(img.Bounds().Size().Y).To(Equal(600))
		})

		It("retrieves the album artwork art if media_file does not have one", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "456", 0, buf)).To(BeNil())

			_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
		})

		It("retrieves the album artwork by album id", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "222", 0, buf)).To(BeNil())

			_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
		})

		It("resized artwork art as requested", func() {
			buf := new(bytes.Buffer)

			Expect(artwork.Get(ctx, "123", 200, buf)).To(BeNil())

			img, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
			Expect(err).To(BeNil())
			Expect(format).To(Equal("jpeg"))
			Expect(img.Bounds().Size().X).To(Equal(200))
			Expect(img.Bounds().Size().Y).To(Equal(200))
		})

		Context("Errors", func() {
			It("returns err if gets error from album table", func() {
				ds.Album(ctx).(*tests.MockAlbum).SetError(true)
				buf := new(bytes.Buffer)

				Expect(artwork.Get(ctx, "al-222", 0, buf)).To(MatchError("Error!"))
			})

			It("returns err if gets error from media_file table", func() {
				ds.MediaFile(ctx).(*tests.MockMediaFile).SetError(true)
				buf := new(bytes.Buffer)

				Expect(artwork.Get(ctx, "123", 0, buf)).To(MatchError("Error!"))
			})
		})
	})
})
