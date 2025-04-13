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
			repo = &fakeFolderRepo{
				result: []model.Folder{
					{
						Path:            "Artist/Album/Disc1",
						ImagesUpdatedAt: expectedAt,
						ImageFiles:      []string{"cover.jpg", "back.jpg"},
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
				},
				err: nil,
			}
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
			_, imgFiles, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(*imagesUpdatedAt).To(Equal(expectedAt))

			// Check that image files are sorted alphabetically
			Expect(imgFiles).To(HaveLen(4))

			// The files should be sorted by full path
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/Disc1/back.jpg")))
			Expect(imgFiles[1]).To(Equal(filepath.FromSlash("Artist/Album/Disc1/cover.jpg")))
			Expect(imgFiles[2]).To(Equal(filepath.FromSlash("Artist/Album/Disc10/cover.jpg")))
			Expect(imgFiles[3]).To(Equal(filepath.FromSlash("Artist/Album/Disc2/cover.jpg")))
		})
	})
})
