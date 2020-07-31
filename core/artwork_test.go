package core

import (
	"bytes"
	"context"
	"image"
	"io/ioutil"
	"os"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork", func() {
	var artwork Artwork
	var ds model.DataStore
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &persistence.MockDataStore{MockedTranscoding: &mockTranscodingRepository{}}
		ds.Album(ctx).(*persistence.MockAlbum).SetData(`[{"id": "222", "coverArtId": "123", "coverArtPath":"tests/fixtures/test.mp3"}, {"id": "333", "coverArtId": ""}, {"id": "444", "coverArtId": "444", "coverArtPath": "tests/fixtures/cover.jpg"}]`)
		ds.MediaFile(ctx).(*persistence.MockMediaFile).SetData(`[{"id": "123", "albumId": "222", "path": "tests/fixtures/test.mp3", "hasCoverArt": true, "updatedAt":"2020-04-02T21:29:31.6377Z"},{"id": "456", "albumId": "222", "path": "tests/fixtures/test.ogg", "hasCoverArt": false, "updatedAt":"2020-04-02T21:29:31.6377Z"}]`)
	})

	Context("Cache is configured", func() {
		BeforeEach(func() {
			conf.Server.DataFolder, _ = ioutil.TempDir("", "file_caches")
			conf.Server.ImageCacheSize = "100MB"
			cache := NewImageCache()
			Eventually(func() bool { return cache.Ready() }).Should(BeTrue())
			artwork = NewArtwork(ds, cache)
		})

		AfterEach(func() {
			os.RemoveAll(conf.Server.DataFolder)
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
				ds.Album(ctx).(*persistence.MockAlbum).SetError(true)
				buf := new(bytes.Buffer)

				Expect(artwork.Get(ctx, "al-222", 0, buf)).To(MatchError("Error!"))
			})

			It("returns err if gets error from media_file table", func() {
				ds.MediaFile(ctx).(*persistence.MockMediaFile).SetError(true)
				buf := new(bytes.Buffer)

				Expect(artwork.Get(ctx, "123", 0, buf)).To(MatchError("Error!"))
			})
		})
	})
})
