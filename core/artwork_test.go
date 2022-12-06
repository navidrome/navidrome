package core

import (
	"context"
	"errors"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing/fstest"

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
	var fsys fs.FS
	ctx := log.NewContext(context.Background())

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.CoverArtPriority = "embedded, cover.*"
		fsys = fstest.MapFS{
			"tests/fixtures":           &fstest.MapFile{Mode: fs.ModeDir},
			"tests/fixtures/cover.jpg": &fstest.MapFile{},
		}
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
	})

	When("cache is configured", func() {
		var cache ArtworkCache
		BeforeEach(func() {
			conf.Server.DataFolder, _ = os.MkdirTemp("", "file_caches")
			conf.Server.ImageCacheSize = "100MB"
			onceImageCache = sync.Once{} // Reinitialize cache!
			cache = GetImageCache()
			Eventually(func() bool { return cache.Ready(context.TODO()) }).Should(BeTrue())
		})
		AfterEach(func() {
			_ = os.RemoveAll(conf.Server.DataFolder)
		})

		Context("id does not exist in DB", func() {
			BeforeEach(func() {
				artwork = NewArtworkWithFS(ds, cache, fsys)
			})
			It("returns the default artwork if id is empty", func() {
				r, err := artwork.Get(ctx, "", 0)
				Expect(err).To(BeNil())

				_, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("png"))
				Expect(r.Close()).To(BeNil())
			})
			It("returns the default artwork if id is empty", func() {
				r, err := artwork.Get(ctx, "al-999-ff", 0)
				Expect(err).To(BeNil())

				_, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("png"))
				Expect(r.Close()).To(BeNil())
			})
			It("returns error if id is invalid", func() {
				_, err := artwork.Get(ctx, "11-xxx", 0)
				Expect(err).To(MatchError("invalid artwork id"))
			})
		})

		Context("album has embedded cover art", func() {
			BeforeEach(func() {
				ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
					{ID: "123", AlbumID: "222", Path: "tests/fixtures/test.mp3", HasCoverArt: true},
					{ID: "456", AlbumID: "222", Path: "tests/fixtures/test.ogg", HasCoverArt: false},
				})
				artwork = NewArtworkWithFS(ds, cache, fsys)
			})
			It("retrieves the embedded artwork art for an album", func() {
				r, err := artwork.Get(ctx, "al-222-ff", 0)
				Expect(err).To(BeNil())

				img, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("jpeg"))
				Expect(img.Bounds().Size().X).To(Equal(600))
				Expect(img.Bounds().Size().Y).To(Equal(600))
				Expect(r.Close()).To(BeNil())
			})
			It("resizes artwork art as requested", func() {
				r, err := artwork.Get(ctx, "al-222-ff", 200)
				Expect(err).To(BeNil())

				img, _, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(img.Bounds().Size().X).To(Equal(200))
				Expect(img.Bounds().Size().Y).To(Equal(200))
				Expect(r.Close()).To(BeNil())
			})
			It("retrieves the album artwork art if media_file does not have one", func() {
				r, err := artwork.Get(ctx, "mf-123-0", 0)
				Expect(err).To(BeNil())

				_, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("jpeg"))
				Expect(r.Close()).To(BeNil())
			})
		})

		Context("album has cover.jpg file", func() {
			BeforeEach(func() {
				ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
					{ID: "123", AlbumID: "222", Path: "tests/fixtures/test.mp3", HasCoverArt: false},
					{ID: "456", AlbumID: "222", Path: "tests/fixtures/test.ogg", HasCoverArt: false},
				})
				artwork = NewArtworkWithFS(ds, cache, fsys)
			})

			It("retrieves the external artwork art for an album", func() {
				r, err := artwork.Get(ctx, "al-222-ff", 0)
				Expect(err).To(BeNil())

				_, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("jpeg"))
				Expect(r.Close()).To(BeNil())
			})
			It("returns the default artwork if album does not have artwork", func() {
				conf.Server.CoverArtPriority = "embedded, front.jpg"
				r, err := artwork.Get(ctx, "al-222-ff", 0)
				Expect(err).To(BeNil())

				_, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("png"))
				Expect(r.Close()).To(BeNil())
			})
			It("retrieves the album artwork art if media_file does not have one", func() {
				r, err := artwork.Get(ctx, "mf-456-0", 0)
				Expect(err).To(BeNil())

				_, format, err := image.Decode(r)
				Expect(err).To(BeNil())
				Expect(format).To(Equal("jpeg"))
				Expect(r.Close()).To(BeNil())
			})
		})

		Context("Errors", func() {
			BeforeEach(func() {
				var err = errors.New("db error")
				ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetError(err)
				artwork = NewArtworkWithFS(ds, cache, fsys)
			})
			It("returns err if gets error from db with album artwork id", func() {
				_, err := artwork.Get(ctx, "al-222-00", 0)
				Expect(err).To(MatchError("db error"))
			})

			It("returns err if gets error from db with mediafile artwork id", func() {
				_, err := artwork.Get(ctx, "mf-123-00", 0)
				Expect(err).To(MatchError("db error"))
			})
		})
	})

	Describe("getAlbumCoverFromPath", func() {
		var embeddedPath string
		var testPath []string
		var fsys fs.FS
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			testPath = []string{"testDir", "testDir2"}
			embeddedPath = filepath.Join("testDir", "somefile.mp3")
			fsys = fstest.MapFS{
				"testDir":            &fstest.MapFile{Mode: fs.ModeDir},
				"testDir/cover.jpeg": &fstest.MapFile{},
				"testDir2":           &fstest.MapFile{Mode: fs.ModeDir},
				"testDir2/front.png": &fstest.MapFile{},
			}
		})

		It("returns audio file for embedded cover", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getAlbumCoverFromPath(fsys, testPath, embeddedPath)).To(Equal(embeddedPath))
		})

		It("returns external file when no embedded cover exists", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getAlbumCoverFromPath(fsys, testPath, "")).To(Equal(filepath.Join("testDir", "cover.jpeg")))
		})

		It("returns embedded cover even if not first choice", func() {
			conf.Server.CoverArtPriority = "something.png, embedded, cover.*, front.*"
			Expect(getAlbumCoverFromPath(fsys, testPath, embeddedPath)).To(Equal(embeddedPath))
		})

		It("returns match in any album folders", func() {
			conf.Server.CoverArtPriority = "embedded, front.p?g, cover.jp?g"
			Expect(getAlbumCoverFromPath(fsys, testPath, "")).To(Equal(filepath.Join("testDir2", "front.png")))
		})

		It("returns empty string if no match was found", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.gif"
			Expect(getAlbumCoverFromPath(fsys, testPath, "")).To(Equal(""))
		})
	})
})
