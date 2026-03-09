package artwork

import (
	"context"
	"errors"
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

		It("includes images from parent folder for multi-disc albums", func() {
			// Simulates: Artist/Album/cover.jpg with tracks in Artist/Album/CD1/ and Artist/Album/CD2/
			repo.result = []model.Folder{
				{
					ID:              "folder1",
					Path:            "Artist/Album",
					Name:            "CD1",
					ParentID:        "parentFolder",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{},
				},
				{
					ID:              "folder2",
					Path:            "Artist/Album",
					Name:            "CD2",
					ParentID:        "parentFolder",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{},
				},
			}
			repo.parentResult = &model.Folder{
				ID:              "parentFolder",
				Path:            "Artist",
				Name:            "Album",
				ImagesUpdatedAt: expectedAt,
				ImageFiles:      []string{"cover.jpg", "back.jpg"},
			}

			_, imgFiles, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(*imagesUpdatedAt).To(Equal(expectedAt))
			Expect(imgFiles).To(HaveLen(2))
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/back.jpg")))
			Expect(imgFiles[1]).To(Equal(filepath.FromSlash("Artist/Album/cover.jpg")))
		})

		It("does not query parent when parent ID is already in album folders", func() {
			// When the parent folder is already one of the album's folders, skip it
			repo.result = []model.Folder{
				{
					ID:              "folder1",
					Path:            "Artist",
					Name:            "Album",
					ParentID:        "folder2",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
				{
					ID:              "folder2",
					Path:            "",
					Name:            "Artist",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{},
				},
			}

			_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(imgFiles).To(HaveLen(1))
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/cover.jpg")))
			// Get should not have been called (parent already in folder set)
			Expect(repo.getCallCount).To(Equal(0))
		})

		It("does not query parent when folders have different parents", func() {
			// When album folders span different parents, don't search any parent
			repo.result = []model.Folder{
				{
					ID:              "folder1",
					Path:            "Artist1/Album",
					Name:            "part1",
					ParentID:        "parentA",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
				{
					ID:              "folder2",
					Path:            "Artist2/Album",
					Name:            "part2",
					ParentID:        "parentB",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{},
				},
			}

			_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(imgFiles).To(HaveLen(1))
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist1/Album/part1/cover.jpg")))
			// Get should not have been called (different parents)
			Expect(repo.getCallCount).To(Equal(0))
		})

		It("does not query parent for single-folder albums", func() {
			// A single-folder album's parent is typically the artist folder,
			// which should not be searched for cover art
			repo.result = []model.Folder{
				{
					ID:              "folder1",
					Path:            "Artist",
					Name:            "Album",
					ParentID:        "artistFolder",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
			}

			_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(imgFiles).To(HaveLen(1))
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/cover.jpg")))
			// Get should not have been called (single folder, no parent lookup)
			Expect(repo.getCallCount).To(Equal(0))
		})

		It("propagates non-ErrNotFound errors from parent folder lookup", func() {
			repo.result = []model.Folder{
				{
					ID:              "folder1",
					Path:            "Artist/Album",
					Name:            "CD1",
					ParentID:        "parentFolder",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
				{
					ID:              "folder2",
					Path:            "Artist/Album",
					Name:            "CD2",
					ParentID:        "parentFolder",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{},
				},
			}
			repo.getErr = errors.New("db connection failed")

			_, _, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).To(MatchError("db connection failed"))
			Expect(repo.getCallCount).To(Equal(1))
		})

		It("continues gracefully when parent folder is not found", func() {
			// Parent folder may have been deleted; should log a warning and continue
			repo.result = []model.Folder{
				{
					ID:              "folder1",
					Path:            "Artist/Album",
					Name:            "CD1",
					ParentID:        "missingParent",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{"cover.jpg"},
				},
				{
					ID:              "folder2",
					Path:            "Artist/Album",
					Name:            "CD2",
					ParentID:        "missingParent",
					ImagesUpdatedAt: now,
					ImageFiles:      []string{},
				},
			}
			// parentResult is nil, so Get will return ErrNotFound

			_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

			Expect(err).ToNot(HaveOccurred())
			Expect(imgFiles).To(HaveLen(1))
			Expect(imgFiles[0]).To(Equal(filepath.FromSlash("Artist/Album/CD1/cover.jpg")))
			Expect(repo.getCallCount).To(Equal(1))
		})
	})
})
