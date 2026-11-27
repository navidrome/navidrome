package e2e

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/tests/harness"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Covers the enqueue → drain → serve chain; per-source resolution rules live in the unit suites.
var _ = Describe("Acquisition → serve loop", func() {
	var (
		ctx        context.Context
		ds         model.DataStore
		artRepo    model.ArtworkRepository
		store      *artwork.ImageStore
		svc        artwork.Artwork
		worker     *artwork.Worker
		coverBytes []byte
	)

	// Requests carry a user, the worker resolves as the first admin, and only admins may store
	// radios and playlists, so every call runs as this admin.
	admin := model.User{ID: "admin-1", UserName: "admin", IsAdmin: true}
	putFolder := func(rel string, images ...string) string {
		GinkgoHelper()
		lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		f := model.NewFolder(*lib, rel)
		f.ImageFiles = images
		Expect(ds.Folder().Put(ctx, f)).To(Succeed())
		return f.ID
	}
	putRadio := func(rd model.Radio) {
		GinkgoHelper()
		Expect(ds.Radio().Put(ctx, &rd)).To(Succeed())
	}

	itemFound := func(kind model.Kind, id string) func() bool {
		return func() bool {
			ia, err := artRepo.GetItemArtwork(ctx, kind, id, model.ImageTypePrimary)
			return err == nil && ia.Hash != ""
		}
	}
	itemAbsent := func(kind model.Kind, id string) func() bool {
		return func() bool {
			ia, err := artRepo.GetItemArtwork(ctx, kind, id, model.ImageTypePrimary)
			return err == nil && ia.Hash == ""
		}
	}
	// Enqueues the way the serving paths do, so the drain is driven by a plain queue row.
	bump := func(kind, id string) {
		GinkgoHelper()
		Expect(ds.ArtworkQueue().EnqueuePreservingBackoff(ctx, model.ArtworkQueueItem{
			ItemKind: kind, ItemID: id, ImageType: model.ImageTypePrimary,
			Priority: model.ArtworkPriorityBump,
		})).To(Succeed())
	}

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), admin)
		repoRoot, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		coverBytes = readFixture(coverFixture)

		conf.Update(func(c *conf.Options) {
			c.CacheFolder = conf.NewDir(GinkgoT().TempDir())
			c.DataFolder = conf.NewDir(GinkgoT().TempDir())
			c.CoverArtPriority = "cover.jpg"
			c.ArtistArtPriority = "artist.png" // keeps artist resolution offline
			c.EnableMediaFileCoverArt = true
			c.DevArtworkWorkerConcurrency = 1
		})

		harness.TruncateDB(GinkgoT().Context(), userTables)
		ds = persistence.New(db.Db())
		artRepo = ds.Artwork()
		u := admin
		Expect(ds.User().Put(ctx, &u)).To(Succeed())
		Expect(ds.Library().Put(ctx, &model.Library{ID: model.DefaultLibraryID, Name: "Music", Path: repoRoot})).To(Succeed())
		ffm := tests.NewMockFFmpeg("")
		store = artwork.NewImageStore(GinkgoT().TempDir())
		// size=0 requests stream originals, so this reader is never called (serving_test covers resizing).
		imgCache := cache.NewFileCache("ArtworkPipelineE2E", "100MB", "images", 0,
			func(context.Context, cache.Item) (io.Reader, error) {
				return nil, errors.New("resize not exercised in e2e")
			})
		Eventually(func() bool { return imgCache.Available(ctx) }, 10*time.Second).Should(BeTrue())

		svc = artwork.NewArtwork(ds, imgCache, store, ffm)
		worker = artwork.NewWorker(ds, store, agents.GetAgents(ds, nil), ffm, events.NoopBroker(), imgCache)
	})

	seedFolderAlbum := func(albumID string) {
		GinkgoHelper()
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: albumID, Name: "Album", FolderIDs: []string{putFolder(albumFolderPath, "cover.jpg")}})
	}

	It("acquires and serves a cover whose format has no registered decoder (#5950)", func() {
		libDir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(libDir, "an-album"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(libDir, "an-album", "cover.jxl"), jxlFixture, 0600)).To(Succeed())

		conf.Update(func(c *conf.Options) { c.CoverArtPriority = "cover.*" })
		persistencetest.SetLibraries(ctx, ds, libDir)
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al1", Name: "Album", FolderIDs: []string{putFolder("an-album", "cover.jxl")}})

		bump("al", "al1")
		runWorkerUntil(ctx, worker, itemFound(model.KindAlbumArtwork, "al1"))

		img, err := svc.Get(ctx, tests.MustParseArtworkID("al-al1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Placeholder).To(BeFalse())
		Expect(readAll(img)).To(Equal(jxlFixture))
	})

	It("acquires album folder art and serves the exact bytes under its hash", func() {
		seedFolderAlbum("al1")
		bump("al", "al1")
		runWorkerUntil(ctx, worker, itemFound(model.KindAlbumArtwork, "al1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("folder"))

		img, err := svc.Get(ctx, tests.MustParseArtworkID("al-al1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		Expect(img.Placeholder).To(BeFalse())
		Expect(readAll(img)).To(Equal(coverBytes))
	})

	It("acquires an artist's uploaded image and serves it", func() {
		name := writeUpload(consts.EntityArtist, "artist-e2e.png", artistPngFixture)
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "ar1", Name: "Artist", UploadedImage: name})
		bump("ar", "ar1")
		runWorkerUntil(ctx, worker, itemFound(model.KindArtistArtwork, "ar1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindArtistArtwork, "ar1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("upload"))

		img, err := svc.Get(ctx, tests.MustParseArtworkID("ar-ar1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		Expect(readAll(img)).To(Equal(readFixture(artistPngFixture)))
	})

	It("generates a playlist grid from its tracks' album art and serves it from the store", func() {
		seedFolderAlbum("al1")
		track := model.MediaFile{ID: "mfpl", AlbumID: "al1", LibraryID: model.DefaultLibraryID, Path: "pl/track.mp3"}
		Expect(ds.MediaFile().Put(ctx, &track)).To(Succeed())
		Expect(ds.Playlist().Put(ctx, &model.Playlist{ID: "pl1", Name: "Playlist", OwnerID: admin.ID,
			Tracks: model.PlaylistTracks{{MediaFileID: track.ID, MediaFile: track}}})).To(Succeed())
		bump("pl", "pl1")
		runWorkerUntil(ctx, worker, itemFound(model.KindPlaylistArtwork, "pl1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindPlaylistArtwork, "pl1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("generated"))

		img, err := svc.Get(ctx, tests.MustParseArtworkID("pl-pl1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		art, err := artRepo.GetImage(ctx, ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Mime).To(Equal("image/png"))
		Expect(readAll(img)).ToNot(BeEmpty())
	})

	It("acquires a radio station's uploaded image and serves it", func() {
		name := writeUpload(consts.EntityRadio, "radio-e2e.jpg", coverFixture)
		putRadio(model.Radio{ID: "ra1", Name: "Station", UploadedImage: name})
		bump("ra", "ra1")
		runWorkerUntil(ctx, worker, itemFound(model.KindRadioArtwork, "ra1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("upload"))

		img, err := svc.Get(ctx, tests.MustParseArtworkID("ra-ra1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Hash).To(Equal(ia.Hash))
		Expect(readAll(img)).To(Equal(coverBytes))
	})

	It("serves an unresolved track provisionally, then upgrades to the worker's state row", func() {
		Expect(ds.MediaFile().Put(ctx, &model.MediaFile{
			ID: "mf1", AlbumID: "al1", HasCoverArt: true, LibraryID: model.DefaultLibraryID, Path: mp3Fixture,
		})).To(Succeed())

		provisional, err := svc.Get(ctx, tests.MustParseArtworkID("mf-mf1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(provisional.Placeholder).To(BeFalse())
		Expect(provisional.Hash).ToNot(BeEmpty())
		provisionalBytes := readAll(provisional)
		Expect(provisionalBytes).ToNot(BeEmpty())

		_, err = artRepo.GetItemArtwork(ctx, model.KindMediaFileArtwork, "mf1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound), "provisional serving must not write a state row")

		// The provisional read enqueued a Bump; drain it.
		runWorkerUntil(ctx, worker, itemFound(model.KindMediaFileArtwork, "mf1"))
		ia, err := artRepo.GetItemArtwork(ctx, model.KindMediaFileArtwork, "mf1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("embedded"))
		Expect(ia.Hash).To(Equal(provisional.Hash))

		resolved, err := svc.Get(ctx, tests.MustParseArtworkID("mf-mf1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(resolved.Hash).To(Equal(ia.Hash))
		Expect(readAll(resolved)).To(Equal(provisionalBytes))
	})

	It("stores dimensions, mime and a real blurhash alongside the acquired bytes", func() {
		seedFolderAlbum("al1")
		bump("al", "al1")
		runWorkerUntil(ctx, worker, itemFound(model.KindAlbumArtwork, "al1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		art, err := artRepo.GetImage(ctx, ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Mime).To(Equal("image/jpeg"))
		Expect(art.Width).To(BeNumerically(">", 0))
		Expect(art.Height).To(BeNumerically(">", 0))
		Expect(art.SizeBytes).To(BeNumerically("==", len(coverBytes)))
		// Never a synthesized value: both hashes are encoded from the real pixels.
		Expect(art.BlurHash).ToNot(BeEmpty())
		Expect(art.ThumbHash).ToNot(BeEmpty())
		raw, err := base64.StdEncoding.DecodeString(art.ThumbHash)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(raw)).To(BeNumerically(">=", 5))
	})

	It("acquires GIF artwork, whose decoder only core/artwork's blank import registers", func() {
		writeUploadedImage(consts.EntityRadio, "station.gif", gifFixture)
		putRadio(model.Radio{ID: "ra1", Name: "Station", UploadedImage: "station.gif"})
		bump("ra", "ra1")
		runWorkerUntil(ctx, worker, itemFound(model.KindRadioArtwork, "ra1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		art, err := artRepo.GetImage(ctx, ia.Hash)
		Expect(err).ToNot(HaveOccurred())
		Expect(art.Mime).To(Equal("image/gif"))
		Expect(art.Width).To(BeNumerically("==", 4))
	})

	It("deduplicates byte-identical art across entities onto one image row", func() {
		fid := putFolder(albumFolderPath, "cover.jpg")
		persistencetest.PutAlbums(ctx, ds,
			model.Album{ID: "al1", Name: "Album", FolderIDs: []string{fid}},
			model.Album{ID: "al2", Name: "Same Cover", FolderIDs: []string{fid}},
		)
		bump("al", "al1")
		bump("al", "al2")
		runWorkerUntil(ctx, worker, func() bool {
			return itemFound(model.KindAlbumArtwork, "al1")() && itemFound(model.KindAlbumArtwork, "al2")()
		})

		ia1, err := artRepo.GetItemArtwork(ctx, model.KindAlbumArtwork, "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		ia2, err := artRepo.GetItemArtwork(ctx, model.KindAlbumArtwork, "al2", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia1.Hash).To(Equal(ia2.Hash), "identical bytes must share one content hash")
		Expect(readAll(mustGet(svc.Get(ctx, tests.MustParseArtworkID("al-al2"), 0, false)))).To(Equal(coverBytes))
	})

	It("stops serving a file-backed image once its source file changes underneath", func() {
		name := writeUpload(consts.EntityRadio, "radio-stale.jpg", coverFixture)
		putRadio(model.Radio{ID: "ra1", Name: "Station", UploadedImage: name})
		bump("ra", "ra1")
		runWorkerUntil(ctx, worker, itemFound(model.KindRadioArtwork, "ra1"))

		ia, err := artRepo.GetItemArtwork(ctx, model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		staleHash := ia.Hash

		path := model.UploadedImagePath(consts.EntityRadio, name)
		Expect(os.WriteFile(path, readFixture(artistPngFixture), 0o600)).To(Succeed())
		newer := time.Now().Add(2 * time.Second)
		Expect(os.Chtimes(path, newer, newer)).To(Succeed())

		// The mtime no longer matches the state row, so the stale bytes are not served.
		_, err = svc.Get(ctx, tests.MustParseArtworkID("ra-ra1"), 0, false)
		Expect(err).To(MatchError(artwork.ErrUnavailable))

		// That failed read enqueued a re-resolution.
		runWorkerUntil(ctx, worker, func() bool {
			cur, gerr := artRepo.GetItemArtwork(ctx, model.KindRadioArtwork, "ra1", model.ImageTypePrimary)
			return gerr == nil && cur.Hash != "" && cur.Hash != staleHash
		})
		img, err := svc.Get(ctx, tests.MustParseArtworkID("ra-ra1"), 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(readAll(img)).To(Equal(readFixture(artistPngFixture)))
	})

	It("records an absent state for an entity with no art and reports it unavailable", func() {
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: "alx", Name: "Artless"})
		bump("al", "alx")
		runWorkerUntil(ctx, worker, itemAbsent(model.KindAlbumArtwork, "alx"))

		_, err := svc.Get(ctx, tests.MustParseArtworkID("al-alx"), 0, false)
		Expect(err).To(MatchError(artwork.ErrUnavailable))

		img, err := svc.GetOrPlaceholder(ctx, "al-alx", 0, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img.Placeholder).To(BeTrue())
	})
})

func mustGet(img *artwork.Image, err error) *artwork.Image {
	GinkgoHelper()
	Expect(err).ToNot(HaveOccurred())
	return img
}

// Raw bytes on purpose: encoding a GIF here would register image/gif in the test binary, masking
// jxlFixture is a JPEG XL bare codestream header: a real image format, with no stdlib decoder.
var jxlFixture = []byte{0xff, 0x0a, 0x00, 0x10, 0x00}

// the production import the spec above guards.
var gifFixture = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x04, 0x00, 0x04, 0x00, 0x80, 0x00,
	0x00, 0x2e, 0x86, 0xc1, 0xf4, 0xd0, 0x3f, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x04, 0x00, 0x04, 0x00, 0x00, 0x02, 0x05, 0x44, 0x7c, 0x67, 0xb8, 0x05,
	0x00, 0x3b,
}
