package artwork

import (
	"bytes"
	"context"
	"image"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		artRepo    *tests.MockArtworkRepo
		queueRepo  *tests.MockArtworkQueueRepo
		albumRepo  *tests.MockAlbumRepo
		mfRepo     *tests.MockMediaFileRepo
		folderRepo *fakeFolderRepo
		libRepo    *tests.MockLibraryRepo
		ffm        *tests.MockFFmpeg
		store      *ImageStore
		imgCache   cache.FileCache
		svc        Artwork
		repoRoot   string
		coverBytes []byte
		seedEntity func(kind, id string)
	)

	primaryKey := func(kind, id string) string { return kind + "|" + id + "|" + model.ImageTypePrimary }

	seedFoundStore := func(kind, id string, imgBytes []byte) string {
		hash, err := hashImage(bytes.NewReader(imgBytes))
		Expect(err).ToNot(HaveOccurred())
		Expect(store.Write(hash, "image/jpeg", bytes.NewReader(imgBytes))).To(Succeed())
		Expect(artRepo.PutImage(&model.Artwork{Hash: hash, Mime: "image/jpeg"})).To(Succeed())
		Expect(artRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: kind, ItemID: id, Hash: hash, Source: "external"})).To(Succeed())
		seedEntity(kind, id)
		return hash
	}

	// Without its owning entity, a state row is not served at all.
	seedEntity = func(kind, id string) {
		GinkgoHelper()
		switch kind {
		case "al":
			Expect(albumRepo.Put(&model.Album{ID: id, Name: "Album"})).To(Succeed())
		case "mf":
			Expect(mfRepo.Put(&model.MediaFile{ID: id})).To(Succeed())
		}
	}

	readAll := func(img *Image) []byte {
		GinkgoHelper()
		defer img.Close()
		data, err := io.ReadAll(img)
		Expect(err).ToNot(HaveOccurred())
		return data
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		var err error
		repoRoot, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		coverBytes, err = os.ReadFile(filepath.Join(repoRoot, "tests/fixtures/artist/an-album/cover.jpg"))
		Expect(err).ToNot(HaveOccurred())

		conf.Server.EnableWebPEncoding = false
		conf.Server.CoverArtQuality = 75
		conf.Server.CoverArtPriority = "cover.*"
		conf.Server.DiscArtPriority = "cover.*"
		conf.Server.CacheFolder = conf.NewDir(GinkgoT().TempDir())

		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		albumRepo = tests.CreateMockAlbumRepo()
		mfRepo = tests.CreateMockMediaFileRepo()
		folderRepo = &fakeFolderRepo{}
		libRepo = &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		ds = &tests.MockDataStore{
			MockedArtwork:      artRepo,
			MockedArtworkQueue: queueRepo,
			MockedAlbum:        albumRepo,
			MockedMediaFile:    mfRepo,
			MockedFolder:       folderRepo,
			MockedLibrary:      libRepo,
		}
		ffm = tests.NewMockFFmpeg("")
		store = NewImageStore(GinkgoT().TempDir())
		imgCache = cache.NewFileCache("ServingTest", "100MB", "images", 0,
			func(ctx context.Context, arg cache.Item) (io.Reader, error) {
				return arg.(artworkReader).Reader(ctx)
			})
		Eventually(func() bool { return imgCache.Available(ctx) }, 10*time.Second).Should(BeTrue())
		svc = NewArtwork(ds, imgCache, store, ffm)
	})

	Describe("found state", func() {
		It("serves a store-backed found image sized (cache miss resizes, second call is a cache hit)", func() {
			seedFoundStore("al", "al1", coverBytes)

			img, err := svc.Get(ctx, model.MustParseArtworkID("al-al1"), 100, false)
			Expect(err).ToNot(HaveOccurred())
			// A resized response versions its ETag with the encode settings, not the pixel hash.
			Expect(img.ETag).To(Equal(representationTag(img.Hash, 100, false)))
			Expect(img.ETag).ToNot(Equal(img.Hash))
			resized := readAll(img)
			cfg, _, err := image.DecodeConfig(bytes.NewReader(resized))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Width).To(Equal(100))

			// Deleting the store file proves the warm entry serves without touching the original.
			hash, _ := hashImage(bytes.NewReader(coverBytes))
			Expect(os.Remove(store.path(hash, "image/jpeg"))).To(Succeed())
			Eventually(func(g Gomega) {
				img2, err := svc.Get(ctx, model.MustParseArtworkID("al-al1"), 100, false)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(readAll(img2)).To(Equal(resized))
			}).Should(Succeed())
		})

		It("treats a negative size as a full-size request, not a giant resize", func() {
			seedFoundStore("al", "alneg", coverBytes)

			img, err := svc.Get(ctx, model.MustParseArtworkID("al-alneg"), -2000000000, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes), "original bytes, no resize (would OOM)")
		})

		It("streams a file-backed found image at full size", func() {
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, coverBytes, 0600)).To(Succeed())
			mtime := fileMtime(imgPath)
			Expect(artRepo.PutImage(&model.Artwork{Hash: "aaaaaaaaaaaaaaaa", Mime: "image/jpeg"})).To(Succeed())
			seedEntity("al", "al2")
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al2", Hash: "aaaaaaaaaaaaaaaa",
				Source: "folder", SourcePath: imgPath, RefMtime: mtime,
			})).To(Succeed())

			img, err := svc.Get(ctx, model.MustParseArtworkID("al-al2"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
		})

		It("treats a full-size mtime mismatch as dangling: unavailable, re-enqueued at Scan, state untouched", func() {
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, coverBytes, 0600)).To(Succeed())
			Expect(artRepo.PutImage(&model.Artwork{Hash: "bbbbbbbbbbbbbbbb", Mime: "image/jpeg"})).To(Succeed())
			seedEntity("al", "al3")
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al3", Hash: "bbbbbbbbbbbbbbbb",
				Source: "folder", SourcePath: imgPath, RefMtime: fileMtime(imgPath) + 999,
			})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al3"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al3")].Priority).To(Equal(model.ArtworkPriorityScan))
			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al3", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(Equal("bbbbbbbbbbbbbbbb"))
		})

		It("enforces the mtime rule on the sized (loader) path too", func() {
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, coverBytes, 0600)).To(Succeed())
			Expect(artRepo.PutImage(&model.Artwork{Hash: "cccccccccccccccc", Mime: "image/jpeg"})).To(Succeed())
			seedEntity("al", "al3b")
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al3b", Hash: "cccccccccccccccc",
				Source: "folder", SourcePath: imgPath, RefMtime: fileMtime(imgPath) + 999,
			})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al3b"), 100, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al3b")].Priority).To(Equal(model.ArtworkPriorityScan))
		})

		// State rows outlive a deleted entity until the next prune.
		It("refuses to serve a found row whose entity is gone", func() {
			hash := seedFoundStore("al", "alzz", coverBytes)
			Expect(hash).ToNot(BeEmpty())
			albumRepo.SetData(model.Albums{}) // the album is deleted; its artwork row survives

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-alzz"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
		})

		It("does not re-enqueue a recently-attempted absent state", func() {
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al4", AttemptedAt: time.Now(),
			})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al4"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data).To(BeEmpty())
		})

		It("promotes a stale absent state at Bump priority on view", func() {
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al4b", AttemptedAt: time.Now().Add(-2 * requestRecheckAge),
			})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al4b"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al4b")].Priority).To(Equal(model.ArtworkPriorityBump))
			ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al4b", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(BeEmpty())
		})
	})

	Describe("provisional read-through", func() {
		It("serves local folder art, enqueues a Bump, and writes no state row", func() {
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "al5", Name: "Album", FolderIDs: []string{"f1"}}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("al-al5"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))

			Expect(queueRepo.Data[primaryKey("al", "al5")].Priority).To(Equal(model.ArtworkPriorityBump))
			_, err = artRepo.GetItemArtwork(model.KindAlbumArtwork, "al5", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns ErrUnavailable and enqueues a Bump when nothing local resolves", func() {
			folderRepo.result = nil
			albumRepo.SetData(model.Albums{{ID: "al6", Name: "Album"}})

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al6"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al6")].Priority).To(Equal(model.ArtworkPriorityBump))
			_, err = artRepo.GetItemArtwork(model.KindAlbumArtwork, "al6", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("media file", func() {
		It("serves a track's own found art", func() {
			seedFoundStore("mf", "mf1", coverBytes)

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf1"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
		})

		It("ignores a resolved mf row and delegates to the album when per-track art is disabled", func() {
			conf.Server.EnableMediaFileCoverArt = false
			seedFoundStore("mf", "mf7", []byte("stale embedded track art"))
			seedFoundStore("al", "albz", coverBytes)
			mfRepo.SetData(model.MediaFiles{{ID: "mf7", AlbumID: "albz"}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf7"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes), "album art, not the persisted embedded art")
		})

		It("delegates to the album when the track's state is absent", func() {
			seedFoundStore("al", "albm", coverBytes)
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "mf", ItemID: "mf2"})).To(Succeed())
			mfRepo.SetData(model.MediaFiles{{ID: "mf2", AlbumID: "albm"}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf2"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
			_, mfEnq := queueRepo.Data[primaryKey("mf", "mf2")]
			Expect(mfEnq).To(BeFalse())
		})

		It("delegates to the album (no enqueue) when the track is not embedded-eligible", func() {
			conf.Server.EnableMediaFileCoverArt = true
			seedFoundStore("al", "albn", coverBytes)
			mfRepo.SetData(model.MediaFiles{{ID: "mf3", AlbumID: "albn", HasCoverArt: false}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf3"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
			_, mfEnq := queueRepo.Data[primaryKey("mf", "mf3")]
			Expect(mfEnq).To(BeFalse())
		})

		It("extracts embedded art provisionally and enqueues the track when eligible", func() {
			conf.Server.EnableMediaFileCoverArt = true
			mfRepo.SetData(model.MediaFiles{{
				ID: "mf4", AlbumID: "albo", HasCoverArt: true,
				Path: "tests/fixtures/artist/an-album/test.mp3", LibraryID: 0,
			}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf4"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(readAll(img))).To(BeNumerically(">", 0))
			Expect(queueRepo.Data[primaryKey("mf", "mf4")].Priority).To(Equal(model.ArtworkPriorityBump))
			_, err = artRepo.GetItemArtwork(model.KindMediaFileArtwork, "mf4", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("delegates a multi-disc track to its disc art, not straight to the album", func() {
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "aldd", Name: "Album", FolderIDs: []string{"f1"}, Discs: model.Discs{1: "One", 2: "Two"}}})
			seedFoundStore("al", "aldd", []byte("album-art-distinct")) // album's own found art differs
			mfRepo.SetData(model.MediaFiles{{ID: "mf5", AlbumID: "aldd", DiscNumber: 1, HasCoverArt: false}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf5"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			// The disc-folder image, not the album's own art: proof it routed through serveDisc.
			Expect(readAll(img)).To(Equal(coverBytes))
		})

		It("falls back to the album when an eligible track's embedded art will not extract", func() {
			conf.Server.EnableMediaFileCoverArt = true
			// HasCoverArt is set, but the file is not audio, so nothing extracts.
			mfRepo.SetData(model.MediaFiles{{
				ID: "mfbad", AlbumID: "albad", LibraryID: 0, HasCoverArt: true,
				Path: "tests/fixtures/artist/an-album/front.png",
			}})
			albumRepo.SetData(model.Albums{{ID: "albad", Name: "Album"}})
			seedFoundStore("al", "albad", coverBytes)

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mfbad"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes), "a placeholder here would be worse than the album cover")
		})

		It("routes a single-disc track through disc resolution too", func() {
			// DiscArtPriority applies to single-disc albums too, over the album's own found art.
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "alsd", Name: "Album", FolderIDs: []string{"f1"}, Discs: model.Discs{1: ""}}})
			seedFoundStore("al", "alsd", []byte("album-art-distinct"))
			mfRepo.SetData(model.MediaFiles{{ID: "mf6", AlbumID: "alsd", DiscNumber: 1, HasCoverArt: false}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf6"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
		})
	})

	Describe("disc", func() {
		It("serves a local disc-folder image", func() {
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "aldc", Name: "Album", FolderIDs: []string{"f1"}}})

			img, err := svc.Get(ctx, model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID("aldc", 1), nil), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
		})

		// The resize cache keys on id + album mtime, not on the bytes, so dropping the source
		// between the two requests is what shows a warm hit never touches the filesystem.
		It("serves a sized disc image from cache without re-reading the source", func() {
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "aldc3", Name: "Album", FolderIDs: []string{"f1"}}})
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID("aldc3", 1), nil)

			first, err := svc.Get(ctx, discID, 64, false)
			Expect(err).ToNot(HaveOccurred())
			warmed := readAll(first)
			Expect(warmed).ToNot(BeEmpty())

			folderRepo.result = nil

			second, err := svc.Get(ctx, discID, 64, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(second)).To(Equal(warmed), "a warm sized request must not touch the source")
		})

		// A disc image can change without the album row changing, so the key folds in ImagesUpdatedAt.
		It("invalidates the cached image when the folder's images change", func() {
			folderRepo.result = []model.Folder{{
				Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"},
				ImagesUpdatedAt: time.Now().Add(-time.Hour),
			}}
			albumRepo.SetData(model.Albums{{ID: "aldc4", Name: "Album", FolderIDs: []string{"f1"}}})
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID("aldc4", 1), nil)

			first, err := svc.Get(ctx, discID, 64, false)
			Expect(err).ToNot(HaveOccurred())
			firstKey := first.ETag
			readAll(first)

			folderRepo.result[0].ImagesUpdatedAt = time.Now()
			second, err := svc.Get(ctx, discID, 64, false)
			Expect(err).ToNot(HaveOccurred())
			readAll(second)
			Expect(second.ETag).ToNot(Equal(firstKey), "a replaced image must not keep the old cache entry")
		})

		// Disc art has no content hash, so without an explicit validator every ETag would be empty.
		It("gives a full-size disc image a validator that tracks the source", func() {
			folderRepo.result = []model.Folder{{
				Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"},
				ImagesUpdatedAt: time.Now().Add(-time.Hour),
			}}
			albumRepo.SetData(model.Albums{{ID: "aldc5", Name: "Album", FolderIDs: []string{"f1"}}})
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID("aldc5", 1), nil)

			first, err := svc.Get(ctx, discID, 0, false)
			Expect(err).ToNot(HaveOccurred())
			readAll(first)
			Expect(first.ETag).ToNot(BeEmpty())

			folderRepo.result[0].ImagesUpdatedAt = time.Now()
			second, err := svc.Get(ctx, discID, 0, false)
			Expect(err).ToNot(HaveOccurred())
			readAll(second)
			Expect(second.ETag).ToNot(Equal(first.ETag), "a replaced image must not revalidate as unchanged")
		})

		It("falls back to album art when no disc image matches", func() {
			folderRepo.result = nil
			albumRepo.SetData(model.Albums{{ID: "aldc2", Name: "Album"}})
			seedFoundStore("al", "aldc2", coverBytes)

			img, err := svc.Get(ctx, model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID("aldc2", 1), nil), 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(readAll(img)).To(Equal(coverBytes))
		})
	})

	Describe("GetOrPlaceholder", func() {
		It("accepts a raw entity id and serves its cover art", func() {
			albumRepo.SetData(model.Albums{{ID: "rawal", Name: "Album"}})
			seedFoundStore("al", "rawal", coverBytes)

			img, err := svc.GetOrPlaceholder(ctx, "rawal", 0, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(img.Placeholder).To(BeFalse())
			Expect(readAll(img)).To(Equal(coverBytes))
		})

		It("falls back to the album placeholder ignoring size and square", func() {
			img, err := svc.GetOrPlaceholder(ctx, "", 300, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(img.Placeholder).To(BeTrue())
			Expect(img.Hash).To(BeEmpty())
			Expect(img.LastUpdated).To(BeZero())

			ph, err := resources.FS().Open(consts.PlaceholderAlbumArt)
			Expect(err).ToNot(HaveOccurred())
			phBytes, _ := io.ReadAll(ph)
			Expect(readAll(img)).To(Equal(phBytes))
		})

		It("falls back to the artist placeholder for an absent artist", func() {
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "arph"})).To(Succeed())

			img, err := svc.GetOrPlaceholder(ctx, "ar-arph", 300, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(img.Placeholder).To(BeTrue())

			ph, err := resources.FS().Open(consts.PlaceholderArtistArt)
			Expect(err).ToNot(HaveOccurred())
			phBytes, _ := io.ReadAll(ph)
			Expect(readAll(img)).To(Equal(phBytes))
		})

		// "No art" and "no such entity" are different answers: clients 404 only on the latter.
		It("reports not-found rather than a placeholder for an id with no entity", func() {
			_, err := svc.GetOrPlaceholder(ctx, "al-nosuchalbum", 0, false)
			Expect(err).To(MatchError(model.ErrNotFound))

			_, err = svc.GetOrPlaceholder(ctx, "nosuchrawid", 0, false)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})
})

