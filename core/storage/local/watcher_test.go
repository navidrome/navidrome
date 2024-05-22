package local_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/core/storage/local"
	_ "github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/model/metadata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Watcher", func() {
	var lsw storage.Watcher
	var tmpFolder string

	BeforeEach(func() {
		tmpFolder = GinkgoT().TempDir()

		local.RegisterExtractor("noop", func(fs fs.FS, path string) local.Extractor { return noopExtractor{} })
		conf.Server.Scanner.Extractor = "noop"

		ls, err := storage.For(tmpFolder)
		Expect(err).ToNot(HaveOccurred())

		// It should implement Watcher
		var ok bool
		lsw, ok = ls.(storage.Watcher)
		Expect(ok).To(BeTrue())

		// Make sure temp folder is created
		Eventually(func() error {
			_, err := os.Stat(tmpFolder)
			return err
		}).Should(Succeed())
	})

	It("should start and stop watcher", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		w, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		cancel()
		Eventually(w).Should(BeClosed())
	})

	It("should return error if watcher is already started", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		_, err = lsw.Start(ctx)
		Expect(err).To(HaveOccurred())
	})

	It("should detect new files", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		changes, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(filepath.Join(tmpFolder, "test.txt"))
		Expect(err).ToNot(HaveOccurred())

		Eventually(changes).Should(Receive(Equal(tmpFolder)))
	})

	It("should detect new subfolders", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		changes, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.Mkdir(filepath.Join(tmpFolder, "subfolder"), 0755)).To(Succeed())

		Eventually(changes).Should(Receive(Equal(filepath.Join(tmpFolder, "subfolder"))))
	})

	It("should detect changes in subfolders recursively", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		subfolder := filepath.Join(tmpFolder, "subfolder1/subfolder2")
		Expect(os.MkdirAll(subfolder, 0755)).To(Succeed())

		changes, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		filePath := filepath.Join(subfolder, "test.txt")
		Expect(os.WriteFile(filePath, []byte("test"), 0600)).To(Succeed())

		Eventually(changes).Should(Receive(Equal(filePath)))
	})

	It("should detect removed in files", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		changes, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		filePath := filepath.Join(tmpFolder, "test.txt")
		Expect(os.WriteFile(filePath, []byte("test"), 0600)).To(Succeed())

		Eventually(changes).Should(Receive(Equal(filePath)))

		Expect(os.Remove(filePath)).To(Succeed())
		Eventually(changes).Should(Receive(Equal(filePath)))
	})

	It("should detect file moves", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		changes, err := lsw.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		filePath := filepath.Join(tmpFolder, "test.txt")
		Expect(os.WriteFile(filePath, []byte("test"), 0600)).To(Succeed())

		Eventually(changes).Should(Receive(Equal(filePath)))

		newPath := filepath.Join(tmpFolder, "test2.txt")
		Expect(os.Rename(filePath, newPath)).To(Succeed())
		Eventually(changes).Should(Receive(Equal(newPath)))
	})
})

type noopExtractor struct{}

func (s noopExtractor) Parse(files ...string) (map[string]metadata.Info, error) { return nil, nil }
func (s noopExtractor) Version() string                                         { return "0" }
