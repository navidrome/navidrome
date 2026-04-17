package artworke2e_test

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.senan.xyz/taglib"
)

// realMP3WithEmbeddedArt is the bytes of the canonical test fixture that
// contains a valid MP3 stream with an embedded picture. Used in the
// embedded-art e2e scenarios where FakeFS's JSON-encoded tag data isn't
// readable by taglib. Swap this into fakeFS.MapFS *after* scanning so the
// scanner still populates EmbedArtPath via the JSON-tagged track, and the
// artwork reader gets real bytes when it calls libFS.Open.
//
//go:embed testdata/embedded_art.mp3
var realMP3WithEmbeddedArt []byte

// embeddedArtBytes is the exact image payload that the artwork reader will
// extract from realMP3WithEmbeddedArt. Computed once via taglib so tests can
// assert byte-for-byte equality — if this ever differs it means the reader
// pulled from a different source.
var embeddedArtBytes = extractEmbeddedArt(realMP3WithEmbeddedArt)

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

// replaceWithRealMP3 swaps the FakeFS entry at the given library-relative
// path so libFS.Open returns an MP3 stream taglib can parse.
func replaceWithRealMP3(relPath string) {
	GinkgoHelper()
	fakeFS.MapFS[relPath] = &fstest.MapFile{Data: realMP3WithEmbeddedArt}
}

// placeholderBytes returns the bundled album-placeholder image bytes — the
// same stream the artwork reader emits when every source falls through.
func placeholderBytes() []byte {
	GinkgoHelper()
	r, err := resources.FS().Open(consts.PlaceholderAlbumArt)
	Expect(err).ToNot(HaveOccurred())
	defer r.Close()
	data, err := io.ReadAll(r)
	Expect(err).ToNot(HaveOccurred())
	return data
}

// writeUploadedImage drops `filename` into <DataFolder>/artwork/<entity>/ with
// the given bytes, matching the on-disk layout expected by
// model.UploadedImagePath.
func writeUploadedImage(entity, filename string, data []byte) {
	GinkgoHelper()
	dir := filepath.Dir(model.UploadedImagePath(entity, filename))
	Expect(os.MkdirAll(dir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(dir, filename), data, 0600)).To(Succeed())
}

func newNoopFFmpeg() *tests.MockFFmpeg {
	ff := tests.NewMockFFmpeg("")
	ff.Error = errors.New("noop")
	return ff
}

// trackFile builds a FakeFS MP3 entry with optional tag overrides.
func trackFile(num int, title string, extra ...map[string]any) *fstest.MapFile {
	tags := storagetest.Track(num, title)
	for _, e := range extra {
		maps.Copy(tags, e)
	}
	return storagetest.MP3(tags)
}

// imageFile builds a label-keyed image entry. The bytes are deterministic
// per-label so tests can assert which file won.
func imageFile(label string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("image:" + label)}
}

// realPNG builds a minimal 2x2 PNG with a color derived from label. Needed by
// tests that feed the bytes into image.Decode (e.g. playlist tiled covers).
func realPNG(label string) *fstest.MapFile {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	// Derive a deterministic color per label.
	h := fnv.New32a()
	_, _ = h.Write([]byte(label))
	sum := h.Sum32()
	c := color.RGBA{R: byte(sum), G: byte(sum >> 8), B: byte(sum >> 16), A: 255}
	for y := range 2 {
		for x := range 2 {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	Expect(png.Encode(&buf, img)).To(Succeed())
	return &fstest.MapFile{Data: buf.Bytes()}
}

// imageBytes returns the bytes that imageFile(label) writes.
func imageBytes(label string) []byte { return imageFile(label).Data }

// setLayout populates fakeFS with the given map. Call after setupHarness.
// All paths must be forward-slash and relative (no leading "/").
func setLayout(files fstest.MapFS) {
	GinkgoHelper()
	fakeFS.SetFiles(files)
}

func readArtwork(artID model.ArtworkID) []byte {
	GinkgoHelper()
	r, _, err := aw.Get(ctx, artID, 0, false)
	Expect(err).ToNot(HaveOccurred())
	defer r.Close()
	b, err := io.ReadAll(r)
	Expect(err).ToNot(HaveOccurred())
	return b
}

func readArtworkOrErr(artID model.ArtworkID) ([]byte, error) {
	r, _, err := aw.Get(ctx, artID, 0, false)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// noopProvider implements external.Provider with not-found returns so the
// "external" priority entry never produces a result.
type noopProvider struct{}

func (n *noopProvider) UpdateAlbumInfo(context.Context, string) (*model.Album, error) {
	return nil, model.ErrNotFound
}
func (n *noopProvider) UpdateArtistInfo(context.Context, string, int, bool) (*model.Artist, error) {
	return nil, model.ErrNotFound
}
func (n *noopProvider) SimilarSongs(context.Context, string, int) (model.MediaFiles, error) {
	return nil, nil
}
func (n *noopProvider) TopSongs(context.Context, string, int) (model.MediaFiles, error) {
	return nil, nil
}
func (n *noopProvider) ArtistImage(context.Context, string) (*url.URL, error) {
	return nil, model.ErrNotFound
}
func (n *noopProvider) AlbumImage(context.Context, string) (*url.URL, error) {
	return nil, model.ErrNotFound
}

var _ external.Provider = (*noopProvider)(nil)
