package artwork

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// jxlCodestream is a JPEG XL bare codestream header: a real image format, with no stdlib decoder.
var jxlCodestream = []byte{0xff, 0x0a, 0x00, 0x10, 0x00}

// DecodeConfig reads only the header, so the pixel data can be omitted entirely.
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

var _ = Describe("processor.acquire", func() {
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
		proc       *processor
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
		proc = &processor{ds: ds, store: store, resolver: newResolver(ds, ag, ffm, nil)}

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

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al1"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Hash).ToNot(BeEmpty())
		Expect(ia.Source).To(Equal("folder"))
		Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/cover.jpg"))
		Expect(ia.RefMtime).To(BeNumerically(">", 0))

		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		// Every placeholder is derived from the one shared thumbnail, so all three land together.
		Expect(art.BlurHash).ToNot(BeEmpty())
		Expect(art.ThumbHash).ToNot(BeEmpty())
		Expect(art.DominantColor).To(MatchRegexp(`^#[0-9a-f]{6}$`))
		_, err = store.Open(ia.Hash, art.Mime)
		Expect(os.IsNotExist(err)).To(BeTrue(), "folder-backed art must not be duplicated into the store")
	})

	Describe("prune lock scope", func() {
		var lock *countingLocker

		BeforeEach(func() {
			lock = &countingLocker{}
			proc.pruneLock = lock
		})

		It("holds it once across the write window", func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "alL1", Name: "Album", FolderIDs: []string{"f1"}},
			})

			out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alL1"})
			Expect(out).To(Equal(outcomeFound))
			Expect(lock.locks).To(BeNumerically(">", 0), "the write window must exclude prune")
			Expect(lock.held()).To(BeFalse(), "the window must close before acquire returns")
		})

		// Resolution can reach the network, so holding the lock across it would let one slow
		// provider block prune, and every drain queued behind prune's writer.
		It("never takes it while only resolving", func() {
			folderRepo.result = nil
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "alL2", Name: "Album", FolderIDs: []string{"f1"}},
			})

			out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alL2"})
			Expect(out).To(Equal(outcomeAbsent))
			Expect(lock.locks).To(BeZero())
		})
	})

	It("found-embedded: writes a store file and computes a non-empty blurhash from a real fixture", func() {
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al2", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
		})
		folderRepo.result = nil

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al2", model.ImageTypePrimary)
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

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al3"})
		Expect(out).To(Equal(outcomeAbsent))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al3", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Hash).To(BeEmpty())
		Expect(ia.Source).To(BeEmpty())
		Expect(ia.AttemptedAt).To(BeTemporally("~", time.Now(), time.Second))
	})

	It("failed-on-unreadable-local: a listed cover that will not open never records absent", func() {
		conf.Server.CoverArtPriority = "cover.jpg"
		// A folder listing that names a cover the FS will not hand over: a stale mount.
		libRoot := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(libRoot, "an-album"), 0o755)).To(Succeed())
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(libRoot)}})
		folderRepo.result = []model.Folder{{Path: "an-album", ImageFiles: []string{"cover.jpg"}}}
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al-io", Name: "Album", FolderIDs: []string{"f1"}},
		})

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al-io"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al-io", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "an I/O fault must not be recorded as absent")
	})

	// An upload outranks every source, so an unreadable one must not let a lower one take over.
	It("failed-on-unreadable-upload: an upload that will not open never records absent", func() {
		if runtime.GOOS == "windows" {
			// The file would still open, so the spec would pass on the decode error instead.
			Skip("chmod does not restrict read access on Windows")
		}
		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{}
		ds.MockedRadio = radioRepo
		dir := GinkgoT().TempDir()
		conf.Server.DataFolder = conf.NewDir(dir)
		upload := model.UploadedImagePath(consts.EntityRadio, "ra-io.jpg")
		Expect(os.MkdirAll(filepath.Dir(upload), 0o755)).To(Succeed())
		Expect(os.WriteFile(upload, []byte("x"), 0o600)).To(Succeed())
		Expect(os.Chmod(upload, 0o000)).To(Succeed()) // present, but unreadable
		DeferCleanup(func() { _ = os.Chmod(upload, 0o600) })
		radioRepo.Data["ra-io"] = &model.Radio{ID: "ra-io", Name: "Station", UploadedImage: "ra-io.jpg"}

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "ra-io"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindRadioArtwork, "ra-io", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "an unreadable upload must not be recorded as absent")
	})

	// Playlist/radio resolvers walk no chain, so a fault records no step; without a fallback,
	// explain would show a give-up with an empty "Gave up after" table.
	It("chainless fault: records a fallback trace step naming the faulted source", func() {
		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{}
		ds.MockedRadio = radioRepo
		dir := GinkgoT().TempDir()
		conf.Server.DataFolder = conf.NewDir(dir)
		upload := model.UploadedImagePath(consts.EntityRadio, "ra-tr.jpg")
		// A plain file where the upload's parent should be makes os.Open fault with ENOTDIR,
		// deterministically and regardless of the test user's privileges.
		Expect(os.MkdirAll(filepath.Dir(filepath.Dir(upload)), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Dir(upload), []byte("x"), 0o600)).To(Succeed())
		radioRepo.Data["ra-tr"] = &model.Radio{ID: "ra-tr", Name: "Station", UploadedImage: "ra-tr.jpg"}

		trace := &ChainTrace{}
		out, _ := proc.acquire(withTrace(ctx, trace), model.ArtworkQueueItem{ItemKind: "ra", ItemID: "ra-tr"})
		Expect(out).To(Equal(outcomeFailed))

		steps := trace.Steps()
		Expect(steps).To(HaveLen(1), "a radio fault must leave one step so explain is not blank")
		Expect(steps[0].Candidate).To(Equal("upload"))
		Expect(steps[0].Outcome).To(Equal(OutcomeUnreadable))
	})

	It("failed-on-extError: leaves the item's state untouched", func() {
		conf.Server.CoverArtPriority = "external"
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al4", Name: "Album"},
		})
		imageAgents(&fakeImageAgent{name: "failAgent", err: errors.New("agent timed out")})

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al4"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al4", model.ImageTypePrimary)
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

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alstale"})
		Expect(out).To(Equal(outcomeFoundStale))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alstale", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Hash).ToNot(BeEmpty())
		Expect(ia.Source).To(Equal("folder"))
	})

	It("undecodable local file: acquires it anyway, with no placeholder metadata", func() {
		libRoot := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(libRoot, "album"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(libRoot, "album", "cover.jpg"), jxlCodestream, 0600)).To(Succeed())
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(libRoot)}})
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alU", Name: "Album", FolderIDs: []string{"f1"}}})
		folderRepo.result = []model.Folder{{Path: "album", ImageFiles: []string{"cover.jpg"}}}

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alU"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alU", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("folder"))
		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Width).To(BeZero())
		Expect(art.BlurHash).To(BeEmpty())
	})

	It("empty local file: fails without writing state", func() {
		libRoot := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(libRoot, "album"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(libRoot, "album", "cover.jpg"), nil, 0600)).To(Succeed())
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(libRoot)}})
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alE", Name: "Album", FolderIDs: []string{"f1"}}})
		folderRepo.result = []model.Folder{{Path: "album", ImageFiles: []string{"cover.jpg"}}}

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alE"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alE", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	// An agent answering 200 with a non-image body must keep retrying, not pin garbage as a cover.
	It("undecodable external body: fails without writing state", func() {
		conf.Server.CoverArtPriority = "external"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>rate limited</html>"))
		}))
		DeferCleanup(srv.Close)

		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alX", Name: "Album"}})
		imageAgents(&fakeImageAgent{name: "deezerFake", imgs: []agents.ExternalImage{{URL: srv.URL, Size: 500}}})

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alX"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alX", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
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

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alext"})
		Expect(out).To(Equal(outcomeFound))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alext", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("external:deezerFake"))
		Expect(ia.Hash).ToNot(BeEmpty())

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

		out1, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al5"})
		Expect(out1).To(Equal(outcomeFound))
		ia1, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al5", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())

		// A re-decode instead of a hash dedup would overwrite this sentinel.
		poisoned := artRepo.Data[ia1.Hash]
		poisoned.BlurHash = "SENTINEL"
		artRepo.Data[ia1.Hash] = poisoned

		out2, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al6"})
		Expect(out2).To(Equal(outcomeFound))
		ia2, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al6", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia2.Hash).To(Equal(ia1.Hash))

		reused, err := artRepo.GetImage(ia1.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(reused.BlurHash).To(Equal("SENTINEL"))
	})

	It("two items, two files, identical bytes: each item keeps its own provenance; the shared artwork row is written once", func() {
		// Byte-identical files share one hash, but provenance is per item.
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
		outN, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alA"})
		Expect(outN).To(Equal(outcomeFound))
		iaA, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alA", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(iaA.Source).To(Equal("folder"))
		Expect(filepath.ToSlash(iaA.SourcePath)).To(HaveSuffix("album-a/cover.jpg"))
		Expect(iaA.RefMtime).To(Equal(time.Unix(1000, 0).UnixNano()))

		// A re-decode instead of a hash dedup would overwrite this sentinel.
		poisoned := artRepo.Data[iaA.Hash]
		poisoned.BlurHash = "SENTINEL"
		artRepo.Data[iaA.Hash] = poisoned

		folderRepo.result = []model.Folder{{Path: "album-b", ImageFiles: []string{"cover.jpg"}}}
		outN, _ = proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alB"})
		Expect(outN).To(Equal(outcomeFound))
		iaB, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alB", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(iaB.Hash).To(Equal(iaA.Hash))
		Expect(filepath.ToSlash(iaB.SourcePath)).To(HaveSuffix("album-b/cover.jpg"))
		Expect(iaB.RefMtime).To(Equal(time.Unix(2000, 0).UnixNano()))

		iaAafter, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "alA", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(filepath.ToSlash(iaAafter.SourcePath)).To(HaveSuffix("album-a/cover.jpg"))
		Expect(iaAafter.RefMtime).To(Equal(time.Unix(1000, 0).UnixNano()))

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
		// Truncated PNG: a known format, so this is a real decode failure, not a missing decoder.
		Expect(os.WriteFile(imgPath, pngHeaderWithDims(100, 100), 0600)).To(Succeed())

		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{"ra1": {ID: "ra1", Name: "Radio", UploadedImage: "ra1_test.jpg"}}
		ds.MockedRadio = radioRepo

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "ra1"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("oversized read: a resolved image larger than the cap fails without writing state", func() {
		tmpDir := GinkgoT().TempDir()
		conf.Server.DataFolder = conf.NewDir(tmpDir)
		Expect(os.MkdirAll(filepath.Join(tmpDir, "artwork", "radio"), 0755)).To(Succeed())
		imgPath := filepath.Join(tmpDir, "artwork", "radio", "big_test.jpg")
		f, err := os.Create(imgPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Truncate(maxImageBytes() + 1)).To(Succeed())
		Expect(f.Close()).To(Succeed())

		radioRepo := tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{"big": {ID: "big", Name: "Radio", UploadedImage: "big_test.jpg"}}
		ds.MockedRadio = radioRepo

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "big"})
		Expect(out).To(Equal(outcomeFailed))

		_, err = artRepo.GetItemArtwork(model.KindRadioArtwork, "big", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	// The cap is compared by division so an overflowing product cannot slip past it; no supported
	// format can declare these dimensions, so this pins the arithmetic, not a live hole.
	DescribeTable("rejects out-of-range declared dimensions",
		func(w, h uint32) {
			_, err := decodeArtwork(ctx, "bomb", pngHeaderWithDims(w, h))
			Expect(err).To(HaveOccurred())
		},
		Entry("both dimensions at the 32-bit maximum", uint32(0xffffffff), uint32(0xffffffff)),
		Entry("both dimensions at the signed 32-bit maximum", uint32(0x7fffffff), uint32(0x7fffffff)),
		Entry("zero width", uint32(0), uint32(100)),
		Entry("zero height", uint32(100), uint32(0)),
	)

	It("decompression bomb: rejects huge declared dimensions before the full decode", func() {
		data := pngHeaderWithDims(50000, 50000) // 2.5 gigapixels, far above the cap
		_, err := decodeArtwork(ctx, "bomb", data)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dimensions"))
	})

	It("unknown format: reports ErrFormat so the caller can decide", func() {
		_, err := decodeArtwork(ctx, "jxl", jxlCodestream)
		Expect(err).To(MatchError(image.ErrFormat))
	})

	It("undecodedArtwork: carries the hash and mime, and nothing a decode would add", func() {
		art := undecodedArtwork("jxl")
		Expect(art.Hash).To(Equal("jxl"))
		Expect(art.Mime).To(Equal("application/octet-stream"))
		Expect(art.Width).To(BeZero())
		Expect(art.Height).To(BeZero())
		Expect(art.BlurHash).To(BeEmpty())
		Expect(art.ThumbHash).To(BeEmpty())
		Expect(art.DominantColor).To(BeEmpty())
	})

	It("corrupt image of a known format: still fails", func() {
		data := pngHeaderWithDims(100, 100) // header declares a decodable size, body is missing
		_, err := decodeArtwork(ctx, "truncated", data)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("decode image"))
	})

	// Without this a metadata-less row would be reused forever, so a decoder added later
	// could never upgrade it.
	It("metadata-less row: re-decodes on reuse instead of skipping", func() {
		libRoot := GinkgoT().TempDir()
		imgBytes, err := os.ReadFile(filepath.Join(repoRoot, "tests/fixtures/artist/an-album/cover.jpg"))
		Expect(err).ToNot(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(libRoot, "album"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(libRoot, "album", "cover.jpg"), imgBytes, 0600)).To(Succeed())
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(libRoot)}})
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "alM", Name: "Album", FolderIDs: []string{"f1"}}})
		folderRepo.result = []model.Folder{{Path: "album", ImageFiles: []string{"cover.jpg"}}}

		hash, err := hashImage(bytes.NewReader(imgBytes))
		Expect(err).ToNot(HaveOccurred())
		Expect(artRepo.PutImage(&model.Artwork{Hash: hash, Mime: "application/octet-stream"})).To(Succeed())

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "alM"})
		Expect(out).To(Equal(outcomeFound))

		upgraded, err := artRepo.GetImage(hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(upgraded.Width).To(BeNumerically(">", 0))
		Expect(upgraded.BlurHash).ToNot(BeEmpty())
	})

	It("store write failure: fails without writing state", func() {
		ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al7", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
		})
		folderRepo.result = nil

		// A store root that is a plain file makes every MkdirAll under it fail.
		blockedRoot := filepath.Join(GinkgoT().TempDir(), "not-a-dir")
		Expect(os.WriteFile(blockedRoot, []byte("x"), 0600)).To(Succeed())
		proc.store = NewImageStore(blockedRoot)

		out, _ := proc.acquire(ctx, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al7"})
		Expect(out).To(Equal(outcomeFailed))

		_, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al7", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})

type countingLocker struct {
	locks   int
	unlocks int
}

func (l *countingLocker) Lock()      { l.locks++ }
func (l *countingLocker) Unlock()    { l.unlocks++ }
func (l *countingLocker) held() bool { return l.locks != l.unlocks }

var _ = Describe("makeThumbnail", func() {
	// Both encoders read Pix directly and thumbhash needs straight alpha, so emitting NRGBA is
	// what keeps either of them from allocating a converted copy of the thumbnail.
	It("emits NRGBA when it downscales", func() {
		src := image.NewRGBA(image.Rect(0, 0, 400, 300))
		Expect(makeThumbnail(src, 100)).To(BeAssignableToTypeOf(&image.NRGBA{}))
	})

	It("fits the longest side to maxSize and keeps the aspect ratio", func() {
		src := image.NewRGBA(image.Rect(0, 0, 400, 300))
		b := makeThumbnail(src, 100).Bounds()
		Expect(b.Dx()).To(Equal(100))
		Expect(b.Dy()).To(Equal(75))
	})

	It("never upscales an image already within bounds", func() {
		src := image.NewRGBA(image.Rect(0, 0, 40, 30))
		Expect(makeThumbnail(src, 100).Bounds()).To(Equal(src.Bounds()))
	})

	// A fully transparent pixel's colour cannot survive any resample, since the scaler works in
	// premultiplied space and multiplying by zero is not invertible. Partial alpha is the case
	// that distinguishes straight storage from premultiplied.
	It("keeps partly transparent colour straight rather than premultiplied", func() {
		src := image.NewNRGBA(image.Rect(0, 0, 400, 400))
		for y := range 400 {
			for x := range 400 {
				src.SetNRGBA(x, y, color.NRGBA{R: 200, G: 40, B: 90, A: 128})
			}
		}
		thumb, ok := makeThumbnail(src, 100).(*image.NRGBA)
		Expect(ok).To(BeTrue())
		// Premultiplied storage would have halved this to ~100.
		Expect(thumb.Pix[0]).To(BeNumerically("==", 200))
		Expect(thumb.Pix[3]).To(BeNumerically("==", 128))
	})
})

var _ = Describe("maxImageBytes", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("reads MaxImageSize from config", func() {
		conf.Server.MaxImageSize = "30MB"
		Expect(maxImageBytes()).To(Equal(int64(30_000_000)))
	})
})
