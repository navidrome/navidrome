package artwork

import (
	"context"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album Artwork Reader", func() {
	Describe("loadAlbumFoldersPaths", func() {
		var (
			ctx        context.Context
			ds         *fakeDataStore
			repo       *fakeFolderRepo
			album      model.Album
			now        time.Time
			expectedAt time.Time
		)

		BeforeEach(func() {
			ctx = context.Background()
			now = time.Now().Truncate(time.Second)
			expectedAt = now.Add(5 * time.Minute)

			// Set up the test folders with image files
			repo = &fakeFolderRepo{}
			ds = &fakeDataStore{
				folderRepo: repo,
			}
			album = model.Album{
				ID:        "album1",
				Name:      "Album",
				FolderIDs: []string{"folder1", "folder2", "folder3"},
			}
		})

		It("returns sorted image files", func() {
			repo.result = []model.Folder{
				{
					Path:            "Artist/Album/Disc1",
					ImagesUpdatedAt: expectedAt,
					ImageFiles:      []string{"cover.jpg", "back.jpg", "cover.1.jpg"},
				},
				{
					Path:            "Artist/Album/Disc2",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
				{
					Path:            "Artist/Album/Disc10",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
			}

			_, imgFiles, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(*imagesUpdatedAt).To(Equal(expectedAt))

			// Check that image files are sorted by base name (without extension)
			Expect(imgFiles).To(HaveLen(5))

			// Files should be sorted by base filename without extension, then by full path
			// "back" < "cover", so back.jpg comes first
			// Then all cover.jpg files, sorted by path
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/Disc1/back.jpg")))
			Expect(imgFiles[1]).To(Equal(filepath.FromSlash("Artist/Album/Disc1/cover.jpg")))
			Expect(imgFiles[2]).To(Equal(filepath.FromSlash("Artist/Album/Disc2/cover.jpg")))
			Expect(imgFiles[3]).To(Equal(filepath.FromSlash("Artist/Album/Disc10/cover.jpg")))
			Expect(imgFiles[4]).To(Equal(filepath.FromSlash("Artist/Album/Disc1/cover.1.jpg")))
		})

		It("prioritizes files without numeric suffixes", func() {
			// Test case for issue #4683: cover.jpg should come before cover.1.jpg
			repo.result = []model.Folder{
				{
					Path:            "Artist/Album",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.1.jpg", "cover.jpg", "cover.2.jpg"},
				},
			}

			_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(imgFiles).To(HaveLen(3))

			// cover.jpg should come first because "cover" < "cover.1" < "cover.2"
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/cover.jpg")))
			Expect(imgFiles[1]).To(Equal(filepath.FromSlash("Artist/Album/cover.1.jpg")))
			Expect(imgFiles[2]).To(Equal(filepath.FromSlash("Artist/Album/cover.2.jpg")))
		})

		It("handles case-insensitive sorting", func() {
			// Test that Cover.jpg and cover.jpg are treated as equivalent
			repo.result = []model.Folder{
				{
					Path:            "Artist/Album",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"Folder.jpg", "cover.jpg", "BACK.jpg"},
				},
			}

			_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(imgFiles).To(HaveLen(3))

			// Files should be sorted case-insensitively: BACK, cover, Folder
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/BACK.jpg")))
			Expect(imgFiles[1]).To(Equal(filepath.FromSlash("Artist/Album/cover.jpg")))
			Expect(imgFiles[2]).To(Equal(filepath.FromSlash("Artist/Album/Folder.jpg")))
		})
	})
})
