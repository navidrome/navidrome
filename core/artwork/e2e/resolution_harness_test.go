package e2e

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing/fstest"
	"time"

	_ "github.com/navidrome/navidrome/adapters/gotaglib"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/tests/harness"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.senan.xyz/taglib"
)

const fakeLibScheme = "artworkfake"
const fakeLibPath = fakeLibScheme + ":///music"

const (
	defaultCoverPriority = "cover.*, folder.*, front.*, embedded, external"
	defaultDiscPriority  = "disc*.*, cd*.*, cover.*, folder.*, front.*, discsubtitle, embedded"
)

var (
	rctx    context.Context
	rds     *tests.MockDataStore
	rstore  *artwork.ImageStore
	rsvc    artwork.Artwork
	rworker *artwork.Worker
	fakeFS  *storagetest.FakeFS
)

// The go-sqlite3 singleton holds the file open for the whole suite, and Windows cannot unlink a
// file with a live handle, so the DB cannot live in Ginkgo's per-spec TempDir.
var suiteDBTempDir string

// Migrating the schema costs ~400ms, so it runs once per suite and specs reset by truncating.
var userTables []string

var _ = BeforeSuite(func() {
	suiteDBTempDir = GinkgoT().TempDir()

	DeferCleanup(configtest.SetupConfig())
	conf.Server.DbPath = filepath.Join(suiteDBTempDir, "artwork-resolution-e2e.db") + "?_journal_mode=WAL"
	conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
	db.Db().SetMaxOpenConns(1)
	db.Init(request.WithUser(context.Background(), model.User{ID: "admin-1", IsAdmin: true}))

	userTables = harness.ResettableTables()
})

var _ = AfterSuite(func() {
	db.Close(context.Background())
})

func setupResolutionHarness() {
	DeferCleanup(configtest.SetupConfig())

	tempDir := GinkgoT().TempDir()
	conf.Server.DbPath = filepath.Join(suiteDBTempDir, "artwork-resolution-e2e.db") + "?_journal_mode=WAL"
	conf.Server.DataFolder = conf.NewDir(tempDir)
	conf.Server.MusicFolder = fakeLibPath
	conf.Server.DevExternalScanner = false
	conf.Server.ImageCacheSize = "0"
	conf.Server.EnableExternalServices = false
	conf.Server.EnableMediaFileCoverArt = true
	conf.Server.DevArtworkWorkerConcurrency = 1

	rctx = request.WithUser(GinkgoT().Context(), model.User{ID: "admin-1", UserName: "admin", IsAdmin: true})
	harness.TruncateDB(userTables)

	rds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}

	adminUser := model.User{ID: "admin-1", UserName: "admin", Name: "Admin", IsAdmin: true, NewPassword: "password"}
	Expect(rds.User(rctx).Put(&adminUser)).To(Succeed())

	lib := model.Library{ID: 1, Name: "Music", Path: fakeLibPath}
	Expect(rds.Library(rctx).Put(&lib)).To(Succeed())
	Expect(rds.User(rctx).SetUserLibraries(adminUser.ID, []int{lib.ID})).To(Succeed())

	loadEmbeddedFixture()

	fakeFS = &storagetest.FakeFS{}
	storagetest.Register(fakeLibScheme, fakeFS)

	ffm := tests.NewMockFFmpeg("")
	rstore = artwork.NewImageStore(filepath.Join(tempDir, consts.HashedArtworkFolder))
	// size=0 requests stream originals, so this reader is never called (serving_test covers resizing).
	imgCache := cache.NewFileCache("ArtworkResolutionE2E", "100MB", "images", 0,
		func(context.Context, cache.Item) (io.Reader, error) {
			return nil, fmt.Errorf("resize not exercised in e2e")
		})
	Eventually(func() bool { return imgCache.Available(rctx) }).Should(BeTrue())

	rsvc = artwork.NewArtwork(rds, imgCache, rstore, ffm)
	rworker = artwork.NewWorker(rds, rstore, agents.GetAgents(rds, nil), ffm, events.NoopBroker(), imgCache)
}

// setLayout paths must be relative and forward-slash.
func setLayout(files fstest.MapFS) {
	GinkgoHelper()
	fakeFS.SetFiles(files)
}

