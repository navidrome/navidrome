package artwork

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("processItem", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		folderRepo *fakeFolderRepo
		libRepo    *tests.MockLibraryRepo
		ffm        *tests.MockFFmpeg
		prov       *fakeExternalProvider
		store      *ImageStore
		artRepo    *tests.MockArtworkRepo
		repoRoot   string
		deps       *workerDeps
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		var err error
		repoRoot, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		folderRepo = &fakeFolderRepo{}
		libRepo = &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		ffm = tests.NewMockFFmpeg("")
		prov = &fakeExternalProvider{}
		artRepo = tests.CreateMockArtworkRepo()
		ds = &tests.MockDataStore{
			MockedFolder:  folderRepo,
			MockedLibrary: libRepo,
			MockedArtwork: artRepo,
		}
		ds.MockedAlbum = tests.CreateMockAlbumRepo()
		store = NewImageStore(GinkgoT().TempDir())
		deps = &workerDeps{ds: ds, store: store, prov: prov, ffmpeg: ffm}

		conf.Server.CoverArtPriority = "cover.jpg, embedded"
	})

	It("found-folder: persists state from a folder image, writes no store file, keeps sourcePath/refMtime", func() {
		folderRepo.result = []model.Folder{{
			Path:       "tests/fixtures/artist/an-album",
			ImageFiles: []string{"cover.jpg"},
		}}
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al1", Name: "Album", FolderIDs: []string{"f1"}},
		})

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al1"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork("al", "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Hash).ToNot(BeEmpty())
		Expect(ia.Source).To(Equal("folder"))

		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(filepath.ToSlash(art.SourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/cover.jpg"))
		Expect(art.RefMtime).To(BeNumerically(">", 0))

		_, err = store.Open(ia.Hash, art.Mime)
		Expect(os.IsNotExist(err)).To(BeTrue(), "folder-backed art must not be duplicated into the store")
	})

	It("found-embedded: writes a store file and computes a non-empty blurhash from a real fixture", func() {
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al2", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
		})
		folderRepo.result = nil

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork("al", "al2", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("embedded"))

		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.BlurHash).ToNot(BeEmpty())
		Expect(filepath.ToSlash(art.SourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/test.mp3"))

		rc, err := store.Open(ia.Hash, art.Mime)
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("absent: no local source and no external error persists a known-absent state", func() {
		folderRepo.result = nil
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al3", Name: "Album"},
		})

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al3"})
		Expect(out).To(Equal(outcomeAbsent))

		ia, err := artRepo.GetItemArtwork("al", "al3", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Hash).To(BeEmpty())
		Expect(ia.Source).To(BeEmpty())
		Expect(ia.AttemptedAt).To(BeTemporally("~", time.Now(), time.Second))
	})

	It("failed-on-extError: leaves the item's state untouched", func() {
		conf.Server.CoverArtPriority = "external"
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al4", Name: "Album"},
		})
		prov.albumImage = func(context.Context, string) (*url.URL, error) {
			return nil, errors.New("agent timed out")
		}

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al4"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork("al", "al4", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("dedup: a second item with identical bytes skips decode and reuses the artwork row", func() {
		folderRepo.result = []model.Folder{{
			Path:       "tests/fixtures/artist/an-album",
			ImageFiles: []string{"cover.jpg"},
		}}
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al5", Name: "Album A", FolderIDs: []string{"f1"}},
			{ID: "al6", Name: "Album B", FolderIDs: []string{"f1"}},
		})

		out1 := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al5"})
		Expect(out1).To(Equal(outcomeFound))
		ia1, err := artRepo.GetItemArtwork("al", "al5", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())

		// Poison the stored blurhash: if the second item re-decodes instead of
		// deduping on hash, this sentinel gets overwritten by a real computed value.
		poisoned := artRepo.Data[ia1.Hash]
		poisoned.BlurHash = "SENTINEL"
		artRepo.Data[ia1.Hash] = poisoned

		out2 := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al6"})
		Expect(out2).To(Equal(outcomeFound))
		ia2, err := artRepo.GetItemArtwork("al", "al6", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia2.Hash).To(Equal(ia1.Hash))

		reused, err := artRepo.GetImage(ia1.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(reused.BlurHash).To(Equal("SENTINEL"))
	})

	It("decode failure on found bytes: fails without writing state", func() {
		tmpDir := GinkgoT().TempDir()
		conf.Server.DataFolder = conf.NewDir(tmpDir)
		Expect(os.MkdirAll(filepath.Join(tmpDir, "artwork", "radio"), 0755)).To(Succeed())
		imgPath := filepath.Join(tmpDir, "artwork", "radio", "ra1_test.jpg")
		Expect(os.WriteFile(imgPath, []byte("not actually an image"), 0600)).To(Succeed())

		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{"ra1": {ID: "ra1", Name: "Radio", UploadedImage: "ra1_test.jpg"}}
		ds.MockedRadio = radioRepo

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "ra1"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork("ra", "ra1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("store write failure: fails without writing state", func() {
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al7", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
		})
		folderRepo.result = nil

		// A store root that is a plain file makes every MkdirAll under it fail.
		blockedRoot := filepath.Join(GinkgoT().TempDir(), "not-a-dir")
		Expect(os.WriteFile(blockedRoot, []byte("x"), 0600)).To(Succeed())
		deps.store = NewImageStore(blockedRoot)

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al7"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork("al", "al7", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
