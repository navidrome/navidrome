package e2e

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/squirrel"
	_ "github.com/navidrome/navidrome/adapters/gotaglib" // registers the "taglib" local-storage extractor
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	_ "github.com/navidrome/navidrome/core/storage/local" // registers the "file" storage scheme
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/tests/harness"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.senan.xyz/taglib"
)

// The serving path streams folder-backed originals via os.Open, which a fake FS cannot back, so
// this suite scans a small REAL on-disk library and drives the real worker and artwork.Artwork.
var _ = Describe("Artwork Serving", Ordered, func() {
	var (
		artRouter   *subsonic.Router
		pubRouter   *public.Router
		artSvc      artwork.Artwork
		worker      *artwork.Worker
		artfulID    string
		artlessID   string
		artfulHash  string
		placeholder []byte
	)

	albumCoverInList := func(albumID string) string {
		GinkgoHelper()
		resp := doReq("getAlbumList2", "type", "alphabeticalByName", "size", "500")
		Expect(resp.AlbumList2).ToNot(BeNil())
		for _, al := range resp.AlbumList2.Album {
			if al.Id == albumID {
				return al.CoverArt
			}
		}
		Fail("album " + albumID + " not found in getAlbumList2")
		return ""
	}

	getCover := func(params ...string) *httptest.ResponseRecorder {
		GinkgoHelper()
		w := httptest.NewRecorder()
		r := buildReq(adminUser, "getCoverArt", params...)
		artRouter.ServeHTTP(w, r)
		return w
	}

	BeforeAll(func() {
		DeferCleanup(configtest.SetupConfig())
		DeferCleanup(func() { Eventually(scanner.IsScanning).Should(BeFalse()) })
		ctx = request.WithUser(GinkgoT().Context(), adminUser)

		musicDir := GinkgoT().TempDir()
		writeArtworkTrack(musicDir, "Artful Artist", "Artful Album", "01 - Come Together", true)
		writeArtworkTrack(musicDir, "Artless Artist", "Artless Album", "01 - Lonely", false)

		conf.Server.MusicFolder = musicDir
		conf.Server.DevExternalScanner = false
		conf.Server.DevEnableMediaFileProbe = false
		conf.Server.CoverArtPriority = "cover.jpg"
		conf.Server.ArtistArtPriority = "artist.png" // offline: artists resolve absent, out of scope here
		conf.Server.EnableMediaFileCoverArt = false
		conf.Server.DevArtworkWorkerConcurrency = 1
		conf.Server.CacheFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.EnableSharing = true
		conf.Server.DevArtworkMaxRequests = 100
		conf.Server.DevArtworkThrottleBacklogLimit = 100
		conf.Server.DevArtworkThrottleBacklogTimeout = time.Minute

		goldenDB.Restore()
		ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
		auth.Init(ds)

		// Re-scanning already-populated data deadlocks the parallel album-refresh phase at the
		// harness's MaxOpenConns=1, so wipe the golden content and import this library fresh.
		wipeScannedContent()
		artLib := model.Library{Name: "Artwork Library", Path: musicDir}
		Expect(ds.Library(ctx).Put(&artLib)).To(Succeed())
		Expect(ds.User(ctx).SetUserLibraries(adminUser.ID, []int{artLib.ID})).To(Succeed())

		s := scanner.New(ctx, ds, events.NoopBroker(),
			playlists.NewPlaylists(ds, artwork.NewUploader(ds)), metrics.NewNoopInstance())
		_, err := s.ScanAll(ctx, true)
		Expect(err).ToNot(HaveOccurred())

		// Focus the worker on the entities under test.
		_, err = db.Db().Exec("DELETE FROM artwork_queue")
		Expect(err).ToNot(HaveOccurred())

		artfulID = albumIDByName("Artful Album")
		artlessID = albumIDByName("Artless Album")

		ph, err := resources.FS().Open(consts.PlaceholderAlbumArt)
		Expect(err).ToNot(HaveOccurred())
		placeholder, err = io.ReadAll(ph)
		Expect(err).ToNot(HaveOccurred())
		_ = ph.Close()

		store := artwork.NewImageStore(GinkgoT().TempDir())
		imgCache := newDummyImageCache(ctx)
		ffm := harness.NoopFFmpeg{}
		artSvc = artwork.NewArtwork(ds, imgCache, store, ffm)
		worker = artwork.NewWorker(ds, store, agents.GetAgents(ds, nil), ffm, events.NoopBroker(), imgCache)

		artRouter = buildArtworkRouter(artSvc)
		router = artRouter // so the shared doReq/doRawReq helpers hit the artwork-wired router
		pubRouter = public.New(ds, artSvc, streamerSpy, core.NewShare(ds), noopArchiver{})
	})

	It("emits a bare optimistic coverArt id before the queue is drained", func() {
		Expect(albumCoverInList(artfulID)).To(Equal("al-" + artfulID))
		Expect(albumCoverInList(artlessID)).To(Equal("al-" + artlessID))
	})

	It("drains the queue: folder art is acquired, the artless album settles absent", func() {
		// Enqueues the way the serving paths do, so the drain is driven by a plain queue row.
		for _, id := range []string{artfulID, artlessID} {
			Expect(ds.ArtworkQueue(ctx).EnqueueBump(model.ArtworkQueueItem{
				ItemKind: model.KindAlbumArtwork.Prefix(), ItemID: id,
				ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBump,
			})).To(Succeed())
		}
		runWorkerUntil(ctx, worker, func() bool {
			found, err := ds.Artwork(ctx).GetItemArtwork(model.KindAlbumArtwork, artfulID, model.ImageTypePrimary)
			if err != nil || found.Hash == "" {
				return false
			}
			absent, err := ds.Artwork(ctx).GetItemArtwork(model.KindAlbumArtwork, artlessID, model.ImageTypePrimary)
			return err == nil && absent.Hash == ""
		})
		ia, err := ds.Artwork(ctx).GetItemArtwork(model.KindAlbumArtwork, artfulID, model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(ia.Source).To(Equal("folder"))
		artfulHash = ia.Hash
		Expect(artfulHash).ToNot(BeEmpty())
	})

	It("promotes the list id to the hash suffix and omits the absent album's coverArt", func() {
		Expect(albumCoverInList(artfulID)).To(Equal("al-" + artfulID + "_" + artfulHash))
		Expect(albumCoverInList(artlessID)).To(BeEmpty())
	})

	It("serves the found album's bytes with revalidation headers for a bare id", func() {
		w := getCover("id", "al-"+artfulID)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(readArtworkFixture("cover.jpg")))
		Expect(w.Header().Get("ETag")).To(Equal(`"` + artfulHash + `"`))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, no-cache"))
	})

	It("serves the found album immutably when the exact hash is requested", func() {
		w := getCover("id", "al-"+artfulID+"_"+artfulHash)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(readArtworkFixture("cover.jpg")))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
	})

	It("returns 304 with an empty body when the client already holds the hash", func() {
		w := httptest.NewRecorder()
		r := buildReq(adminUser, "getCoverArt", "id", "al-"+artfulID)
		r.Header.Set("If-None-Match", `"`+artfulHash+`"`)
		artRouter.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Bytes()).To(BeEmpty())
	})

	It("serves the placeholder with no-store for an absent album", func() {
		w := getCover("id", "al-"+artlessID)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(placeholder))
		Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))
	})

	// Only adminUser was granted this library, so regularUser must get the placeholder even
	// though the artwork state and bytes are perfectly servable by id.
	It("serves the placeholder for an album in a library the caller cannot see", func() {
		w := httptest.NewRecorder()
		artRouter.ServeHTTP(w, buildReq(regularUser, "getCoverArt", "id", "al-"+artfulID))

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(placeholder), "must not leak the real cover")
		Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))

		Expect(getCover("id", "al-"+artfulID).Body.Bytes()).ToNot(Equal(placeholder))
	})

	// An id naming no entity is a different answer from an album with no art: no placeholder.
	It("answers error 70 for an id that matches no entity", func() {
		w := getCover("id", "al-nosuchalbum")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(ContainSubstring(`"code":70`))
		Expect(w.Body.Bytes()).ToNot(Equal(placeholder))
	})

	It("serves /share/img immutably for a JWT whose payload carries the hash", func() {
		token, err := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour),
			auth.Claims{ID: "al-" + artfulID + "_" + artfulHash})
		Expect(err).ToNot(HaveOccurred())

		w := httptest.NewRecorder()
		pubRouter.ServeHTTP(w, httptest.NewRequest("GET", "/img/"+token, nil))
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(readArtworkFixture("cover.jpg")))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
	})

	It("returns 404 from /share/img for an absent entity", func() {
		token, err := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour),
			auth.Claims{ID: "al-" + artlessID})
		Expect(err).ToNot(HaveOccurred())

		w := httptest.NewRecorder()
		pubRouter.ServeHTTP(w, httptest.NewRequest("GET", "/img/"+token, nil))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

