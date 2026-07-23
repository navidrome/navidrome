// Package e2e exercises the artwork pipeline end to end: it enqueues real entities, drives the
// real Worker to drain the queue, and serves the result through the real Service, over a real
// ImageStore and real library files.
package e2e

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/navidrome/navidrome/adapters/gotaglib" // registers the "taglib" local-storage extractor
	"github.com/navidrome/navidrome/core/artwork"
	_ "github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtworkE2E(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artwork Pipeline E2E Suite")
}

// Fixtures relative to the project root (tests.Init chdirs there).
const (
	coverFixture     = "tests/fixtures/artist/an-album/cover.jpg"
	mp3Fixture       = "tests/fixtures/artist/an-album/test.mp3"
	artistPngFixture = "tests/fixtures/artist/an-album/artist.png"
	albumFolderPath  = "tests/fixtures/artist/an-album"
)

// readFixture returns the raw bytes of a project-relative fixture file.
func readFixture(rel string) []byte {
	GinkgoHelper()
	data, err := os.ReadFile(rel)
	Expect(err).ToNot(HaveOccurred(), "reading fixture %q", rel)
	return data
}

// readAll drains an artwork image to bytes and closes it.
func readAll(img *artwork.Image) []byte {
	GinkgoHelper()
	Expect(img).ToNot(BeNil())
	defer img.Close()
	data, err := io.ReadAll(img)
	Expect(err).ToNot(HaveOccurred())
	return data
}

// runWorkerUntil starts the real worker loop, waits for a condition, then cancels and joins it,
// mirroring how cmd drives Worker.Run in production.
func runWorkerUntil(ctx context.Context, worker *artwork.Worker, until func() bool) {
	GinkgoHelper()
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- worker.Run(runCtx) }()
	Eventually(until, 5*time.Second, 10*time.Millisecond).Should(BeTrue())
	cancel()
	Eventually(done, 2*time.Second).Should(Receive(BeNil()))
}

// fakeFolderRepo is the minimal FolderRepository the album/playlist resolution chains touch:
// GetAll yields the seeded folders and the album-root parent lookup finds nothing.
type fakeFolderRepo struct {
	model.FolderRepository
	result []model.Folder
}

func (f *fakeFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) { return f.result, nil }

func (f *fakeFolderRepo) HasAudioOutsideFolders(model.Folder, []string) (bool, error) {
	return false, nil
}

func (f *fakeFolderRepo) Get(string) (*model.Folder, error) { return nil, model.ErrNotFound }

// writeUpload copies a fixture into the per-entity upload folder under the data dir and returns
// the bare filename UploadedImagePath expects.
func writeUpload(entityType, name, srcFixture string) string {
	GinkgoHelper()
	dst := model.UploadedImagePath(entityType, name)
	Expect(os.MkdirAll(filepath.Dir(dst), 0o755)).To(Succeed())
	Expect(os.WriteFile(dst, readFixture(srcFixture), 0o600)).To(Succeed())
	return name
}