func fileMtime(path string) int64 {
	GinkgoHelper()
	info, err := os.Stat(path)
	Expect(err).ToNot(HaveOccurred())
	return info.ModTime().UnixNano()
}

var _ = Describe("EntityExists", func() {
	var ctx context.Context
	var ds *tests.MockDataStore

	BeforeEach(func() {
		ctx = context.Background()
		albumRepo := tests.CreateMockAlbumRepo()
		albumRepo.SetData(model.Albums{{ID: "al1"}})
		artistRepo := tests.CreateMockArtistRepo()
		artistRepo.SetData(model.Artists{{ID: "ar1"}})
		radioRepo := tests.CreateMockedRadioRepo()
		Expect(radioRepo.Put(&model.Radio{ID: "ra1", Name: "R"})).To(Succeed())
		ds = &tests.MockDataStore{MockedAlbum: albumRepo, MockedArtist: artistRepo, MockedRadio: radioRepo}
	})

	DescribeTable("reports whether the owning entity is still there",
		func(id string, expected bool) {
			Expect(entityExists(ctx, ds, model.MustParseArtworkID(id))).To(Equal(expected))
		},
		Entry("existing album", "al-al1", true),
		Entry("deleted album", "al-gone", false),
		Entry("existing artist", "ar-ar1", true),
		Entry("deleted artist", "ar-gone", false),
		Entry("existing radio", "ra-ra1", true),
		Entry("deleted radio", "ra-gone", false),
		// Disc art has no entity of its own; it stands or falls with its album.
		Entry("disc of an existing album", "dc-al1:1", true),
		Entry("disc of a deleted album", "dc-gone:1", false),
		Entry("malformed disc id", "dc-nodiscnum", false),
	)
})
