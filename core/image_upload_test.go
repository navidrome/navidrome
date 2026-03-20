package core_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ImageUploadService", func() {
	var svc core.ImageUploadService
	var tmpDir string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tmpDir = GinkgoT().TempDir()
		conf.Server.DataFolder = tmpDir
		svc = core.NewImageUploadService()
	})

	Describe("SetImage", func() {
		It("creates directory and saves image file", func() {
			ctx := context.Background()
			reader := strings.NewReader("fake image data")
			filename, err := svc.SetImage(ctx, consts.EntityArtist, "ar-1", "Pink Floyd", "", reader, ".jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(filename).To(Equal("ar-1_pink_floyd.jpg"))

			absPath := filepath.Join(tmpDir, "artwork", "artist", "ar-1_pink_floyd.jpg")
			data, err := os.ReadFile(absPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal("fake image data"))
		})

		It("falls back to ID-only filename when name cleans to empty", func() {
			ctx := context.Background()
			reader := strings.NewReader("data")
			filename, err := svc.SetImage(ctx, consts.EntityPlaylist, "pl-1", "!!!", "", reader, ".png")
			Expect(err).ToNot(HaveOccurred())
			Expect(filename).To(Equal("pl-1.png"))
		})

		It("removes old image when replacing", func() {
			ctx := context.Background()
			oldDir := filepath.Join(tmpDir, "artwork", "artist")
			Expect(os.MkdirAll(oldDir, 0755)).To(Succeed())
			oldFile := filepath.Join(oldDir, "ar-1_old.png")
			Expect(os.WriteFile(oldFile, []byte("old"), 0600)).To(Succeed())

			reader := strings.NewReader("new image")
			_, err := svc.SetImage(ctx, consts.EntityArtist, "ar-1", "New Name", oldFile, reader, ".jpg")
			Expect(err).ToNot(HaveOccurred())
			Expect(oldFile).ToNot(BeAnExistingFile())

			newPath := filepath.Join(oldDir, "ar-1_new_name.jpg")
			Expect(newPath).To(BeAnExistingFile())
		})

		It("ignores missing old file without error", func() {
			ctx := context.Background()
			reader := strings.NewReader("data")
			_, err := svc.SetImage(ctx, consts.EntityArtist, "ar-1", "Name", "/nonexistent/path.jpg", reader, ".jpg")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("RemoveImage", func() {
		It("removes the file at the given path", func() {
			ctx := context.Background()
			dir := filepath.Join(tmpDir, "artwork", "artist")
			Expect(os.MkdirAll(dir, 0755)).To(Succeed())
			path := filepath.Join(dir, "ar-1_test.jpg")
			Expect(os.WriteFile(path, []byte("img"), 0600)).To(Succeed())

			err := svc.RemoveImage(ctx, path)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).ToNot(BeAnExistingFile())
		})

		It("succeeds when file does not exist", func() {
			ctx := context.Background()
			err := svc.RemoveImage(ctx, "/nonexistent/file.jpg")
			Expect(err).ToNot(HaveOccurred())
		})

		It("succeeds with empty path", func() {
			ctx := context.Background()
			err := svc.RemoveImage(ctx, "")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