func scan() {
	GinkgoHelper()
	s := scanner.New(rctx, rds, events.NoopBroker(),
		playlists.NewPlaylists(rds, artwork.NewUploader(rds)), metrics.NewNoopInstance())
	_, err := s.ScanAll(rctx, true)
	Expect(err).ToNot(HaveOccurred())
}

func acquire(kind model.Kind, id string) model.ItemArtwork {
	GinkgoHelper()
	// Enqueues the way the serving paths do, so the drain is driven by a plain queue row.
	Expect(rds.ArtworkQueue(rctx).EnqueueBump(model.ArtworkQueueItem{
		ItemKind: kind.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary,
		Priority: model.ArtworkPriorityBump,
	})).To(Succeed())
	var ia *model.ItemArtwork
	runResolutionWorkerUntil(func() bool {
		got, err := rds.Artwork(rctx).GetItemArtwork(kind, id, model.ImageTypePrimary)
		if err != nil {
			return false
		}
		ia = got
		return true
	})
	return *ia
}

func runResolutionWorkerUntil(until func() bool) {
	GinkgoHelper()
	runCtx, cancel := context.WithCancel(rctx)
	done := make(chan error, 1)
	go func() { done <- rworker.Run(runCtx) }()
	Eventually(until, 5*time.Second, 10*time.Millisecond).Should(BeTrue())
	cancel()
	Eventually(done, 2*time.Second).Should(Receive(BeNil()))
}

func serveBytes(artID model.ArtworkID) []byte {
	GinkgoHelper()
	img, err := rsvc.Get(rctx, artID, 0, false)
	Expect(err).ToNot(HaveOccurred())
	defer img.Close()
	data, err := io.ReadAll(img)
	Expect(err).ToNot(HaveOccurred())
	return data
}

func serveErr(artID model.ArtworkID) error {
	img, err := rsvc.Get(rctx, artID, 0, false)
	if img != nil {
		img.Close()
	}
	return err
}

func libFileBytes(suffix string) []byte {
	GinkgoHelper()
	var match string
	for name := range fakeFS.MapFS {
		if strings.HasSuffix(name, suffix) {
			Expect(match).To(BeEmpty(), "suffix %q is ambiguous: %q and %q", suffix, match, name)
			match = name
		}
	}
	Expect(match).ToNot(BeEmpty(), "no library file ends with %q", suffix)
	return fakeFS.MapFS[match].Data
}

// Serving before acquiring is deliberate: with no state row the request resolves through the
// library FS, while a settled folder row is read with os.Open, which the in-memory FS cannot serve.
func expectAlbumFolderCover(al model.Album, suffix string) {
	GinkgoHelper()
	requireNoStateRow(model.KindAlbumArtwork, al.ID)
	Expect(serveBytes(al.CoverArtID())).To(Equal(libFileBytes(suffix)))
	ia := acquire(model.KindAlbumArtwork, al.ID)
	Expect(ia.Source).To(Equal("folder"))
	Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix(suffix))
}

// A drain settles every ready item, so byte-level folder assertions must precede any acquire.
func requireNoStateRow(kind model.Kind, id string) {
	GinkgoHelper()
	_, err := rds.Artwork(rctx).GetItemArtwork(kind, id, model.ImageTypePrimary)
	Expect(err).To(MatchError(model.ErrNotFound),
		"assert %s %q before acquiring any other entity in this spec", kind, id)
}

func expectAlbumAbsent(al model.Album) {
	GinkgoHelper()
	ia := acquire(model.KindAlbumArtwork, al.ID)
	Expect(ia.Hash).To(BeEmpty())
	Expect(serveErr(al.CoverArtID())).To(MatchError(artwork.ErrUnavailable))
}

func expectArtistFolder(ar model.Artist, suffix string) {
	GinkgoHelper()
	requireNoStateRow(model.KindArtistArtwork, ar.ID)
	Expect(serveBytes(model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil))).To(Equal(libFileBytes(suffix)))
	ia := acquire(model.KindArtistArtwork, ar.ID)
	Expect(ia.Source).To(Equal("folder"))
	Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix(suffix))
}

func writeUploadedImage(entity, filename string, data []byte) {
	GinkgoHelper()
	dst := model.UploadedImagePath(entity, filename)
	Expect(os.MkdirAll(filepath.Dir(dst), 0o755)).To(Succeed())
	Expect(os.WriteFile(dst, data, 0o600)).To(Succeed())
}

