package e2e

import (
	"context"
	"errors"
	"io"
	"os"

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
	itemFound := func(kind, id string) func() bool {
		return func() bool {
			ia, err := artRepo.GetItemArtwork(kind, id, model.ImageTypePrimary)
			return err == nil && ia.Hash != ""
		}
	}
	itemAbsent := func(kind, id string) func() bool {
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
		conf.Server.DevArtworkWorkerConcurrency = 1

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
		runWorkerUntil(ctx, worker, itemFound("al", "al1"))

		ia, err := artRepo.GetItemArtwork("al", "al1", model.ImageTypePrimary)
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
		runWorkerUntil(ctx, worker, itemFound("ar", "ar1"))

		ia, err := artRepo.GetItemArtwork("ar", "ar1", model.ImageTypePrimary)
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
		runWorkerUntil(ctx, worker, itemFound("pl", "pl1"))

		ia, err := artRepo.GetItemArtwork("pl", "pl1", model.ImageTypePrimary)
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
		runWorkerUntil(ctx, worker, itemFound("ra", "ra1"))

		ia, err := artRepo.GetItemArtwork("ra", "ra1", model.ImageTypePrimary)
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

		_, err = artRepo.GetItemArtwork("mf", "mf1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "provisional serving must not write a state row")

		// The provisional read enqueued a Bump; drain it and confirm the persisted hash matches.
		runWorkerUntil(ctx, worker, itemFound("mf", "mf1"))
		ia, err := artRepo.GetItemArtwork("mf", "mf1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("embedded"))
		Expect(ia.Hash).To(Equal(provisional.Hash))

		// Second read: now served from the persisted state row / store, same bytes.
		resolved, err := svc.Get(ctx, model.MustParseArtworkID("mf-mf1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(resolved.Hash).To(Equal(ia.Hash))
		Expect(readAll(resolved)).To(Equal(provisionalBytes))
	})

	It("records an absent state for an entity with no art and reports it unavailable", func() {
		albumRepo.SetData(model.Albums{{ID: "alx", Name: "Artless", LibraryID: 0}})
		worker.Bump("al", "alx")
		runWorkerUntil(ctx, worker, itemAbsent("al", "alx"))

		_, err := svc.Get(ctx, model.MustParseArtworkID("al-alx"), 0, false)
		Expect(err).To(MatchError(artwork.ErrUnavailable))

		img, err := svc.GetOrPlaceholder(ctx, "al-alx", 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Placeholder).To(BeTrue())
	})
})
