package core

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Maintenance", func() {
	var ds *extendedDataStore
	var mfRepo *extendedMediaFileRepo
	var service Maintenance
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		ctx = request.WithUser(ctx, model.User{ID: "user1", IsAdmin: true})

		ds = createTestDataStore()
		mfRepo = ds.MockedMediaFile.(*extendedMediaFileRepo)
		service = NewMaintenance(ds)
	})

	Describe("DeleteMissingFiles", func() {
		Context("with specific IDs", func() {
			It("deletes specific missing files and runs GC", func() {
				// Setup: mock missing files with album IDs
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album2", Missing: true},
				})

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
				Expect(mfRepo.deleteMissingCalled).To(BeTrue())
				Expect(mfRepo.deletedIDs).To(Equal([]string{"mf1", "mf2"}))
				Expect(ds.gcCalled).To(BeTrue(), "GC should be called after deletion")
			})

			It("triggers artist stats refresh and album refresh after deletion", func() {
				artistRepo := ds.MockedArtist.(*extendedArtistRepo)
				// Setup: mock missing files with albums
				albumRepo := ds.MockedAlbum.(*extendedAlbumRepo)
				albumRepo.SetData(model.Albums{
					{ID: "album1", Name: "Test Album", SongCount: 5},
				})
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album1", Missing: false, Size: 1000, Duration: 180},
					{ID: "mf3", AlbumID: "album1", Missing: false, Size: 2000, Duration: 200},
				})

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).ToNot(HaveOccurred())
				// RefreshStats should be called asynchronously
				Eventually(func() bool {
					return artistRepo.refreshStatsCalled
				}).Should(BeTrue(), "Artist stats should be refreshed")

				// Album should be updated with new calculated values
				Eventually(func() bool {
					return albumRepo.putCalled
				}).Should(BeTrue(), "Album.Put() should be called to refresh album data")
			})

			It("returns error if deletion fails", func() {
				mfRepo.deleteMissingError = errors.New("delete failed")

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("delete failed"))
			})

			It("continues even if album tracking fails", func() {
				mfRepo.SetError(true)

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				// Should not fail, just log warning
				Expect(err).ToNot(HaveOccurred())
				Expect(mfRepo.deleteMissingCalled).To(BeTrue())
			})

			It("returns error if GC fails", func() {
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
				})

				// Set GC to return error
				ds.gcError = errors.New("gc failed")

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("gc failed"))
			})
		})

		Context("album ID extraction", func() {
			It("extracts unique album IDs from missing files", func() {
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album1", Missing: true},
					{ID: "mf3", AlbumID: "album2", Missing: true},
				})

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2", "mf3"})

				Expect(err).ToNot(HaveOccurred())
			})

			It("skips files without album IDs", func() {
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "", Missing: true},
					{ID: "mf2", AlbumID: "album1", Missing: true},
				})

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("DeleteAllMissingFiles", func() {
		It("deletes all missing files and runs GC", func() {
			mfRepo.SetData(model.MediaFiles{
				{ID: "mf1", AlbumID: "album1", Missing: true},
				{ID: "mf2", AlbumID: "album2", Missing: true},
				{ID: "mf3", AlbumID: "album3", Missing: true},
			})

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(ds.gcCalled).To(BeTrue(), "GC should be called after deletion")
		})

		It("returns error if deletion fails", func() {
			mfRepo.SetError(true)

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).To(HaveOccurred())
		})

		It("handles empty result gracefully", func() {
			mfRepo.SetData(model.MediaFiles{})

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
		})
	})
})

// Test helper to create a mock DataStore with controllable behavior
func createTestDataStore() *extendedDataStore {
	// Create extended datastore with GC tracking
	ds := &extendedDataStore{
		MockDataStore: &tests.MockDataStore{},
	}

	// Create extended album repo with Put tracking
	albumRepo := &extendedAlbumRepo{
		MockAlbumRepo: tests.CreateMockAlbumRepo(),
	}
	ds.MockedAlbum = albumRepo

	// Create extended artist repo with RefreshStats tracking
	artistRepo := &extendedArtistRepo{
		MockArtistRepo: tests.CreateMockArtistRepo(),
	}
	ds.MockedArtist = artistRepo

	// Create extended media file repo with DeleteMissing support
	mfRepo := &extendedMediaFileRepo{
		MockMediaFileRepo: tests.CreateMockMediaFileRepo(),
	}
	ds.MockedMediaFile = mfRepo

	return ds
}

// Extension of MockMediaFileRepo to add DeleteMissing method
type extendedMediaFileRepo struct {
	*tests.MockMediaFileRepo
	deleteMissingCalled bool
	deletedIDs          []string
	deleteMissingError  error
}

func (m *extendedMediaFileRepo) DeleteMissing(ids []string) error {
	m.deleteMissingCalled = true
	m.deletedIDs = ids
	if m.deleteMissingError != nil {
		return m.deleteMissingError
	}
	// Actually delete from the mock data
	for _, id := range ids {
		delete(m.Data, id)
	}
	return nil
}

// Extension of MockAlbumRepo to track Put calls
type extendedAlbumRepo struct {
	*tests.MockAlbumRepo
	putCalled   bool
	lastPutData *model.Album
}

func (m *extendedAlbumRepo) Put(album *model.Album) error {
	m.putCalled = true
	m.lastPutData = album
	return m.MockAlbumRepo.Put(album)
}

// Extension of MockArtistRepo to track RefreshStats calls
type extendedArtistRepo struct {
	*tests.MockArtistRepo
	refreshStatsCalled bool
	refreshStatsError  error
}

func (m *extendedArtistRepo) RefreshStats(allArtists bool) (int64, error) {
	m.refreshStatsCalled = true
	if m.refreshStatsError != nil {
		return 0, m.refreshStatsError
	}
	return m.MockArtistRepo.RefreshStats(allArtists)
}

// Extension of MockDataStore to track GC calls
type extendedDataStore struct {
	*tests.MockDataStore
	gcCalled bool
	gcError  error
}

func (ds *extendedDataStore) GC(ctx context.Context) error {
	ds.gcCalled = true
	if ds.gcError != nil {
		return ds.gcError
	}
	return ds.MockDataStore.GC(ctx)
}
