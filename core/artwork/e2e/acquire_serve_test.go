package e2e

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These specs wire the real Worker and Service over the same ImageStore and mock repositories,
// then drive the full enqueue → drain → serve loop. They assert the integration of the chain, not
// the per-source resolution rules (which the unit suites in package artwork already cover).
var _ = Describe("Acquisition → serve loop", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		artRepo    *tests.MockArtworkRepo
		queueRepo  *tests.MockArtworkQueueRepo
		albumRepo  *tests.MockAlbumRepo
		artistRepo *tests.MockArtistRepo
		mfRepo     *tests.MockMediaFileRepo
		plRepo     *tests.MockPlaylistRepo
		radioRepo  *tests.MockedRadioRepo
		folderRepo *fakeFolderRepo
		libRepo    *tests.MockLibraryRepo
		store      *artwork.ImageStore
		svc        artwork.Service
		worker     *artwork.Worker
		coverBytes []byte
	)

	// itemFound reports whether the worker has persisted a resolved (hash-bearing) state row.
	itemFound := func(kind model.Kind, id string) func() bool {
		return func() bool {
			ia, err := artRepo.GetItemArtwork(kind, id, model.ImageTypePrimary)
			return err == nil && ia.Hash != ""
		}
	}
	itemAbsent := func(kind model.Kind, id string) func() bool {
		return func() bool {
			ia, err := artRepo.GetItemArtwork(kind, id, model.ImageTypePrimary)
			return err == nil && ia.Hash == ""
		}
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		coverBytes = readFixture(coverFixture)

		conf.Server.CacheFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.CoverArtPriority = "cover.jpg"
		conf.Server.ArtistArtPriority = "artist.png" // upload wins first; kept offline as a safety net
		conf.Server.EnableMediaFileCoverArt = true
		conf.Server.ArtworkWorkerConcurrency = 1

		folderRepo = &fakeFolderRepo{}
		libRepo = &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: repoRoot}})
		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		albumRepo = tests.CreateMockAlbumRepo()
		artistRepo = tests.CreateMockArtistRepo()
		mfRepo = tests.CreateMockMediaFileRepo()
		plRepo = tests.CreateMockPlaylistRepo()
		radioRepo = tests.CreateMockedRadioRepo()
		radioRepo.Data = map[string]*model.Radio{}
		ds = &tests.MockDataStore{
			MockedArtwork:      artRepo,
			MockedArtworkQueue: queueRepo,
			MockedAlbum:        albumRepo,
			MockedArtist:       artistRepo,
			MockedMediaFile:    mfRepo,
			MockedPlaylist:     plRepo,
			MockedRadio:        radioRepo,
			MockedFolder:       folderRepo,
			MockedLibrary:      libRepo,
		}
		ffm := tests.NewMockFFmpeg("")
		store = artwork.NewImageStore(GinkgoT().TempDir())
		// size=0 requests stream originals and never touch the resize cache, so this reader is a
		// compile-time stand-in only; the resize path is covered by the package's serving_test.
		imgCache := cache.NewFileCache("ArtworkPipelineE2E", "100MB", "images", 0,
			func(context.Context, cache.Item) (io.Reader, error) {
				return nil, errors.New("resize not exercised in e2e")
			})
		Eventually(func() bool { return imgCache.Available(ctx) }).Should(BeTrue())

		svc = artwork.NewService(ds, imgCache, store, ffm)
		worker = artwork.NewWorker(ds, store, agents.GetAgents(ds, nil), ffm, events.NoopBroker(), imgCache)
	})

	// seedFolderAlbum wires an album backed by the real fixture folder cover, shared by the album
	// and playlist-grid scenarios.
	seedFolderAlbum := func(albumID string) {
		folderRepo.result = []model.Folder{{Path: albumFolderPath, ImageFiles: []string{"cover.jpg"}}}
		albumRepo.SetData(model.Albums{{ID: albumID, Name: "Album", FolderIDs: []string{"f1"}, LibraryID: 0}})
	}

	It("acquires album folder art and serves the exact bytes under its hash", func() {
		seedFolderAlbum("al1")
		worker.Bump("al", "al1")
		runWorkerUntil(ctx, worker, itemFound(model.KindAlbumArtwork, "al1"))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("folder"))

		img, err := svc.Get(ctx, model.MustParseArtworkID("al-al1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		Expect(img.Placeholder).To(BeFalse())
		Expect(readAll(img)).To(Equal(coverBytes))
	})

	It("acquires an artist's uploaded image and serves it", func() {
		name := writeUpload(consts.EntityArtist, "artist-e2e.png", artistPngFixture)
		artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist", UploadedImage: name}})
		worker.Bump("ar", "ar1")
		runWorkerUntil(ctx, worker, itemFound(model.KindArtistArtwork, "ar1"))

		ia, err := artRepo.GetItemArtwork(model.KindArtistArtwork, "ar1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("upload"))

		img, err := svc.Get(ctx, model.MustParseArtworkID("ar-ar1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		Expect(readAll(img)).To(Equal(readFixture(artistPngFixture)))
	})

	It("generates a playlist grid from its tracks' album art and serves it from the store", func() {
		seedFolderAlbum("al1")
		plRepo.SetData(model.Playlists{{ID: "pl1", Name: "Playlist"}})
		plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"al1"}}
		worker.Bump("pl", "pl1")
		runWorkerUntil(ctx, worker, itemFound(model.KindPlaylistArtwork, "pl1"))

		ia, err := artRepo.GetItemArtwork(model.KindPlaylistArtwork, "pl1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("generated"))

		img, err := svc.Get(ctx, model.MustParseArtworkID("pl-pl1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		// The generated grid is a fresh PNG placed in the content-addressed store.
		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Mime).To(Equal("image/png"))
		Expect(len(readAll(img))).To(BeNumerically(">", 0))
	})

	It("acquires a radio station's uploaded image and serves it", func() {
		name := writeUpload(consts.EntityRadio, "radio-e2e.jpg", coverFixture)
		radioRepo.Data["ra1"] = &model.Radio{ID: "ra1", Name: "Station", UploadedImage: name}
		worker.Bump("ra", "ra1")
		runWorkerUntil(ctx, worker, itemFound(model.KindRadioArtwork, "ra1"))

		ia, err := artRepo.GetItemArtwork(model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("upload"))

		img, err := svc.Get(ctx, model.MustParseArtworkID("ra-ra1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		Expect(readAll(img)).To(Equal(coverBytes))
	})

	It("serves an unresolved track provisionally, then upgrades to the worker's state row", func() {
		mfRepo.SetData(model.MediaFiles{{
			ID: "mf1", AlbumID: "al1", HasCoverArt: true, LibraryID: 0, Path: mp3Fixture,
		}})

		// First read: no state row yet → extract embedded art provisionally and enqueue the track.
		provisional, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(provisional.Placeholder).To(BeFalse())
		Expect(provisional.Hash).ToNot(BeEmpty())
		provisionalBytes := readAll(provisional)
		Expect(len(provisionalBytes)).To(BeNumerically(">", 0))

		_, err = artRepo.GetItemArtwork(model.KindMediaFileArtwork, "mf1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "provisional serving must not write a state row")

		// The provisional read enqueued a Bump; drain it and confirm the persisted hash matches.
		runWorkerUntil(ctx, worker, itemFound(model.KindMediaFileArtwork, "mf1"))
		ia, err := artRepo.GetItemArtwork(model.KindMediaFileArtwork, "mf1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("embedded"))
		Expect(ia.Hash).To(Equal(provisional.Hash))

		// Second read: now served from the persisted state row / store, same bytes.
		resolved, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(resolved.Hash).To(Equal(ia.Hash))
		Expect(readAll(resolved)).To(Equal(provisionalBytes))
	})

	It("stores dimensions, mime and a real blurhash alongside the acquired bytes", func() {
		seedFolderAlbum("al1")
		worker.Bump("al", "al1")
		runWorkerUntil(ctx, worker, itemFound(model.KindAlbumArtwork, "al1"))

		ia, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Mime).To(Equal("image/jpeg"))
		Expect(art.Width).To(BeNumerically(">", 0))
		Expect(art.Height).To(BeNumerically(">", 0))
		Expect(art.SizeBytes).To(BeNumerically("==", len(coverBytes)))
		// Never a synthesized value: the blurhash is encoded from the real pixels.
		Expect(art.BlurHash).ToNot(BeEmpty())
	})

	It("acquires GIF artwork, whose decoder only core/artwork's blank import registers", func() {
		writeUploadedImage(consts.EntityRadio, "station.gif", gifFixture)
		radioRepo.Data["ra1"] = &model.Radio{ID: "ra1", Name: "Station", UploadedImage: "station.gif"}
		worker.Bump("ra", "ra1")
		runWorkerUntil(ctx, worker, itemFound(model.KindRadioArtwork, "ra1"))

		ia, err := artRepo.GetItemArtwork(model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		art, err := artRepo.GetImage(ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Mime).To(Equal("image/gif"))
		Expect(art.Width).To(BeNumerically("==", 4))
	})

	It("deduplicates byte-identical art across entities onto one image row", func() {
		folderRepo.result = []model.Folder{{Path: albumFolderPath, ImageFiles: []string{"cover.jpg"}}}
		albumRepo.SetData(model.Albums{
			{ID: "al1", Name: "Album", FolderIDs: []string{"f1"}, LibraryID: 0},
			{ID: "al2", Name: "Same Cover", FolderIDs: []string{"f1"}, LibraryID: 0},
		})
		worker.Bump("al", "al1")
		worker.Bump("al", "al2")
		runWorkerUntil(ctx, worker, func() bool {
			return itemFound(model.KindAlbumArtwork, "al1")() && itemFound(model.KindAlbumArtwork, "al2")()
		})

		ia1, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		ia2, err := artRepo.GetItemArtwork(model.KindAlbumArtwork, "al2", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia1.Hash).To(Equal(ia2.Hash), "identical bytes must share one content hash")
		Expect(readAll(mustGet(svc.Get(ctx, model.MustParseArtworkID("al-al2"), 0, false)))).To(Equal(coverBytes))
	})

	It("stops serving a file-backed image once its source file changes underneath", func() {
		name := writeUpload(consts.EntityRadio, "radio-stale.jpg", coverFixture)
		radioRepo.Data["ra1"] = &model.Radio{ID: "ra1", Name: "Station", UploadedImage: name}
		worker.Bump("ra", "ra1")
		runWorkerUntil(ctx, worker, itemFound(model.KindRadioArtwork, "ra1"))

		ia, err := artRepo.GetItemArtwork(model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		staleHash := ia.Hash

		// Replace the backing file with different bytes and a newer mtime.
		path := model.UploadedImagePath(consts.EntityRadio, name)
		Expect(os.WriteFile(path, readFixture(artistPngFixture), 0o600)).To(Succeed())
		newer := time.Now().Add(2 * time.Second)
		Expect(os.Chtimes(path, newer, newer)).To(Succeed())

		// The mtime no longer matches the state row, so the old hash's bytes are never served.
		_, err = svc.Get(ctx, model.MustParseArtworkID("ra-ra1"), 0, false)
		Expect(err).To(MatchError(artwork.ErrUnavailable))

		// That read enqueued a re-resolution; draining it republishes the new bytes.
		runWorkerUntil(ctx, worker, func() bool {
			cur, gerr := artRepo.GetItemArtwork(model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
			return gerr == nil && cur.Hash != "" && cur.Hash != staleHash
		})
		img, err := svc.Get(ctx, model.MustParseArtworkID("ra-ra1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(readAll(img)).To(Equal(readFixture(artistPngFixture)))
	})

	It("records an absent state for an entity with no art and reports it unavailable", func() {
		albumRepo.SetData(model.Albums{{ID: "alx", Name: "Artless", LibraryID: 0}})
		worker.Bump("al", "alx")
		runWorkerUntil(ctx, worker, itemAbsent(model.KindAlbumArtwork, "alx"))

		_, err := svc.Get(ctx, model.MustParseArtworkID("al-alx"), 0, false)
		Expect(err).To(MatchError(artwork.ErrUnavailable))

		img, err := svc.GetOrPlaceholder(ctx, "al-alx", 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Placeholder).To(BeTrue())
	})
})

// mustGet unwraps a Service.Get result for inline byte assertions.
func mustGet(img *artwork.Image, err error) *artwork.Image {
	GinkgoHelper()
	Expect(err).ToNot(HaveOccurred())
	return img
}

// gifFixture is a 4x4 GIF held as raw bytes on purpose: encoding one would import image/gif into
// this test binary and register the decoder, masking the production import the spec above guards.
var gifFixture = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x04, 0x00, 0x04, 0x00, 0x80, 0x00,
	0x00, 0x2e, 0x86, 0xc1, 0xf4, 0xd0, 0x3f, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x04, 0x00, 0x04, 0x00, 0x00, 0x02, 0x05, 0x44, 0x7c, 0x67, 0xb8, 0x05,
	0x00, 0x3b,
}
