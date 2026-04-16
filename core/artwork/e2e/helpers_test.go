package artworke2e_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	nakedMP3Path    = "tests/fixtures/test.mp3"
	embeddedMP3Path = "tests/fixtures/artist/an-album/test.mp3"
)

var (
	nakedMP3Once     sync.Once
	nakedMP3Bytes    []byte
	embeddedMP3Once  sync.Once
	embeddedMP3Bytes []byte
	jpegCache        sync.Map // label -> []byte
)

func loadFixture(path string, once *sync.Once, dst *[]byte) []byte {
	once.Do(func() {
		b, err := os.ReadFile(path)
		Expect(err).ToNot(HaveOccurred(), "reading fixture %s", path)
		*dst = b
	})
	return *dst
}

type layoutFile struct {
	relPath string
	source  []byte
}

func mp3Naked(relPath string) layoutFile {
	return layoutFile{relPath: relPath, source: loadFixture(nakedMP3Path, &nakedMP3Once, &nakedMP3Bytes)}
}

func mp3Embedded(relPath string) layoutFile {
	return layoutFile{relPath: relPath, source: loadFixture(embeddedMP3Path, &embeddedMP3Once, &embeddedMP3Bytes)}
}

func imgJPEG(relPath, label string) layoutFile {
	return layoutFile{relPath: relPath, source: jpegBytes(label)}
}

func writeLayout(files ...layoutFile) {
	GinkgoHelper()
	for _, f := range files {
		full := filepath.Join(musicDir, f.relPath)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, f.source, 0o600)).To(Succeed())
	}
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

func jpegLabel(label string) []byte { return jpegBytes(label) }

func jpegBytes(label string) []byte {
	if v, ok := jpegCache.Load(label); ok {
		return v.([]byte)
	}
	img := solidImage(label)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	b := buf.Bytes()
	actual, _ := jpegCache.LoadOrStore(label, b)
	return actual.([]byte)
}

func solidImage(label string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	c := colorFromLabel(label)
	for y := range 2 {
		for x := range 2 {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// colorFromLabel maps any string to a unique-enough RGBA via FNV mix so distinct
// labels produce distinct bytes after JPEG encoding.
func colorFromLabel(label string) color.RGBA {
	var h uint32 = 2166136261
	for i := range len(label) {
		h ^= uint32(label[i])
		h *= 16777619
	}
	return color.RGBA{R: uint8(h >> 16), G: uint8(h >> 8), B: uint8(h), A: 255}
}

func newNoopFFmpeg() *tests.MockFFmpeg {
	ff := tests.NewMockFFmpeg("")
	ff.Error = errors.New("noop")
	return ff
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