// buildArtworkRouter mirrors setupTestDB's Subsonic wiring but with the real artwork.Artwork.
func buildArtworkRouter(art artwork.Artwork) *subsonic.Router {
	decider := stream.NewTranscodeDecider(ds, harness.NoopFFmpeg{})
	s := scanner.New(ctx, ds, events.NoopBroker(),
		playlists.NewPlaylists(ds, artwork.NewUploader(ds)), metrics.NewNoopInstance())
	return subsonic.New(
		ds, art, streamerSpy, noopArchiver{}, core.NewPlayers(ds), noopProvider{}, s,
		events.NoopBroker(), playlists.NewPlaylists(ds, artwork.NewUploader(ds)),
		scrobbler.NewPlayTracker(ds, events.NoopBroker(), nil), core.NewShare(ds),
		playback.PlaybackServer(nil), metrics.NewNoopInstance(), lyrics.NewLyrics(ds, nil), decider, nil,
	)
}

// wipeScannedContent clears all scanned content so the next scan is a clean import. FKs are
// disabled for the bulk delete, mirroring harness.Restore.
func wipeScannedContent() {
	GinkgoHelper()
	_, err := db.Db().Exec("PRAGMA foreign_keys = OFF")
	Expect(err).ToNot(HaveOccurred())
	for _, t := range []string{"media_file", "album", "artist", "folder", "item_artwork", "artwork_queue", "artwork", "library"} {
		_, err = db.Db().Exec(`DELETE FROM "` + t + `"`)
		Expect(err).ToNot(HaveOccurred(), "wiping %s", t)
	}
	_, err = db.Db().Exec("PRAGMA foreign_keys = ON")
	Expect(err).ToNot(HaveOccurred())
}

