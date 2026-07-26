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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing/fstest"
	"time"

	_ "github.com/navidrome/navidrome/adapters/gotaglib"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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

// This harness restores the pre-cutover artwork resolution edge-case coverage, but drives it
// through the real pipeline: a real scanner populates the folder graph from an in-memory library,
// the real Worker drains the queue to resolve/persist state, and the real Service serves it.
// It documents the folder-selection rules (album/disc/artist priority, #5376/#5456/#5451/#5457)
// that the lightweight acquire_serve_test.go intentionally leaves to this suite.

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
	rsvc    artwork.Service
	rworker *artwork.Worker
	fakeFS  *storagetest.FakeFS
)

// The DB file lives in a suite-level tempdir: the go-sqlite3 singleton keeps the file open for the
// whole suite, and Ginkgo's per-spec TempDir cleanup can't unlink a file with a live handle on
// Windows. A suite-level tempdir plus an AfterSuite close avoids the lock conflict.
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
	conf.Server.ArtworkWorkerConcurrency = 1

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
	rstore = artwork.NewImageStore(filepath.Join(tempDir, "store"))
	// size=0 requests stream originals and never touch the resize cache, so this reader is a
	// compile-time stand-in only; the resize path is covered by the package's serving_test.
	imgCache := cache.NewFileCache("ArtworkResolutionE2E", "100MB", "images", 0,
		func(context.Context, cache.Item) (io.Reader, error) {
			return nil, fmt.Errorf("resize not exercised in e2e")
		})
	Eventually(func() bool { return imgCache.Available(rctx) }).Should(BeTrue())

	rsvc = artwork.NewService(rds, imgCache, rstore, ffm)
	rworker = artwork.NewWorker(rds, rstore, agents.GetAgents(rds, nil), ffm, events.NoopBroker(), imgCache)
}

// setLayout populates the fake library. All paths must be forward-slash and relative.
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

// acquire drives the worker to resolve one entity and returns its persisted state row. It fails if
// the worker never settles (found or absent) within the timeout.
func acquire(kind model.Kind, id string) model.ItemArtwork {
	GinkgoHelper()
	rworker.Bump(kind.Prefix(), id)
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

// serveBytes reads an artwork ID through the real Service at full size and returns its bytes.
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

// libFileBytes returns the contents of the one library file whose path ends with suffix.
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

// expectAlbumFolderCover asserts the album resolves to the library image at the given path suffix,
// byte-for-byte. The serve happens before acquisition on purpose: with no state row the request
// path resolves locally through the library FS, whereas a settled folder row is file-backed and
// read with os.Open, which the in-memory FS cannot satisfy.
func expectAlbumFolderCover(al model.Album, suffix string) {
	GinkgoHelper()
	requireNoStateRow(model.KindAlbumArtwork, al.ID)
	Expect(serveBytes(al.CoverArtID())).To(Equal(libFileBytes(suffix)))
	ia := acquire(model.KindAlbumArtwork, al.ID)
	Expect(ia.Source).To(Equal("folder"))
	Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix(suffix))
}

// requireNoStateRow guards the byte-level folder assertions: a drain resolves every ready queue
// row, so acquiring one entity can settle others. Once settled, folder art is file-backed and the
// in-memory FS cannot serve it — so these assertions must come before any acquire in a spec.
func requireNoStateRow(kind model.Kind, id string) {
	GinkgoHelper()
	_, err := rds.Artwork(rctx).GetItemArtwork(kind, id, model.ImageTypePrimary)
	Expect(err).To(MatchError(model.ErrNotFound),
		"assert %s %q before acquiring any other entity in this spec", kind, id)
}

// expectAlbumAbsent asserts the album settled absent (no source resolved) and serves unavailable.
func expectAlbumAbsent(al model.Album) {
	GinkgoHelper()
	ia := acquire(model.KindAlbumArtwork, al.ID)
	Expect(ia.Hash).To(BeEmpty())
	Expect(serveErr(al.CoverArtID())).To(MatchError(artwork.ErrUnavailable))
}

// expectArtistFolder asserts the worker selected a library folder image (artist.*, album/artist.*)
// as the artist image; like album folder art it is file-backed, so it is asserted on the state row.
func expectArtistFolder(ar model.Artist, suffix string) {
	GinkgoHelper()
	requireNoStateRow(model.KindArtistArtwork, ar.ID)
	Expect(serveBytes(model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil))).To(Equal(libFileBytes(suffix)))
	ia := acquire(model.KindArtistArtwork, ar.ID)
	Expect(ia.Source).To(Equal("folder"))
	Expect(filepath.ToSlash(ia.SourcePath)).To(HaveSuffix(suffix))
}

// writeUploadedImage drops raw bytes into the per-entity upload folder under DataFolder, matching
// the layout model.UploadedImagePath expects. Uploads are real files on disk, so they serve back.
func writeUploadedImage(entity, filename string, data []byte) {
	GinkgoHelper()
	dst := model.UploadedImagePath(entity, filename)
	Expect(os.MkdirAll(filepath.Dir(dst), 0o755)).To(Succeed())
	Expect(os.WriteFile(dst, data, 0o600)).To(Succeed())
}

// discArtID is the artwork ID for one disc of an album.
func discArtID(al model.Album, disc int) model.ArtworkID {
	return model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, disc), &al.UpdatedAt)
}

// expectDiscImage asserts a multi-disc album serves the given disc's art byte-for-byte. Disc art
// is a pure serve-time read through the library FS (no worker/state row), so this serves it live.
func expectDiscImage(al model.Album, disc int, label string) {
	GinkgoHelper()
	Expect(serveBytes(discArtID(al, disc))).To(Equal(pngBytes(label)))
}

// gridQuadrants decodes a generated 2x2 playlist cover and samples the center of each quadrant,
// in rect() order: top-left, top-right, bottom-left, bottom-right. Each tile is a solid color, so
// the samples identify which album art landed where (and whether tiles were mirrored).
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

// storedBytes returns the bytes the worker placed in the content-addressed store for a
// store-backed resolution (embedded/generated). Folder/upload sources are file-backed and are
// not in the store; assert those on ia.SourcePath instead.
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

// smallPNG builds a tiny valid PNG whose pixel color is derived from label, so the bytes are
// distinct per label (a resolver picking a different file yields a different hash/path) while
// still decoding cleanly for the worker's blurhash step.
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

// pngBytes returns the bytes smallPNG(label) writes, for byte-for-byte serve assertions.
func pngBytes(label string) []byte {
	GinkgoHelper()
	return smallPNG(label).Data
}

// trackFile builds a fake MP3 entry with optional tag overrides (album, disc, discsubtitle, ...).
func trackFile(num int, title string, extra ...map[string]any) *fstest.MapFile {
	tags := storagetest.Track(num, title)
	for _, e := range extra {
		for k, v := range e {
			tags[k] = v
		}
	}
	return storagetest.MP3(tags)
}

// embeddedArtFixture is a real MP3 with an embedded picture; FakeFS's JSON-encoded tags aren't
// taglib-readable, so embedded-art scenarios swap these bytes in after scanning. embeddedArtBytes
// is the exact image taglib extracts from it. Both load lazily (after tests.Init chdirs to the
// project root and registers Gomega), via loadEmbeddedFixture from setupResolutionHarness.
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

// replaceWithRealMP3 swaps the fake entry at relPath for the real embedded-art MP3, so the
// library FS returns a taglib-parseable stream during resolution.
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
