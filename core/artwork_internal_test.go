package core

import (
	"context"
	"errors"
	"image"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork", func() {
	var aw *artwork
	var ds model.DataStore
	var ffmpeg *tests.MockFFmpeg
	ctx := log.NewContext(context.TODO())
	var alOnlyEmbed, alEmbedNotFound, alOnlyExternal, alExternalNotFound, alAllOptions model.Album
	var mfWithEmbed, mfWithoutEmbed, mfCorruptedCover model.MediaFile

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		alOnlyEmbed = model.Album{ID: "222", Name: "Only embed", EmbedArtPath: "tests/fixtures/test.mp3"}
		alEmbedNotFound = model.Album{ID: "333", Name: "Embed not found", EmbedArtPath: "tests/fixtures/NON_EXISTENT.mp3"}
		alOnlyExternal = model.Album{ID: "444", Name: "Only external", ImageFiles: "tests/fixtures/front.png"}
		alExternalNotFound = model.Album{ID: "555", Name: "External not found", ImageFiles: "tests/fixtures/NON_EXISTENT.png"}
		alAllOptions = model.Album{ID: "666", Name: "All options", EmbedArtPath: "tests/fixtures/test.mp3",
			ImageFiles: "tests/fixtures/cover.jpg:tests/fixtures/front.png",
		}
		mfWithEmbed = model.MediaFile{ID: "22", Path: "tests/fixtures/test.mp3", HasCoverArt: true, AlbumID: "222"}
		mfWithoutEmbed = model.MediaFile{ID: "44", Path: "tests/fixtures/test.ogg", AlbumID: "444"}
		mfCorruptedCover = model.MediaFile{ID: "45", Path: "tests/fixtures/test.ogg", HasCoverArt: true, AlbumID: "444"}

		DeferCleanup(configtest.SetupConfig())
		conf.Server.ImageCacheSize = "0" // Disable cache

		cache := GetImageCache()
		ffmpeg = tests.NewMockFFmpeg("")
		aw = NewArtwork(ds, cache, ffmpeg).(*artwork)
	})

	Context("Empty ID", func() {
		It("returns placeholder if album is not in the DB", func() {
			_, path, err := aw.get(context.Background(), model.ArtworkID{}, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal(consts.PlaceholderAlbumArt))
		})
	})

	Context("Albums", func() {
		Context("ID not found", func() {
			It("returns placeholder if album is not in the DB", func() {
				_, path, err := aw.get(context.Background(), model.MustParseArtworkID("al-NOT_FOUND-0"), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(consts.PlaceholderAlbumArt))
			})
		})
		Context("Embed images", func() {
			BeforeEach(func() {
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alOnlyEmbed,
					alEmbedNotFound,
				})
			})
			It("returns embed cover", func() {
				_, path, err := aw.get(context.Background(), alOnlyEmbed.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/test.mp3"))
			})
			It("returns placeholder if embed path is not available", func() {
				ffmpeg.Error = errors.New("not available")
				_, path, err := aw.get(context.Background(), alEmbedNotFound.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(consts.PlaceholderAlbumArt))
			})
		})
		Context("External images", func() {
			BeforeEach(func() {
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alOnlyExternal,
					alAllOptions,
				})
			})
			It("returns external cover", func() {
				_, path, err := aw.get(context.Background(), alOnlyExternal.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/front.png"))
			})
			It("returns the first image if more than one is available", func() {
				_, path, err := aw.get(context.Background(), alAllOptions.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/cover.jpg"))
			})
			It("returns placeholder if external file is not available", func() {
				_, path, err := aw.get(context.Background(), alExternalNotFound.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(consts.PlaceholderAlbumArt))
			})
		})
	})
	Context("MediaFiles", func() {
		Context("ID not found", func() {
			It("returns placeholder if album is not in the DB", func() {
				_, path, err := aw.get(context.Background(), model.MustParseArtworkID("mf-NOT_FOUND-0"), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal(consts.PlaceholderAlbumArt))
			})
		})
		Context("Embed images", func() {
			BeforeEach(func() {
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alOnlyEmbed,
					alOnlyExternal,
				})
				ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
					mfWithEmbed,
					mfWithoutEmbed,
					mfCorruptedCover,
				})
			})
			It("returns embed cover", func() {
				_, path, err := aw.get(context.Background(), mfWithEmbed.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/test.mp3"))
			})
			It("returns embed cover if successfully extracted by ffmpeg", func() {
				_, path, err := aw.get(context.Background(), mfCorruptedCover.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/test.ogg"))
			})
			It("returns album cover if cannot read embed artwork", func() {
				ffmpeg.Error = errors.New("not available")
				_, path, err := aw.get(context.Background(), mfCorruptedCover.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/front.png"))
			})
			It("returns album cover if media file has no cover art", func() {
				_, path, err := aw.get(context.Background(), mfWithoutEmbed.CoverArtID(), 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/front.png"))
			})
		})
	})
	Context("Resize", func() {
		BeforeEach(func() {
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				alOnlyExternal,
			})
		})
		It("returns external cover resized", func() {
			r, path, err := aw.get(context.Background(), alOnlyExternal.CoverArtID(), 300)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("tests/fixtures/front.png@300"))
			img, _, err := image.Decode(r)
			Expect(err).To(BeNil())
			Expect(img.Bounds().Size().X).To(Equal(300))
			Expect(img.Bounds().Size().Y).To(Equal(300))
		})
	})
})
