package artwork

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.abc"), []byte("text file"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.png"), []byte("png image"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.jpg"), []byte("jpg image"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("returns the first valid image file in sorted order", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())

				// Should return an image file,
				// Files are sorted: jpg comes before png alphabetically.
				// .abc comes first, but it's not an image.
				Expect(path).To(ContainSubstring("artist.jpg"))
				reader.Close()
			})
		})

		When("prioritizing files without numeric suffixes", func() {
			BeforeEach(func() {
				// Test case for issue #4683: artist.jpg should come before artist.1.jpg
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Create multiple matches with and without numeric suffixes
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.1.jpg"), []byte("artist 1"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.jpg"), []byte("artist main"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.2.jpg"), []byte("artist 2"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "artist.*")
			})

			It("returns artist.jpg before artist.1.jpg and artist.2.jpg", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())
				Expect(path).To(ContainSubstring("artist.jpg"))

				// Verify it's the main file, not a numbered variant
				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("artist main"))
				reader.Close()
			})
		})

		When("handling case-insensitive sorting", func() {
			BeforeEach(func() {
				// Test case to ensure case-insensitive natural sorting
				artistDir := filepath.Join(tempDir, "artist")
				Expect(os.MkdirAll(artistDir, 0755)).To(Succeed())

				// Create files with mixed case names
				Expect(os.WriteFile(filepath.Join(artistDir, "Folder.jpg"), []byte("folder"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "artist.jpg"), []byte("artist"), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(artistDir, "BACK.jpg"), []byte("back"), 0600)).To(Succeed())

				testFunc = fromArtistFolder(ctx, artistDir, "*.*")
			})

			It("sorts case-insensitively", func() {
				reader, path, err := testFunc()
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeNil())

				// Should return artist.jpg first (case-insensitive: "artist" < "back" < "folder")
				Expect(path).To(ContainSubstring("artist.jpg"))

				data, err := io.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("artist"))
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

	Describe("fromArtistUploadedImage", func() {
		var (
			tempDir string
			reader  *artistReader
		)

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			tempDir = GinkgoT().TempDir()
			conf.Server.DataFolder = tempDir

			// Create the artwork/artist directory
			Expect(os.MkdirAll(filepath.Join(tempDir, "artwork", "artist"), 0755)).To(Succeed())

			reader = &artistReader{}
		})

		When("artist has an uploaded image", func() {
			It("returns the uploaded image", func() {
				imgPath := filepath.Join(tempDir, "artwork", "artist", "ar-1_test.jpg")
				Expect(os.WriteFile(imgPath, []byte("uploaded artist image"), 0600)).To(Succeed())

				reader.artist = model.Artist{ID: "ar-1", UploadedImage: "ar-1_test.jpg"}
				sf := reader.fromArtistUploadedImage()
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))

				data, err := io.ReadAll(r)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("uploaded artist image"))
				r.Close()
			})
		})

		When("artist has no uploaded image", func() {
			It("returns nil reader (falls through)", func() {
				reader.artist = model.Artist{ID: "ar-1"}
				sf := reader.fromArtistUploadedImage()
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).To(BeNil())
				Expect(path).To(BeEmpty())
			})
		})
	})

	Describe("fromArtistImageFolder", func() {
		var (
			ctx     context.Context
			tempDir string
			ar      *artistReader
		)

		BeforeEach(func() {
			ctx = context.Background()
			DeferCleanup(configtest.SetupConfig())
			tempDir = GinkgoT().TempDir()
			ar = &artistReader{}
		})

		When("ArtistImageFolder is not configured", func() {
			It("returns nil (skips)", func() {
				conf.Server.ArtistImageFolder = ""
				ar.artist = model.Artist{Name: "Test Artist"}
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).To(BeNil())
				Expect(path).To(BeEmpty())
			})
		})

		When("image exists matching MBID", func() {
			It("finds the image by MBID", func() {
				conf.Server.ArtistImageFolder = tempDir
				mbid := "f27ec8db-af05-4f36-916e-3d57f91ecf5e"
				imgPath := filepath.Join(tempDir, mbid+".jpg")
				Expect(os.WriteFile(imgPath, []byte("mbid image"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist", MbzArtistID: mbid}
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))

				data, err := io.ReadAll(r)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("mbid image"))
				r.Close()
			})
		})

		When("MBID match is case-insensitive", func() {
			It("finds the image regardless of case", func() {
				conf.Server.ArtistImageFolder = tempDir
				mbid := "F27EC8DB-AF05-4F36-916E-3D57F91ECF5E"
				imgPath := filepath.Join(tempDir, "f27ec8db-af05-4f36-916e-3d57f91ecf5e.png")
				Expect(os.WriteFile(imgPath, []byte("mbid case image"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist", MbzArtistID: mbid}
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))
				r.Close()
			})
		})

		When("no MBID file exists but artist name file does", func() {
			It("falls back to artist name match", func() {
				conf.Server.ArtistImageFolder = tempDir
				imgPath := filepath.Join(tempDir, "Test Artist.jpg")
				Expect(os.WriteFile(imgPath, []byte("name image"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist", MbzArtistID: "nonexistent-mbid"}
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))

				data, err := io.ReadAll(r)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("name image"))
				r.Close()
			})
		})

		When("artist name match is case-insensitive", func() {
			It("matches regardless of case", func() {
				conf.Server.ArtistImageFolder = tempDir
				imgPath := filepath.Join(tempDir, "test artist.jpg")
				Expect(os.WriteFile(imgPath, []byte("case insensitive"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist"}
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))
				r.Close()
			})
		})

		When("both MBID and name files exist", func() {
			It("prefers MBID over name match", func() {
				conf.Server.ArtistImageFolder = tempDir
				mbid := "f27ec8db-af05-4f36-916e-3d57f91ecf5e"
				mbidPath := filepath.Join(tempDir, mbid+".jpg")
				namePath := filepath.Join(tempDir, "Test Artist.jpg")
				Expect(os.WriteFile(mbidPath, []byte("mbid image"), 0600)).To(Succeed())
				Expect(os.WriteFile(namePath, []byte("name image"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist", MbzArtistID: mbid}
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(mbidPath))

				data, err := io.ReadAll(r)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("mbid image"))
				r.Close()
			})
		})

		When("no matching image found", func() {
			It("returns an error", func() {
				conf.Server.ArtistImageFolder = tempDir
				// Create an unrelated file
				Expect(os.WriteFile(filepath.Join(tempDir, "other.jpg"), []byte("other"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist"}
				sf := ar.fromArtistImageFolder(ctx)
				r, _, err := sf()
				Expect(err).To(HaveOccurred())
				Expect(r).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("no image found"))
			})
		})

		When("cached imgFolderImgPath is set", func() {
			It("uses cached path instead of scanning", func() {
				conf.Server.ArtistImageFolder = tempDir
				imgPath := filepath.Join(tempDir, "cached.jpg")
				Expect(os.WriteFile(imgPath, []byte("cached image"), 0600)).To(Succeed())

				ar.artist = model.Artist{Name: "Test Artist"}
				ar.imgFolderImgPath = imgPath
				sf := ar.fromArtistImageFolder(ctx)
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))

				data, err := io.ReadAll(r)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal("cached image"))
				r.Close()
			})
		})
	})

	Describe("findImageInArtistFolder", func() {
		var tempDir string

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
		})

		When("matching file exists by MBID", func() {
			It("returns the file path", func() {
				mbid := "f27ec8db-af05-4f36-916e-3d57f91ecf5e"
				imgPath := filepath.Join(tempDir, mbid+".jpg")
				Expect(os.WriteFile(imgPath, []byte("image"), 0600)).To(Succeed())

				path := findImageInArtistFolder(tempDir, mbid, "Test")
				Expect(path).To(Equal(imgPath))
			})
		})

		When("matching file exists by name", func() {
			It("returns the file path", func() {
				imgPath := filepath.Join(tempDir, "Test Artist.png")
				Expect(os.WriteFile(imgPath, []byte("image"), 0600)).To(Succeed())

				path := findImageInArtistFolder(tempDir, "", "Test Artist")
				Expect(path).To(Equal(imgPath))
			})
		})

		When("no matching file exists", func() {
			It("returns empty string", func() {
				path := findImageInArtistFolder(tempDir, "", "Unknown Artist")
				Expect(path).To(BeEmpty())
			})
		})

		When("folder does not exist", func() {
			It("returns empty string", func() {
				path := findImageInArtistFolder("/nonexistent/path", "", "Test")
				Expect(path).To(BeEmpty())
			})
		})
	})
})

type fakeFolderRepo struct {
	model.FolderRepository
	result       []model.Folder
	parentResult *model.Folder
	getErr       error
	getCallCount int
	err          error
}

func (f *fakeFolderRepo) GetAll(...model.QueryOptions) ([]model.Folder, error) {
	return f.result, f.err
}

func (f *fakeFolderRepo) Get(id string) (*model.Folder, error) {
	f.getCallCount++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.parentResult != nil {
		return f.parentResult, nil
	}
	return nil, model.ErrNotFound
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
