package artwork

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// commonDir and loadArtistFolder decide how far above the albums the artist folder sits. A layout
// can only show the image that won, so the arithmetic is pinned here.
var _ = Describe("commonDir", func() {
	It("returns the folder itself for a single path", func() {
		Expect(commonDir([]string{filepath.FromSlash("/music/artist/album")})).
			To(Equal(filepath.FromSlash("/music/artist/album")))
	})

	It("returns the deepest shared folder", func() {
		Expect(commonDir([]string{
			filepath.FromSlash("/music/artist/album/cd1"),
			filepath.FromSlash("/music/artist/album/cd2"),
		})).To(Equal(filepath.FromSlash("/music/artist/album")))
	})

	It("does not read a shared name fragment as a shared folder", func() {
		Expect(commonDir([]string{
			filepath.FromSlash("/music/artist/Album"),
			filepath.FromSlash("/music/artist/Album2"),
		})).To(Equal(filepath.FromSlash("/music/artist")))
	})
})

var _ = Describe("loadArtistFolder", func() {
	var (
		ctx       context.Context
		ds        *tests.MockDataStore
		repo      *fakeFolderRepo
		albums    model.Albums
		updatedAt time.Time
	)

	BeforeEach(func() {
		ctx = context.Background()
		DeferCleanup(stubCoreAbsolutePath())

		updatedAt = time.Now().Truncate(time.Second).Add(5 * time.Minute)
		repo = &fakeFolderRepo{result: []model.Folder{{ImagesUpdatedAt: updatedAt}}}
		ds = &tests.MockDataStore{MockedFolder: repo}
		albums = model.Albums{{LibraryID: 1, ID: "album1", Name: "Album 1"}}
	})

	It("returns empty when the artist has no albums", func() {
		folder, upd, err := loadArtistFolder(ctx, ds, model.Albums{}, []string{"/dummy/path"})

		Expect(err).ToNot(HaveOccurred())
		Expect(folder).To(BeEmpty())
		Expect(upd).To(BeZero())
	})

	It("climbs above the album folder when the artist has a single album root", func() {
		folder, upd, err := loadArtistFolder(ctx, ds, albums,
			[]string{filepath.FromSlash("/music/artist/album1")})

		Expect(err).ToNot(HaveOccurred())
		Expect(folder).To(Equal(filepath.FromSlash("/music/artist")))
		Expect(upd).To(Equal(updatedAt))
	})

	It("climbs above the shared folder when two albums live in the same one", func() {
		folder, upd, err := loadArtistFolder(ctx, ds, albums, []string{
			filepath.FromSlash("/music/artist/split"),
			filepath.FromSlash("/music/artist/split"),
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(folder).To(Equal(filepath.FromSlash("/music/artist")))
		Expect(upd).To(Equal(updatedAt))
	})

	It("stops at the folder where distinct album roots already meet", func() {
		folder, upd, err := loadArtistFolder(ctx, ds, albums, []string{
			filepath.FromSlash("/music/artist/album1"),
			filepath.FromSlash("/music/artist/album2"),
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(folder).To(Equal(filepath.FromSlash("/music/artist")))
		Expect(upd).To(Equal(updatedAt))
	})

	It("returns the error when the folder lookup fails", func() {
		repo.err = errors.New("fake error")

		folder, upd, err := loadArtistFolder(ctx, ds, albums, []string{
			filepath.FromSlash("/music/artist/album1"),
			filepath.FromSlash("/music/artist/album2"),
		})

		Expect(err).To(MatchError(ContainSubstring("fake error")))
		Expect(folder).To(BeEmpty())
		Expect(upd).To(BeZero())
	})
})

func stubCoreAbsolutePath() func() {
	original := core.AbsolutePath
	core.AbsolutePath = func(context.Context, model.DataStore, int, string) string {
		return filepath.FromSlash("/music")
	}
	return func() { core.AbsolutePath = original }
}
