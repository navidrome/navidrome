package core

import (
	"context"
	"image"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork", func() {
	var artwork Artwork
	var ds model.DataStore
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "222", CoverArtId: "123", CoverArtPath: "tests/fixtures/test.mp3"},
			{ID: "333", CoverArtId: ""},
			{ID: "444", CoverArtId: "444", CoverArtPath: "tests/fixtures/cover.jpg"},
		})
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", AlbumID: "222", Path: "tests/fixtures/test.mp3", HasCoverArt: true},
			{ID: "456", AlbumID: "222", Path: "tests/fixtures/test.ogg", HasCoverArt: false},
		})
	})

	Context("Cache is configured", func() {
		BeforeEach(func() {
			conf.Server.DataFolder, _ = os.MkdirTemp("", "file_caches")
			conf.Server.ImageCacheSize = "100MB"
			cache := GetImageCache()
			Eventually(func() bool { return cache.Ready(context.TODO()) }).Should(BeTrue())
			artwork = NewArtwork(ds, cache)
		})
		AfterEach(func() {
			_ = os.RemoveAll(conf.Server.DataFolder)
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
				ds.Album(ctx).(*tests.MockAlbumRepo).SetError(true)
				_, err := artwork.Get(ctx, "al-222", 0)
				Expect(err).To(MatchError("Error!"))
			})

			It("returns err if gets error from media_file table", func() {
				ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetError(true)
				_, err := artwork.Get(ctx, "123", 0)
				Expect(err).To(MatchError("Error!"))
			})
		})
	})

	Describe("getAlbumCoverFromPath", func() {
		var testFolder, embeddedPath string
		var testPath []string
		BeforeEach(func() {
			testFolder, _ = os.MkdirTemp("", "album_persistence_tests")
			if err := os.MkdirAll(testFolder, 0777); err != nil {
				panic(err)
			}
			if _, err := os.Create(filepath.Join(testFolder, "Cover.jpeg")); err != nil {
				panic(err)
			}
			if _, err := os.Create(filepath.Join(testFolder, "FRONT.PNG")); err != nil {
				panic(err)
			}
			testPath = []string{filepath.Join(testFolder, "somefile.test")}
			embeddedPath = filepath.Join(testFolder, "somefile.mp3")

			DeferCleanup(configtest.SetupConfig())
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
		})
		AfterEach(func() {
			_ = os.RemoveAll(testFolder)
		})

		It("returns audio file for embedded cover", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getAlbumCoverFromPath(testPath, embeddedPath)).To(Equal(""))
		})

		It("returns external file when no embedded cover exists", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getAlbumCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "Cover.jpeg")))
		})

		It("returns embedded cover even if not first choice", func() {
			conf.Server.CoverArtPriority = "something.png, embedded, cover.*, front.*"
			Expect(getAlbumCoverFromPath(testPath, embeddedPath)).To(Equal(""))
		})

		It("returns first correct match case-insensitively", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.svg, front.png"
			Expect(getAlbumCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "FRONT.PNG")))
		})

		It("returns match for embedded pattern", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jp?g, front.png"
			Expect(getAlbumCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "Cover.jpeg")))
		})

		It("returns empty string if no match was found", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.apng"
			Expect(getAlbumCoverFromPath(testPath, "")).To(Equal(""))
		})
	})

})
