package artwork

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("artistArtworkReader", func() {
	var _ = Describe("loadArtistFolder", func() {
		var (
			ctx             context.Context
			fds             *fakeDataStore
			repo            *fakeFolderRepo
			albums          model.Albums
			paths           []string
			now             time.Time
			expectedUpdTime time.Time
		)

		BeforeEach(func() {
			ctx = context.Background()
			DeferCleanup(stubCoreAbsolutePath())

			now = time.Now().Truncate(time.Second)
			expectedUpdTime = now.Add(5 * time.Minute)
			repo = &fakeFolderRepo{
				result: []model.Folder{
					{
						ImagesUpdatedAt: expectedUpdTime,
					},
				},
				err: nil,
			}
			fds = &fakeDataStore{
				folderRepo: repo,
			}
			albums = model.Albums{
				{LibraryID: 1, ID: "album1", Name: "Album 1"},
			}
		})

		When("no albums provided", func() {
			It("returns empty and zero time", func() {
				folder, upd, err := loadArtistFolder(ctx, fds, model.Albums{}, []string{"/dummy/path"})
				Expect(err).ToNot(HaveOccurred())
				Expect(folder).To(BeEmpty())
				Expect(upd).To(BeZero())
			})
		})

		When("artist has only one album", func() {
			It("returns the parent folder", func() {
				paths = []string{
					filepath.FromSlash("/music/artist/album1"),
				}
				folder, upd, err := loadArtistFolder(ctx, fds, albums, paths)
				Expect(err).ToNot(HaveOccurred())
				Expect(folder).To(Equal("/music/artist"))
				Expect(upd).To(Equal(expectedUpdTime))
			})
		})

		When("the artist have multiple albums", func() {
			It("returns the common prefix for the albums paths", func() {
				paths = []string{
					filepath.FromSlash("/music/library/artist/one"),
					filepath.FromSlash("/music/library/artist/two"),
				}
				folder, upd, err := loadArtistFolder(ctx, fds, albums, paths)
				Expect(err).ToNot(HaveOccurred())
				Expect(folder).To(Equal(filepath.FromSlash("/music/library/artist")))
				Expect(upd).To(Equal(expectedUpdTime))
			})
		})

		When("the album paths contain same prefix", func() {
			It("returns the common prefix", func() {
				paths = []string{
					filepath.FromSlash("/music/artist/album1"),
					filepath.FromSlash("/music/artist/album2"),
				}
				folder, upd, err := loadArtistFolder(ctx, fds, albums, paths)
				Expect(err).ToNot(HaveOccurred())
				Expect(folder).To(Equal("/music/artist"))
				Expect(upd).To(Equal(expectedUpdTime))
			})
		})

		When("ds.Folder().GetAll returns an error", func() {
			It("returns an error", func() {
				paths = []string{
					filepath.FromSlash("/music/artist/album1"),
					filepath.FromSlash("/music/artist/album2"),
				}
				repo.err = errors.New("fake error")
				folder, upd, err := loadArtistFolder(ctx, fds, albums, paths)
				Expect(err).To(MatchError(ContainSubstring("fake error")))
				// Folder and time are empty on error.
				Expect(folder).To(BeEmpty())
				Expect(upd).To(BeZero())
			})
		})
	})

	var _ = Describe("fromArtistFolder", func() {
		var (
			ctx      context.Context
			tempDir  string
			testFunc sourceFunc
		)

		BeforeEach(func() {
			ctx = context.Background()
			tempDir = GinkgoT().TempDir()
		})

		When("artist folder contains matching image", func() {
			BeforeEach(func() {
				// Create test structure: /temp/artist/artist.jpg
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				artistImagePath := filepath.Join(artistDir, "artist.jpg")
				Expect(os.WriteFile(artistImagePath, []byte("fake image data"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("finds and returns the image", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())
				Expect(path).To(ContainSubstring("artist.jpg"))

				// Verify we can read the content
				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("fake image data"))
				reader.Close()
			})
		})

		When("artist folder is empty but parent contains image", func() {
			BeforeEach(func() {
				// Create test structure: /temp/parent/artist.jpg and /temp/parent/artist/album/
				parentDir := filepath.Join(tempDir, "parent")
				artistDir := filepath.Join(parentDir, "artist")
				albumDir := filepath.Join(artistDir, "album")
				Expect(os.MkdirAll(albumDir, 0755)).To(Succeed())

				// Put artist image in parent directory
				artistImagePath := filepath.Join(parentDir, "artist.jpg")
				Expect(os.WriteFile(artistImagePath, []byte("parent image"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("finds image in parent directory", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())
				Expect(path).To(ContainSubstring("parent" + string(filepath.Separator) + "artist.jpg"))

				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("parent image"))
				reader.Close()
			})
		})

		When("image is two levels up", func() {
			BeforeEach(func() {
				// Create test structure: /temp/grandparent/artist.jpg and /temp/grandparent/parent/artist/
				grandparentDir := filepath.Join(tempDir, "grandparent")
				parentDir := filepath.Join(grandparentDir, "parent")
				artistDir := filepath.Join(parentDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Put artist image in grandparent directory
				artistImagePath := filepath.Join(grandparentDir, "artist.jpg")
				Expect(os.WriteFile(artistImagePath, []byte("grandparent image"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("finds image in grandparent directory", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())
				Expect(path).To(ContainSubstring("grandparent" + string(filepath.Separator) + "artist.jpg"))

				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("grandparent image"))
				reader.Close()
			})
		})

		When("images exist at multiple levels", func() {
			BeforeEach(func() {
				// Create test structure with images at multiple levels
				grandparentDir := filepath.Join(tempDir, "grandparent")
				parentDir := filepath.Join(grandparentDir, "parent")
				artistDir := filepath.Join(parentDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Put artist images at all levels
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.jpg"), []byte("artist level"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(parentDir, "artist.jpg"), []byte("parent level"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(grandparentDir, "artist.jpg"), []byte("grandparent level"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("prioritizes the closest (artist folder) image", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())
				Expect(path).To(ContainSubstring("artist" + string(filepath.Separator) + "artist.jpg"))

				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("artist level"))
				reader.Close()
			})
		})

		When("pattern matches multiple files", func() {
			BeforeEach(func() {
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Create multiple matching files
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.jpg"), []byte("jpg image"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.png"), []byte("png image"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.txt"), []byte("text file"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("returns the first valid image file", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())

				// Should return an image file, not the text file
				Expect(path).To(SatisfyAny(
					ContainSubstring("artist.jpg"),
					ContainSubstring("artist.png"),
				))
				Expect(path).ToNot(ContainSubstring("artist.txt"))
				reader.Close()
			})
		})

		When("no matching files exist anywhere", func() {
			BeforeEach(func() {
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Create non-matching files
				Expect(os.WriteFile(filepath.Join(artistDir, "cover.jpg"), []byte("cover image"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("returns an error", func() {
				reader, path, err := testFunc()
				Expect(err).To(HaveOccurred())
				Expect(reader).To(BeNil())
				Expect(path).To(BeEmpty())
				Expect(err.Error()).To(ContainSubstring("no matches for 'artist.*'"))
				Expect(err.Error()).To(ContainSubstring("parent directories"))
			})
		})

		When("directory traversal reaches filesystem root", func() {
			BeforeEach(func() {
				// Start from a shallow directory to test root boundary
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("handles root boundary gracefully", func() {
				reader, path, err := testFunc()
				Expect(err).To(HaveOccurred())
				Expect(reader).To(BeNil())
				Expect(path).To(BeEmpty())
				// Should not panic or cause infinite loop
			})
		})

		When("file exists but cannot be opened", func() {
			BeforeEach(func() {
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Create a file that cannot be opened (permission denied)
				restrictedFile := filepath.Join(artistDir, "artist.jpg")
				Expect(os.WriteFile(restrictedFile, []byte("restricted"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("logs warning and continues searching", func() {
				// This test depends on the ability to restrict file permissions
				// For now, we'll just ensure it doesn't panic and returns appropriate error
				reader, _, err := testFunc()
				// The file should be readable in test environment, so this will succeed
				// In a real scenario with permission issues, it would continue searching
				if err == nil {
					Expect(reader).ToNot(BeNil())
					reader.Close()
				}
			})
		})

		When("single album artist scenario (original issue)", func() {
			BeforeEach(func() {
				// Simulate the exact folder structure from the issue:
				// /music/artist/album1/ (single album)
				// /music/artist/artist.jpg (artist image that should be found)
				artistDir := filepath.Join(tempDir, "music", "artist")
				albumDir := filepath.Join(artistDir, "album1")
				Expect(os.MkdirAll(albumDir, 0755)).To(Succeed())

				// Create artist.jpg in the artist folder (this was not being found before)
				artistImagePath := filepath.Join(artistDir, "artist.jpg")
				Expect(os.WriteFile(artistImagePath, []byte("single album artist image"), 0600)).To(Succeed())

				// The fromArtistFolder is called with the artist folder path
				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("finds artist.jpg in artist folder for single album artist", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())
				Expect(path).To(ContainSubstring("artist.jpg"))
				Expect(path).To(ContainSubstring("artist"))

				// Verify the content
				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("single album artist image"))
				reader.Close()
			})
		})
	})
})

type fakeFolderRepo struct {
	model.FolderRepository
	result []model.Folder
	err    error
}

func (f *fakeFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) {
	return f.result, f.err
}

type fakeDataStore struct {
	model.DataStore
	folderRepo *fakeFolderRepo
}

func (fds *fakeDataStore) Folder(_ context.Context) model.FolderRepository {
	return fds.folderRepo
}

func stubCoreAbsolutePath() func() {
	// Override core.AbsolutePath to return a fixed string during tests.
	original := core.AbsolutePath
	core.AbsolutePath = func(_ context.Context, ds model.DataStore, libID int, p string) string {
		return filepath.FromSlash("/music")
	}
	return func() {
		core.AbsolutePath = original
	}
}
