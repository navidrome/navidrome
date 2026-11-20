package artwork

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// spyFolderRepo counts parent lookups and can fail them or the album-root check.
type spyFolderRepo struct {
	model.FolderRepository
	getCalls      int
	getErr        error
	otherAudioErr error
}

func (f *spyFolderRepo) Get(ctx context.Context, id string) (*model.Folder, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.FolderRepository.Get(ctx, id)
}

func (f *spyFolderRepo) HasAudioOutsideFolders(ctx context.Context, parent model.Folder, excludeIDs []string) (bool, error) {
	if f.otherAudioErr != nil {
		return false, f.otherAudioErr
	}
	return f.FolderRepository.HasAudioOutsideFolders(ctx, parent, excludeIDs)
}

// The e2e specs cover which image wins for a layout; these pin what a layout cannot reach: that
// the album-root parent is only fetched when it could qualify, and what happens when that fails.
var _ = Describe("loadAlbumFoldersPaths", func() {
	var (
		ctx   context.Context
		ds    model.DataStore
		repo  *spyFolderRepo
		album model.Album
		now   time.Time
	)

	putFolders := func(folders ...model.Folder) {
		GinkgoHelper()
		for _, f := range folders {
			f.LibraryID = model.DefaultLibraryID
			Expect(ds.Folder().Put(ctx, &f)).To(Succeed())
		}
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		now = time.Now().Truncate(time.Second)
		real := newTestDS(ctx)
		repo = &spyFolderRepo{FolderRepository: real.Folder()}
		ds = persistencetest.Overlay(real, func(ds model.DataStore) model.DataStore {
			return &folderOverlay{DataStore: ds, folder: repo}
		})
		album = model.Album{
			ID:        "album1",
			Name:      "Album",
			FolderIDs: []string{"folder1", "folder2", "folder3"},
		}
	})

	It("does not query the parent when it is already one of the album's folders", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist", Name: "Album", ParentID: "folder2",
				ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}},
			model.Folder{ID: "folder2", Path: "", Name: "Artist", ImagesUpdatedAt: now},
		)

		_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("Artist/Album/cover.jpg"))
		Expect(repo.getCalls).To(BeZero())
	})

	It("does not query the parent when the album's folders have different parents", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist1/Album", Name: "part1", ParentID: "parentA",
				ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}},
			model.Folder{ID: "folder2", Path: "Artist2/Album", Name: "part2", ParentID: "parentB",
				ImagesUpdatedAt: now},
		)

		_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("Artist1/Album/part1/cover.jpg"))
		Expect(repo.getCalls).To(BeZero())
	})

	It("does not query the parent for a single-folder album that has images of its own", func() {
		putFolders(model.Folder{ID: "folder1", Path: "Artist", Name: "Album", ParentID: "artistFolder",
			ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}})

		_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("Artist/Album/cover.jpg"))
		Expect(repo.getCalls).To(BeZero())
	})

	It("does not promote the library root, so its images never become album art", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: ".", Name: "AlbumPart1", ParentID: "rootFolder",
				ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}},
			model.Folder{ID: "folder2", Path: ".", Name: "AlbumPart2", ParentID: "rootFolder",
				ImagesUpdatedAt: now},
			model.Folder{ID: "rootFolder", Name: ".", ImageFiles: []string{"unrelated.jpg"}},
		)

		_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("AlbumPart1/cover.jpg"))
		Expect(repo.getCalls).To(Equal(1))
	})

	It("does not promote a parent that holds another album's audio", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist/Album", Name: "CD1", ParentID: "albumFolder",
				ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}},
			model.Folder{ID: "folder2", Path: "Artist/Album", Name: "CD2", ParentID: "albumFolder",
				ImagesUpdatedAt: now},
			model.Folder{ID: "albumFolder", Path: "Artist", Name: "Album",
				ParentID: "artistFolder", ImageFiles: []string{"artist.jpg"}},
			model.Folder{ID: "otherAlbum", Path: "Artist/Album", Name: "Bonus", ParentID: "albumFolder",
				NumAudioFiles: 3},
		)

		_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("Artist/Album/CD1/cover.jpg"))
	})

	It("promotes the album root parent into the returned paths", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist/Album", Name: "CD1", ParentID: "albumFolder",
				ImagesUpdatedAt: now, NumAudioFiles: 5},
			model.Folder{ID: "folder2", Path: "Artist/Album", Name: "CD2", ParentID: "albumFolder",
				ImagesUpdatedAt: now, NumAudioFiles: 5},
			model.Folder{ID: "albumFolder", Path: "Artist", Name: "Album",
				ParentID: "artistFolder", ImageFiles: []string{"cover.jpg"}},
		)

		paths, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("Artist/Album/cover.jpg"))
		Expect(paths).To(HaveLen(3))
	})

	It("propagates errors from the album-root check", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist/Album", Name: "disc1", ParentID: "albumFolder",
				ImagesUpdatedAt: now},
			model.Folder{ID: "albumFolder", Path: "Artist", Name: "Album",
				ParentID: "artistFolder", ImageFiles: []string{"cover.jpg"}},
		)
		repo.otherAudioErr = errors.New("db connection failed")

		_, _, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).To(MatchError("db connection failed"))
	})

	It("propagates non-ErrNotFound errors from the parent folder lookup", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist/Album", Name: "CD1", ParentID: "parentFolder",
				ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}},
			model.Folder{ID: "folder2", Path: "Artist/Album", Name: "CD2", ParentID: "parentFolder",
				ImagesUpdatedAt: now},
		)
		repo.getErr = errors.New("db connection failed")

		_, _, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).To(MatchError("db connection failed"))
		Expect(repo.getCalls).To(Equal(1))
	})

	It("continues when the parent folder has been deleted", func() {
		putFolders(
			model.Folder{ID: "folder1", Path: "Artist/Album", Name: "CD1", ParentID: "missingParent",
				ImagesUpdatedAt: now, ImageFiles: []string{"cover.jpg"}},
			model.Folder{ID: "folder2", Path: "Artist/Album", Name: "CD2", ParentID: "missingParent",
				ImagesUpdatedAt: now},
		)

		_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, album)

		Expect(err).ToNot(HaveOccurred())
		Expect(imgFiles).To(ConsistOf("Artist/Album/CD1/cover.jpg"))
		Expect(repo.getCalls).To(Equal(1))
	})
})

