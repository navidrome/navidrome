package artwork

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("folderArtworkReader", func() {
	var (
		ctx        context.Context
		a          *artwork
		tmpDir     string
		folderRepo *fakeFolderRepo
		folder     model.Folder
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		tmpDir = GinkgoT().TempDir()
		conf.Server.CoverArtPriority = "cover.*, front.*, *"

		folderRepo = &fakeFolderRepo{}
		ds := &fakeDataStore{folderRepo: folderRepo}
		a = &artwork{ds: ds}

		folder = model.Folder{
			ID:              "folder-1",
			LibraryPath:     tmpDir,
			Path:            ".",
			Name:            "Jazz",
			ImageFiles:      []string{"cover.jpg"},
			ImagesUpdatedAt: time.Now().Truncate(time.Second),
		}
	})

	createImage := func(name string) {
		fullPath := filepath.Join(folder.AbsolutePath(), name)
		Expect(os.MkdirAll(filepath.Dir(fullPath), 0755)).To(Succeed())
		Expect(os.WriteFile(fullPath, []byte("image data"), 0600)).To(Succeed())
	}

	Describe("newFolderArtworkReader", func() {
		It("returns a reader when the folder is found", func() {
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			Expect(reader).ToNot(BeNil())
		})

		It("returns an error when the folder is not found", func() {
			artID := model.NewArtworkID(model.KindFolderArtwork, "missing", nil)
			_, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("builds absolute image file paths from folder.ImageFiles", func() {
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			Expect(reader.imgFiles).To(ConsistOf(
				filepath.Join(folder.AbsolutePath(), "cover.jpg"),
			))
		})

		It("uses ImagesUpdatedAt as the cache key lastUpdate", func() {
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			Expect(reader.LastUpdated()).To(Equal(folder.ImagesUpdatedAt))
		})
	})

	Describe("Reader", func() {
		It("returns the matching image file", func() {
			createImage("cover.jpg")
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			rc, path, err := reader.Reader(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(rc).ToNot(BeNil())
			Expect(path).To(ContainSubstring("cover.jpg"))
			rc.Close()
		})

		It("returns ErrUnavailable when no images match the priority patterns", func() {
			conf.Server.CoverArtPriority = "cover.*"
			folder.ImageFiles = []string{"other.jpg"}
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			_, _, err = reader.Reader(ctx)
			Expect(err).To(MatchError(ErrUnavailable))
		})

		It("skips embedded and external patterns without error", func() {
			conf.Server.CoverArtPriority = "embedded, external, cover.*"
			createImage("cover.jpg")
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			rc, _, err := reader.Reader(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(rc).ToNot(BeNil())
			rc.Close()
		})

		It("returns ErrUnavailable when folder has no images and only external/embedded in priority", func() {
			conf.Server.CoverArtPriority = "embedded, external"
			folder.ImageFiles = []string{}
			folderRepo.parentResult = &folder
			artID := model.NewArtworkID(model.KindFolderArtwork, "folder-1", nil)
			reader, err := newFolderArtworkReader(ctx, a, artID)
			Expect(err).ToNot(HaveOccurred())
			_, _, err = reader.Reader(ctx)
			Expect(err).To(MatchError(ErrUnavailable))
		})
	})
})
