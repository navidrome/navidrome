package artwork

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("artistReader", func() {
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