func albumIDByName(name string) string {
	GinkgoHelper()
	albums, err := ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album.name": name}})
	Expect(err).ToNot(HaveOccurred())
	Expect(albums).To(HaveLen(1), "expected exactly one album named %q", name)
	return albums[0].ID
}

// Distinct ALBUM/ALBUMARTIST tags keep the generated albums from merging into one.
func writeArtworkTrack(root, artist, album, title string, withCover bool) {
	GinkgoHelper()
	dir := filepath.Join(root, artist, album)
	Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
	mp3 := filepath.Join(dir, title+".mp3")
	Expect(os.WriteFile(mp3, readArtworkFixture("test.mp3"), 0o600)).To(Succeed())
	Expect(taglib.WriteTags(mp3, map[string][]string{
		"ALBUM": {album}, "ALBUMARTIST": {artist}, "ARTIST": {artist}, "TITLE": {title},
	}, taglib.Clear)).To(Succeed())
	if withCover {
		Expect(os.WriteFile(filepath.Join(dir, "cover.jpg"), readArtworkFixture("cover.jpg"), 0o600)).To(Succeed())
	}
}

func readArtworkFixture(name string) []byte {
	GinkgoHelper()
	data, err := os.ReadFile(filepath.Join("tests", "fixtures", "artist", "an-album", name))
	Expect(err).ToNot(HaveOccurred())
	return data
}

// size=0 requests stream originals and never invoke the reader, so this resize cache only has
// to satisfy the constructor.
func newDummyImageCache(ctx context.Context) cache.FileCache {
	GinkgoHelper()
	c := cache.NewFileCache("SubsonicArtworkE2E", "100MB", "images", 0,
		func(context.Context, cache.Item) (io.Reader, error) {
			return nil, errors.New("resize not exercised in subsonic artwork e2e")
		})
	Eventually(func() bool { return c.Available(ctx) }).Should(BeTrue())
	return c
}

func runWorkerUntil(ctx context.Context, worker *artwork.Worker, until func() bool) {
	GinkgoHelper()
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- worker.Run(runCtx) }()
	Eventually(until, 10*time.Second, 20*time.Millisecond).Should(BeTrue())
	cancel()
	Eventually(done, 2*time.Second).Should(Receive(BeNil()))
}
