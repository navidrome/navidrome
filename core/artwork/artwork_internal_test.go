package artwork

import (
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
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
	var aw *artwork
	var ds model.DataStore
	var ffmpeg *tests.MockFFmpeg
	var folderRepo *fakeFolderRepo
	ctx := log.NewContext(context.TODO())
	var alOnlyEmbed, alEmbedNotFound, alOnlyExternal, alExternalNotFound, alMultipleCovers model.Album
	var arMultipleCovers model.Artist
	var mfWithEmbed, mfAnotherWithEmbed, mfWithoutEmbed, mfCorruptedCover model.MediaFile

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.ImageCacheSize = "0" // Disable cache
		conf.Server.CoverArtPriority = "folder.*, cover.*, embedded , front.*"

		folderRepo = &fakeFolderRepo{}
		ds = &tests.MockDataStore{
			MockedTranscoding: &tests.MockTranscodingRepo{},
			MockedFolder:      folderRepo,
		}
		alOnlyEmbed = model.Album{ID: "222", Name: "Only embed", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}}
		alEmbedNotFound = model.Album{ID: "333", Name: "Embed not found", EmbedArtPath: "tests/fixtures/NON_EXISTENT.mp3", FolderIDs: []string{"f1"}}
		alOnlyExternal = model.Album{ID: "444", Name: "Only external", FolderIDs: []string{"f1"}}
		alExternalNotFound = model.Album{ID: "555", Name: "External not found", FolderIDs: []string{"f2"}}
		arMultipleCovers = model.Artist{ID: "777", Name: "All options"}
		alMultipleCovers = model.Album{
			ID:            "666",
			Name:          "All options",
			EmbedArtPath:  "tests/fixtures/artist/an-album/test.mp3",
			FolderIDs:     []string{"f1"},
			AlbumArtistID: "777",
		}
		mfWithEmbed = model.MediaFile{ID: "22", Path: "tests/fixtures/test.mp3", HasCoverArt: true, AlbumID: "222"}
		mfAnotherWithEmbed = model.MediaFile{ID: "23", Path: "tests/fixtures/artist/an-album/test.mp3", HasCoverArt: true, AlbumID: "666"}
		mfWithoutEmbed = model.MediaFile{ID: "44", Path: "tests/fixtures/test.ogg", AlbumID: "444"}
		mfCorruptedCover = model.MediaFile{ID: "45", Path: "tests/fixtures/test.ogg", HasCoverArt: true, AlbumID: "444"}

		cache := GetImageCache()
		ffmpeg = tests.NewMockFFmpeg("content from ffmpeg")
		aw = NewArtwork(ds, cache, ffmpeg, nil).(*artwork)
	})

	Describe("albumArtworkReader", func() {
		Context("ID not found", func() {
			It("returns ErrNotFound if album is not in the DB", func() {
				_, err := newAlbumArtworkReader(ctx, aw, model.MustParseArtworkID("al-NOT-FOUND"), nil)
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})
		Context("Embed images", func() {
			BeforeEach(func() {
				folderRepo.result = nil
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alOnlyEmbed,
					alEmbedNotFound,
				})
			})
			It("returns embed cover", func() {
				aw, err := newAlbumArtworkReader(ctx, aw, alOnlyEmbed.CoverArtID(), nil)
				Expect(err).ToNot(HaveOccurred())
				_, path, err := aw.Reader(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/artist/an-album/test.mp3"))
			})
			It("returns ErrUnavailable if embed path is not available", func() {
				ffmpeg.Error = errors.New("not available")
				aw, err := newAlbumArtworkReader(ctx, aw, alEmbedNotFound.CoverArtID(), nil)
				Expect(err).ToNot(HaveOccurred())
				_, _, err = aw.Reader(ctx)
				Expect(err).To(MatchError(ErrUnavailable))
			})
		})
		Context("External images", func() {
			BeforeEach(func() {
				folderRepo.result = []model.Folder{}
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alOnlyExternal,
					alExternalNotFound,
				})
			})
			It("returns external cover", func() {
				folderRepo.result = []model.Folder{{
					Path:       "tests/fixtures/artist/an-album",
					ImageFiles: []string{"front.png"},
				}}
				aw, err := newAlbumArtworkReader(ctx, aw, alOnlyExternal.CoverArtID(), nil)
				Expect(err).ToNot(HaveOccurred())
				_, path, err := aw.Reader(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/artist/an-album/front.png"))
			})
			It("returns ErrUnavailable if external file is not available", func() {
				folderRepo.result = []model.Folder{}
				aw, err := newAlbumArtworkReader(ctx, aw, alExternalNotFound.CoverArtID(), nil)
				Expect(err).ToNot(HaveOccurred())
				_, _, err = aw.Reader(ctx)
				Expect(err).To(MatchError(ErrUnavailable))
			})
		})
		Context("Multiple covers", func() {
			BeforeEach(func() {
				folderRepo.result = []model.Folder{{
					Path:       "tests/fixtures/artist/an-album",
					ImageFiles: []string{"cover.jpg", "front.png", "artist.png"},
				}}
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alMultipleCovers,
				})
			})
			DescribeTable("CoverArtPriority",
				func(priority string, expected string) {
					conf.Server.CoverArtPriority = priority
					aw, err := newAlbumArtworkReader(ctx, aw, alMultipleCovers.CoverArtID(), nil)
					Expect(err).ToNot(HaveOccurred())
					_, path, err := aw.Reader(ctx)
					Expect(err).ToNot(HaveOccurred())
					Expect(path).To(Equal(expected))
				},
				Entry(nil, " folder.* , cover.*,embedded,front.*", "tests/fixtures/artist/an-album/cover.jpg"),
				Entry(nil, "front.* , cover.*, embedded ,folder.*", "tests/fixtures/artist/an-album/front.png"),
				Entry(nil, " embedded , front.* , cover.*,folder.*", "tests/fixtures/artist/an-album/test.mp3"),
			)
		})
	})
	Describe("artistArtworkReader", func() {
		Context("Multiple covers", func() {
			BeforeEach(func() {
				folderRepo.result = []model.Folder{{
					Path:       "tests/fixtures/artist/an-album",
					ImageFiles: []string{"artist.png"},
				}}
				ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
					arMultipleCovers,
				})
				ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
					alMultipleCovers,
				})
				ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
					mfAnotherWithEmbed,
				})
			})
			DescribeTable("ArtistArtPriority",
				func(priority string, expected string) {
					conf.Server.ArtistArtPriority = priority
					aw, err := newArtistArtworkReader(ctx, aw, arMultipleCovers.CoverArtID(), nil)
					Expect(err).ToNot(HaveOccurred())
					_, path, err := aw.Reader(ctx)
					Expect(err).ToNot(HaveOccurred())
					Expect(path).To(Equal(expected))
				},
				Entry(nil, " folder.* , artist.*,album/artist.*", "tests/fixtures/artist/artist.jpg"),
				Entry(nil, "album/artist.*, folder.*,artist.*", "tests/fixtures/artist/an-album/artist.png"),
			)
		})
	})
	Describe("mediafileArtworkReader", func() {
		Context("ID not found", func() {
			It("returns ErrNotFound if mediafile is not in the DB", func() {
				_, err := newMediafileArtworkReader(ctx, aw, model.MustParseArtworkID("mf-NOT-FOUND"))
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})
		Context("Embed images", func() {
			BeforeEach(func() {
				folderRepo.result = []model.Folder{{
					Path:       "tests/fixtures/artist/an-album",
					ImageFiles: []string{"front.png"},
				}}
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
				aw, err := newMediafileArtworkReader(ctx, aw, mfWithEmbed.CoverArtID())
				Expect(err).ToNot(HaveOccurred())
				_, path, err := aw.Reader(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("tests/fixtures/test.mp3"))
			})
			It("returns embed cover if successfully extracted by ffmpeg", func() {
				aw, err := newMediafileArtworkReader(ctx, aw, mfCorruptedCover.CoverArtID())
				Expect(err).ToNot(HaveOccurred())
				r, path, err := aw.Reader(ctx)
				Expect(err).ToNot(HaveOccurred())
				data, _ := io.ReadAll(r)
				Expect(data).ToNot(BeEmpty())
				Expect(path).To(Equal("tests/fixtures/test.ogg"))
			})
			It("returns album cover if cannot read embed artwork", func() {
				// Force fromTag to fail
				mfCorruptedCover.Path = "tests/fixtures/DOES_NOT_EXIST.ogg"
				Expect(ds.MediaFile(ctx).(*tests.MockMediaFileRepo).Put(&mfCorruptedCover)).To(Succeed())
				// Simulate ffmpeg error
				ffmpeg.Error = errors.New("not available")

				aw, err := newMediafileArtworkReader(ctx, aw, mfCorruptedCover.CoverArtID())
				Expect(err).ToNot(HaveOccurred())
				_, path, err := aw.Reader(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("al-444_0"))
			})
			It("returns album cover if media file has no cover art", func() {
				aw, err := newMediafileArtworkReader(ctx, aw, model.MustParseArtworkID("mf-"+mfWithoutEmbed.ID))
				Expect(err).ToNot(HaveOccurred())
				_, path, err := aw.Reader(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(path).To(Equal("al-444_0"))
			})
		})
	})
	Describe("resizedArtworkReader", func() {
		BeforeEach(func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg", "front.png"},
			}}
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				alMultipleCovers,
			})
		})
		When("Square is false", func() {
			It("returns a PNG if original image is a PNG", func() {
				conf.Server.CoverArtPriority = "front.png"
				r, _, err := aw.Get(context.Background(), alMultipleCovers.CoverArtID(), 15, false)
				Expect(err).ToNot(HaveOccurred())

				img, format, err := image.Decode(r)
				Expect(err).ToNot(HaveOccurred())
				Expect(format).To(Equal("png"))
				Expect(img.Bounds().Size().X).To(Equal(15))
				Expect(img.Bounds().Size().Y).To(Equal(15))
			})
			It("returns a JPEG if original image is not a PNG", func() {
				conf.Server.CoverArtPriority = "cover.jpg"
				r, _, err := aw.Get(context.Background(), alMultipleCovers.CoverArtID(), 200, false)
				Expect(err).ToNot(HaveOccurred())

				img, format, err := image.Decode(r)
				Expect(format).To(Equal("jpeg"))
				Expect(err).ToNot(HaveOccurred())
				Expect(img.Bounds().Size().X).To(Equal(200))
				Expect(img.Bounds().Size().Y).To(Equal(200))
			})
		})
		When("When square is true", func() {
			var alCover model.Album

			DescribeTable("resize",
				func(format string, landscape bool, size int) {
					coverFileName := "cover." + format
					dirName := createImage(format, landscape, size)
					alCover = model.Album{
						ID:        "444",
						Name:      "Only external",
						FolderIDs: []string{"tmp"},
					}
					folderRepo.result = []model.Folder{{Path: dirName, ImageFiles: []string{coverFileName}}}
					ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
						alCover,
					})

					conf.Server.CoverArtPriority = coverFileName
					r, _, err := aw.Get(context.Background(), alCover.CoverArtID(), size, true)
					Expect(err).ToNot(HaveOccurred())

					img, format, err := image.Decode(r)
					Expect(err).ToNot(HaveOccurred())
					Expect(format).To(Equal("png"))
					Expect(img.Bounds().Size().X).To(Equal(size))
					Expect(img.Bounds().Size().Y).To(Equal(size))
				},
				Entry("portrait png image", "png", false, 200),
				Entry("landscape png image", "png", true, 200),
				Entry("portrait jpg image", "jpg", false, 200),
				Entry("landscape jpg image", "jpg", true, 200),
			)
		})
	})
})

func createImage(format string, landscape bool, size int) string {
	var img image.Image

	if landscape {
		img = image.NewRGBA(image.Rect(0, 0, size, size/2))
	} else {
		img = image.NewRGBA(image.Rect(0, 0, size/2, size))
	}

	tmpDir := GinkgoT().TempDir()
	f, _ := os.Create(filepath.Join(tmpDir, "cover."+format))
	defer f.Close()
	switch format {
	case "png":
		_ = png.Encode(f, img)
	case "jpg":
		_ = jpeg.Encode(f, img, &jpeg.Options{Quality: 75})
	}

	return tmpDir
}