// folderImages is the sort that decides which of several same-named images wins, so it is pinned
// directly rather than through a layout that can only show the winner.
var _ = Describe("folderImages", func() {
	It("prefers base filenames over numeric-suffixed ones", func() {
		imgFiles, _ := folderImages([]model.Folder{
			{Path: "Artist", Name: "Album", ImageFiles: []string{"cover.1.jpg", "cover.jpg", "cover.2.jpg"}},
		})

		Expect(imgFiles).To(HaveExactElements(
			"Artist/Album/cover.jpg", "Artist/Album/cover.1.jpg", "Artist/Album/cover.2.jpg"))
	})

	It("prefers shallower paths when the base filenames tie", func() {
		imgFiles, _ := folderImages([]model.Folder{
			{Path: "Artist/Album", Name: "CD1", ImageFiles: []string{"cover.jpg"}},
			{Path: "Artist", Name: "Album", ImageFiles: []string{"cover.jpg"}},
		})

		Expect(imgFiles).To(HaveExactElements("Artist/Album/cover.jpg", "Artist/Album/CD1/cover.jpg"))
	})

	It("sorts case-insensitively", func() {
		imgFiles, _ := folderImages([]model.Folder{
			{Path: "Artist", Name: "Album", ImageFiles: []string{"Cover.jpg", "back.JPG"}},
		})

		Expect(imgFiles).To(HaveExactElements("Artist/Album/back.JPG", "Artist/Album/Cover.jpg"))
	})

	It("reports the newest ImagesUpdatedAt across the folders", func() {
		now := time.Now().Truncate(time.Second)
		newest := now.Add(5 * time.Minute)

		_, updatedAt := folderImages([]model.Folder{
			{Path: "Artist", Name: "Album", ImagesUpdatedAt: now},
			{Path: "Artist/Album", Name: "CD1", ImagesUpdatedAt: newest},
		})

		Expect(updatedAt).To(Equal(newest))
	})
})