func discArtID(al model.Album, disc int) model.ArtworkID {
	return model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, disc), &al.UpdatedAt)
}

// Disc art is a pure serve-time read through the library FS: no worker, no state row.
func expectDiscImage(al model.Album, disc int, label string) {
	GinkgoHelper()
	Expect(serveBytes(discArtID(al, disc))).To(Equal(pngBytes(label)))
}

// Samples in rect() order: top-left, top-right, bottom-left, bottom-right.
func gridQuadrants(data []byte) [4]color.RGBA {
	GinkgoHelper()
	img, _, err := image.Decode(bytes.NewReader(data))
	Expect(err).ToNot(HaveOccurred())
	b := img.Bounds()
	qw, qh := b.Dx()/4, b.Dy()/4
	at := func(x, y int) color.RGBA {
		c := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y))
		return c.(color.RGBA)
	}
	return [4]color.RGBA{at(qw, qh), at(3*qw, qh), at(qw, 3*qh), at(3*qw, 3*qh)}
}

// Store-backed sources only (embedded/generated); file-backed ones assert on ia.SourcePath.
func storedBytes(ia model.ItemArtwork) []byte {
	GinkgoHelper()
	art, err := rds.Artwork(rctx).GetImage(ia.Hash)
	Expect(err).ToNot(HaveOccurred())
	r, err := rstore.Open(ia.Hash, art.Mime)
	Expect(err).ToNot(HaveOccurred())
	defer r.Close()
	data, err := io.ReadAll(r)
	Expect(err).ToNot(HaveOccurred())
	return data
}

// The pixel color derives from label, so each label yields distinct, still-decodable bytes.
func smallPNG(label string) *fstest.MapFile {
	h := fnv.New32a()
	_, _ = h.Write([]byte(label))
	sum := h.Sum32()
	c := color.RGBA{R: byte(sum), G: byte(sum >> 8), B: byte(sum >> 16), A: 255}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := range 2 {
		for x := range 2 {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	Expect(png.Encode(&buf, img)).To(Succeed())
	return &fstest.MapFile{Data: buf.Bytes()}
}

func pngBytes(label string) []byte {
	GinkgoHelper()
	return smallPNG(label).Data
}

func trackFile(num int, title string, extra ...map[string]any) *fstest.MapFile {
	tags := storagetest.Track(num, title)
	for _, e := range extra {
		maps.Copy(tags, e)
	}
	return storagetest.MP3(tags)
}

// FakeFS's JSON-encoded tags aren't taglib-readable, so embedded-art specs swap in these real MP3
// bytes after scanning. Loaded lazily: tests.Init must chdir to the project root first.
var (
	embeddedFixtureOnce sync.Once
	embeddedArtFixture  []byte
	embeddedArtBytes    []byte
)

func loadEmbeddedFixture() {
	embeddedFixtureOnce.Do(func() {
		embeddedArtFixture = readFixture(mp3Fixture)
		embeddedArtBytes = extractEmbeddedArt(embeddedArtFixture)
	})
}

func extractEmbeddedArt(mp3 []byte) []byte {
	tf, err := taglib.OpenStream(bytes.NewReader(mp3))
	if err != nil {
		panic("embedded-art fixture: taglib.OpenStream failed: " + err.Error())
	}
	defer tf.Close()
	images := tf.Properties().Images
	if len(images) == 0 {
		panic("embedded-art fixture has no embedded images")
	}
	data, err := tf.Image(0)
	if err != nil || len(data) == 0 {
		panic("embedded-art fixture: could not read image 0")
	}
	return data
}

func replaceWithRealMP3(relPath string) {
	GinkgoHelper()
	fakeFS.MapFS[relPath] = &fstest.MapFile{Data: embeddedArtFixture}
}

func firstAlbum() model.Album {
	GinkgoHelper()
	albums, err := rds.Album(rctx).GetAll(model.QueryOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(albums).To(HaveLen(1), "expected exactly one album, got %d", len(albums))
	return albums[0]
}

func albumByName(name string) model.Album {
	GinkgoHelper()
	albums, err := rds.Album(rctx).GetAll(model.QueryOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, al := range albums {
		if al.Name == name {
			return al
		}
	}
	Fail(fmt.Sprintf("album %q not found among %d albums", name, len(albums)))
	return model.Album{}
}
