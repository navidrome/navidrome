package artworke2e_test

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/url"
	"testing/fstest"

	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

// imageBytes returns the bytes that imageFile(label) writes.
func imageBytes(label string) []byte { return []byte("image:" + label) }

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

// noopFFmpeg satisfies ffmpeg.FFmpeg and always returns an error.
type noopFFmpeg struct{}

func (n *noopFFmpeg) Transcode(context.Context, ffmpeg.TranscodeOptions) (io.ReadCloser, error) {
	return nil, errors.New("noop")
}
func (n *noopFFmpeg) ExtractImage(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("noop")
}
func (n *noopFFmpeg) Probe(context.Context, []string) (string, error) { return "", nil }
func (n *noopFFmpeg) ProbeAudioStream(context.Context, string) (*ffmpeg.AudioProbeResult, error) {
	return nil, errors.New("noop")
}
func (n *noopFFmpeg) ConvertAnimatedImage(context.Context, io.Reader, int, int) (io.ReadCloser, error) {
	return nil, errors.New("noop")
}
func (n *noopFFmpeg) CmdPath() (string, error) { return "", nil }
func (n *noopFFmpeg) IsAvailable() bool        { return false }
func (n *noopFFmpeg) IsProbeAvailable() bool   { return false }
func (n *noopFFmpeg) Version() string          { return "noop" }

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

var (
	_ ffmpeg.FFmpeg     = (*noopFFmpeg)(nil)
	_ external.Provider = (*noopProvider)(nil)
)
