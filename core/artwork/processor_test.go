package artwork

import (
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// pngHeaderWithDims builds just a PNG signature + IHDR chunk declaring w×h. DecodeConfig
// reads the header without touching pixel data, so the body can be omitted entirely.
func pngHeaderWithDims(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8] = 8 // bit depth
	ihdr[9] = 2 // color type: truecolor
	chunk := append([]byte("IHDR"), ihdr...)
	out := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	out = binary.BigEndian.AppendUint32(out, uint32(len(ihdr)))
	out = append(out, chunk...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(chunk))
}

var _ = Describe("processItem", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		folderRepo *fakeFolderRepo
		libRepo    *tests.MockLibraryRepo
		ffm        *tests.MockFFmpeg
		ag         *agents.Agents
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
		ag = agents.GetAgents(&tests.MockDataStore{}, nil)
		artRepo = tests.CreateMockArtworkRepo()
		ds = &tests.MockDataStore{
			MockedFolder:  folderRepo,
			MockedLibrary: libRepo,
			MockedArtwork: artRepo,
		}
		ds.MockedAlbum = tests.CreateMockAlbumRepo()
		store = NewImageStore(GinkgoT().TempDir())
		deps = &workerDeps{ds: ds, store: store, agents: ag, ffmpeg: ffm}

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
		Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/cover.jpg"))
		Expect(ia.RefMtime).To(BeNumerically(">", 0))

		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
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
		Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/test.mp3"))

		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.BlurHash).ToNot(BeEmpty())

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
		imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al4"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork("al", "al4", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("found-stale: a fallback hit after a transient external failure persists state and returns outcomeFoundStale", func() {
		conf.Server.CoverArtPriority = "external, cover.jpg"
		folderRepo.result = []model.Folder{{
			Path:       "tests/fixtures/artist/an-album",
			ImageFiles: []string{"cover.jpg"},
		}}
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "alstale", Name: "Album", FolderIDs: []string{"f1"}},
		})
		imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alstale"})
		Expect(out).To(Equal(outcomeFoundStale))

		ia, err := artRepo.GetItemArtwork("al", "alstale", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Hash).ToNot(BeEmpty())
		Expect(ia.Source).To(Equal("folder"))
	})

	It("found-external: persists source as external:<agentName> and stores the fetched bytes", func() {
		conf.Server.CoverArtPriority = "external"
		imgBytes, err := os.ReadFile(filepath.Join(repoRoot, "tests/fixtures/artist/an-album/cover.jpg"))
		Expect(err).ToNot(HaveOccurred())
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(imgBytes)
		}))
		DeferCleanup(srv.Close)

		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alext", Name: "Album"}})
		imageAgents(&fakeImageAgent{name: "deezerFake", imgs: []agents.ExternalImage{{URL: srv.URL, Size: 500}}})

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alext"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork("al", "alext", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("external:deezerFake"))
		Expect(ia.Hash).ToNot(BeEmpty())

		// External art is content-addressed into the store, not file-backed.
		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		rc, err := store.Open(ia.Hash, art.Mime)
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
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

	It("two items, two files, identical bytes: each item keeps its own provenance; the shared artwork row is written once", func() {
		// Two distinct library files with byte-identical content resolve to the same
		// hash. Provenance is per-item, so neither file's path may overwrite the other.
		libRoot := GinkgoT().TempDir()
		imgBytes, err := os.ReadFile(filepath.Join(repoRoot, "tests/fixtures/artist/an-album/cover.jpg"))
		Expect(err).ToNot(HaveOccurred())
		for sub, mtime := range map[string]int64{"album-a": 1000, "album-b": 2000} {
			dir := filepath.Join(libRoot, sub)
			Expect(os.MkdirAll(dir, 0755)).To(Succeed())
			img := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(img, imgBytes, 0600)).To(Succeed())
			Expect(os.Chtimes(img, time.Unix(mtime, 0), time.Unix(mtime, 0))).To(Succeed())
		}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(libRoot)}})
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "alA", Name: "Album A", FolderIDs: []string{"fa"}},
			{ID: "alB", Name: "Album B", FolderIDs: []string{"fb"}},
		})

		folderRepo.result = []model.Folder{{Path: "album-a", ImageFiles: []string{"cover.jpg"}}}
		Expect(processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alA"})).To(Equal(outcomeFound))
		iaA, err := artRepo.GetItemArtwork("al", "alA", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(iaA.Source).To(Equal("folder"))
		Expect(filepath.ToSlash(iaA.SourcePath)).To(HaveSuffix("album-a/cover.jpg"))
		Expect(iaA.RefMtime).To(Equal(time.Unix(1000, 0).UnixNano()))

		// Poison the shared row's blurhash: the second item must dedup on hash, not re-decode.
		poisoned := artRepo.Data[iaA.Hash]
		poisoned.BlurHash = "SENTINEL"
		artRepo.Data[iaA.Hash] = poisoned

		folderRepo.result = []model.Folder{{Path: "album-b", ImageFiles: []string{"cover.jpg"}}}
		Expect(processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alB"})).To(Equal(outcomeFound))
		iaB, err := artRepo.GetItemArtwork("al", "alB", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(iaB.Hash).To(Equal(iaA.Hash))
		Expect(filepath.ToSlash(iaB.SourcePath)).To(HaveSuffix("album-b/cover.jpg"))
		Expect(iaB.RefMtime).To(Equal(time.Unix(2000, 0).UnixNano()))

		// The first item's provenance survives the second item processing identical bytes.
		iaAafter, err := artRepo.GetItemArtwork("al", "alA", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(filepath.ToSlash(iaAafter.SourcePath)).To(HaveSuffix("album-a/cover.jpg"))
		Expect(iaAafter.RefMtime).To(Equal(time.Unix(1000, 0).UnixNano()))

		// One shared artwork row, and dedup preserved it untouched.
		Expect(artRepo.Data).To(HaveLen(1))
		reused, err := artRepo.GetImage(iaA.Hash)
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

	It("oversized read: a resolved image larger than the cap fails without writing state", func() {
		tmpDir := GinkgoT().TempDir()
		conf.Server.DataFolder = conf.NewDir(tmpDir)
		Expect(os.MkdirAll(filepath.Join(tmpDir, "artwork", "radio"), 0755)).To(Succeed())
		imgPath := filepath.Join(tmpDir, "artwork", "radio", "big_test.jpg")
		f, err := os.Create(imgPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Truncate(maxImageBytes + 1)).To(Succeed())
		Expect(f.Close()).To(Succeed())

		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{"big": {ID: "big", Name: "Radio", UploadedImage: "big_test.jpg"}}
		ds.MockedRadio = radioRepo

		out := processItem(ctx, deps, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "big"})
		Expect(out).To(Equal(outcomeFailed))

		_, err = artRepo.GetItemArtwork("ra", "big", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("decompression bomb: rejects huge declared dimensions before the full decode", func() {
		data := pngHeaderWithDims(50000, 50000) // 2.5 gigapixels, far above the cap
		_, err := decodeArtwork(ctx, "bomb", data)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dimensions"))
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
