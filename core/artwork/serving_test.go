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

var _ = Describe("Service", func() {
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
		svc        Service
		repoRoot   string
		coverBytes []byte
	)

	primaryKey := func(kind, id string) string { return kind + "|" + id + "|" + model.ImageTypePrimary }

	// seedFoundStore installs a store-backed found state (bytes in the content-addressed
	// store, no backing file) and returns the hash.
	seedFoundStore := func(kind, id string, imgBytes []byte) string {
		hash, err := HashImage(bytes.NewReader(imgBytes))
		Expect(err).ToNot(HaveOccurred())
		Expect(store.Write(hash, "image/jpeg", bytes.NewReader(imgBytes))).To(Succeed())
		Expect(artRepo.PutImage(&model.Artwork{Hash: hash, Mime: "image/jpeg"})).To(Succeed())
		Expect(artRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: kind, ItemID: id, Hash: hash, Source: "external"})).To(Succeed())
		return hash
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
				r, _, err := arg.(artworkReader).Reader(ctx)
				return r, err
			})
		Eventually(func() bool { return imgCache.Available(ctx) }).Should(BeTrue())
		svc = NewService(ds, imgCache, store, ffm)
	})

	Describe("found state", func() {
		It("serves a store-backed found image sized (cache miss resizes, second call is a cache hit)", func() {
			seedFoundStore("al", "al1", coverBytes)

			img, err := svc.Get(ctx, model.MustParseArtworkID("al-al1"), 100, false)
			Expect(err).ToNot(HaveOccurred())
			// A resized response versions its validator with the encode settings, distinct from
			// the pixel hash, so a CoverArtQuality/WebP change invalidates client caches.
			Expect(img.ETag).To(Equal(representationTag(img.Hash, 100, false)))
			Expect(img.ETag).ToNot(Equal(img.Hash))
			resized := readAll(img)
			cfg, _, err := image.DecodeConfig(bytes.NewReader(resized))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Width).To(Equal(100))

			// Delete the store file: a warm resize-cache entry must keep serving without
			// ever touching the original (the stale-serve self-heal).
			hash, _ := HashImage(bytes.NewReader(coverBytes))
			Expect(store.Remove(hash, "image/jpeg", time.Now().Add(time.Hour))).To(Succeed())
			Eventually(func(g Gomega) {
				img2, err := svc.Get(ctx, model.MustParseArtworkID("al-al1"), 100, false)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(readAll(img2)).To(Equal(resized))
			}).Should(Succeed())
		})

		It("streams a file-backed found image at full size", func() {
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, coverBytes, 0600)).To(Succeed())
			mtime := fileMtime(imgPath)
			Expect(artRepo.PutImage(&model.Artwork{Hash: "aaaaaaaaaaaaaaaa", Mime: "image/jpeg"})).To(Succeed())
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
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al3", Hash: "bbbbbbbbbbbbbbbb",
				Source: "folder", SourcePath: imgPath, RefMtime: fileMtime(imgPath) + 999,
			})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al3"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al3")].Priority).To(Equal(model.ArtworkPriorityScan))
			ia, err := artRepo.GetItemArtwork("al", "al3", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(Equal("bbbbbbbbbbbbbbbb"))
		})

		It("enforces the mtime rule on the sized (loader) path too", func() {
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, coverBytes, 0600)).To(Succeed())
			Expect(artRepo.PutImage(&model.Artwork{Hash: "cccccccccccccccc", Mime: "image/jpeg"})).To(Succeed())
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al3b", Hash: "cccccccccccccccc",
				Source: "folder", SourcePath: imgPath, RefMtime: fileMtime(imgPath) + 999,
			})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al3b"), 100, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al3b")].Priority).To(Equal(model.ArtworkPriorityScan))
		})

		It("returns ErrUnavailable for an absent state without enqueuing", func() {
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "al4"})).To(Succeed())

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al4"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data).To(BeEmpty())
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
			_, err = artRepo.GetItemArtwork("al", "al5", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns ErrUnavailable and enqueues a Bump when nothing local resolves", func() {
			folderRepo.result = nil
			albumRepo.SetData(model.Albums{{ID: "al6", Name: "Album"}})

			_, err := svc.Get(ctx, model.MustParseArtworkID("al-al6"), 0, false)
			Expect(err).To(MatchError(ErrUnavailable))
			Expect(queueRepo.Data[primaryKey("al", "al6")].Priority).To(Equal(model.ArtworkPriorityBump))
			_, err = artRepo.GetItemArtwork("al", "al6", model.ImageTypePrimary)
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
			_, err = artRepo.GetItemArtwork("mf", "mf4", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("delegates a multi-disc track to its disc art, not straight to the album", func() {
			// Not embedded-eligible: the fallback must mirror CoverArtID (disc first, then album).
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "aldd", Name: "Album", FolderIDs: []string{"f1"}, Discs: model.Discs{1: "One", 2: "Two"}}})
			seedFoundStore("al", "aldd", []byte("album-art-distinct")) // album's own found art differs
			mfRepo.SetData(model.MediaFiles{{ID: "mf5", AlbumID: "aldd", DiscNumber: 1, HasCoverArt: false}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf5"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			// The disc-folder image wins over the album's found art, proving it routed via serveDisc.
			Expect(readAll(img)).To(Equal(coverBytes))
		})

		It("delegates a single-disc track straight to the album, skipping disc resolution", func() {
			folderRepo.result = []model.Folder{{Path: "tests/fixtures/artist/an-album", ImageFiles: []string{"cover.jpg"}}}
			albumRepo.SetData(model.Albums{{ID: "alsd", Name: "Album", FolderIDs: []string{"f1"}, Discs: model.Discs{1: ""}}})
			seedFoundStore("al", "alsd", []byte("album-art-distinct"))
			mfRepo.SetData(model.MediaFiles{{ID: "mf6", AlbumID: "alsd", DiscNumber: 1, HasCoverArt: false}})

			img, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf6"), 0, false)
			Expect(err).ToNot(HaveOccurred())
			// Single-disc album: album art wins; the folder disc image must not shadow it.
			Expect(readAll(img)).To(Equal([]byte("album-art-distinct")))
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
	})
})

func fileMtime(path string) int64 {
	GinkgoHelper()
	info, err := os.Stat(path)
	Expect(err).ToNot(HaveOccurred())
	return info.ModTime().Unix()
}
