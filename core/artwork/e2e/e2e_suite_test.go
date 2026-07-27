// Package e2e exercises the artwork pipeline end to end: the real Worker drains the queue and the
// real Service serves the result, over a real ImageStore and real library files.
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

func readFixture(rel string) []byte {
	GinkgoHelper()
	data, err := os.ReadFile(rel)
	Expect(err).ToNot(HaveOccurred(), "reading fixture %q", rel)
	return data
}

func readAll(img *artwork.Image) []byte {
	GinkgoHelper()
	Expect(img).ToNot(BeNil())
	defer img.Close()
	data, err := io.ReadAll(img)
	Expect(err).ToNot(HaveOccurred())
	return data
}

func runWorkerUntil(ctx context.Context, worker *artwork.Worker, until func() bool) {
	GinkgoHelper()
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- worker.Run(runCtx) }()
	Eventually(until, 5*time.Second, 10*time.Millisecond).Should(BeTrue())
	cancel()
	Eventually(done, 2*time.Second).Should(Receive(BeNil()))
}

type fakeFolderRepo struct {
	model.FolderRepository
	result []model.Folder
}

func (f *fakeFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) { return f.result, nil }

func (f *fakeFolderRepo) HasAudioOutsideFolders(model.Folder, []string) (bool, error) {
	return false, nil
}

func (f *fakeFolderRepo) Get(string) (*model.Folder, error) { return nil, model.ErrNotFound }

func writeUpload(entityType, name, srcFixture string) string {
	GinkgoHelper()
	dst := model.UploadedImagePath(entityType, name)
	Expect(os.MkdirAll(filepath.Dir(dst), 0o755)).To(Succeed())
	Expect(os.WriteFile(dst, readFixture(srcFixture), 0o600)).To(Succeed())
	return name
}
