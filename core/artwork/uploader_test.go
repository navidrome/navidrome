package artwork

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Uploader", func() {
	var svc Uploader
	var tmpDir string
	var artRepo *tests.MockArtworkRepo
	var queueRepo *tests.MockArtworkQueueRepo

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tmpDir = GinkgoT().TempDir()
		conf.Server.DataFolder = conf.NewDir(tmpDir)
		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		ds := &tests.MockDataStore{MockedArtwork: artRepo, MockedArtworkQueue: queueRepo}
		svc = NewUploader(ds)
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

		It("does not touch artwork state or the queue (that is EnqueueArtwork's job, post-Put)", func() {
			ctx := context.Background()
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "ar", ItemID: "ar-1", Hash: "oldhash", Source: "external",
			})).To(Succeed())

			_, err := svc.SetImage(ctx, consts.EntityArtist, "ar-1", "Pink Floyd", "", strings.NewReader("img"), ".jpg")
			Expect(err).ToNot(HaveOccurred())

			// SetImage only writes the file; the state row survives and nothing is queued until
			// the caller has persisted the new filename and called EnqueueArtwork.
			_, err = artRepo.GetItemArtwork(model.KindArtistArtwork, "ar-1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(queueRepo.DequeueBatch(1000)).To(BeEmpty())
		})
	})

	Describe("SetAvatar", func() {
		It("writes into the avatar folder, named after id and username", func() {
			ctx := context.Background()
			big := makePNG(1024, 1024)
			filename, err := svc.SetAvatar(ctx, "u1", "deluan", "", bytes.NewReader(big), ".png")
			Expect(err).ToNot(HaveOccurred())
			Expect(filename).To(Equal("u1_deluan.png"))
			Expect(filepath.Join(tmpDir, "avatar", filename)).To(BeAnExistingFile())
		})

		It("shrinks a large image to MaxAvatarSize, preserving aspect ratio", func() {
			ctx := context.Background()
			big := makePNG(1024, 768)
			filename, err := svc.SetAvatar(ctx, "u1", "deluan", "", bytes.NewReader(big), ".png")
			Expect(err).ToNot(HaveOccurred())

			f, err := os.Open(filepath.Join(tmpDir, "avatar", filename))
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()
			cfg, _, err := image.DecodeConfig(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Width).To(Equal(consts.MaxAvatarSize))
			Expect(cfg.Height).To(Equal(384)) // aspect ratio kept, not padded to a square
		})

		It("keeps a small image intact instead of writing an empty file", func() {
			ctx := context.Background()
			small := makePNG(64, 64)
			filename, err := svc.SetAvatar(ctx, "u1", "deluan", "", bytes.NewReader(small), ".png")
			Expect(err).ToNot(HaveOccurred())

			data, err := os.ReadFile(filepath.Join(tmpDir, "avatar", filename))
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeEmpty())

			cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Width).To(Equal(64))
			Expect(cfg.Height).To(Equal(64))
		})

		It("removes the previous file", func() {
			ctx := context.Background()
			old := filepath.Join(tmpDir, "avatar", "u1_old.png")
			Expect(os.MkdirAll(filepath.Dir(old), 0755)).To(Succeed())
			Expect(os.WriteFile(old, []byte("x"), 0600)).To(Succeed())

			_, err := svc.SetAvatar(ctx, "u1", "deluan", old, bytes.NewReader(makePNG(64, 64)), ".png")
			Expect(err).ToNot(HaveOccurred())
			Expect(old).ToNot(BeAnExistingFile())
		})

		It("rejects a body that is not a decodable image", func() {
			ctx := context.Background()
			_, err := svc.SetAvatar(ctx, "u1", "deluan", "", strings.NewReader("not an image"), ".png")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("EnqueueArtwork", func() {
		It("clears artwork state and enqueues a Bump", func() {
			ctx := context.Background()
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "ar", ItemID: "ar-1", Hash: "oldhash", Source: "external",
			})).To(Succeed())

			svc.EnqueueArtwork(ctx, consts.EntityArtist, "ar-1")

			_, err := artRepo.GetItemArtwork(model.KindArtistArtwork, "ar-1", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))

			queued, err := queueRepo.DequeueBatch(1000)
			Expect(err).ToNot(HaveOccurred())
			Expect(queued).To(ContainElement(SatisfyAll(
				HaveField("ItemKind", "ar"),
				HaveField("ItemID", "ar-1"),
				HaveField("Priority", model.ArtworkPriorityBump),
			)))
		})

		It("is a no-op for an unknown entity type", func() {
			svc.EnqueueArtwork(context.Background(), "unknown", "x-1")
			Expect(queueRepo.DequeueBatch(1000)).To(BeEmpty())
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

var _ = Describe("MaxImageUploadSize", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("returns the configured size when valid", func() {
		conf.Server.MaxImageUploadSize = "20MB"
		Expect(MaxImageUploadSize()).To(Equal(int64(20_000_000)))
	})

	It("returns the default size when config is empty", func() {
		conf.Server.MaxImageUploadSize = ""
		Expect(MaxImageUploadSize()).To(Equal(int64(10_000_000)))
	})

	It("returns the default size when config is invalid", func() {
		conf.Server.MaxImageUploadSize = "not-a-size"
		Expect(MaxImageUploadSize()).To(Equal(int64(10_000_000)))
	})

	It("parses raw byte values", func() {
		conf.Server.MaxImageUploadSize = "52428800"
		Expect(MaxImageUploadSize()).To(Equal(int64(52_428_800)))
	})
})

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	Expect(png.Encode(&buf, img)).To(Succeed())
	return buf.Bytes()
}
