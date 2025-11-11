package core

import (
	"context"
	"errors"
	"sync"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
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

				// Wait for background goroutines to complete
				service.(*maintenanceService).wait()

				// RefreshStats should be called
				Expect(artistRepo.IsRefreshStatsCalled()).To(BeTrue(), "Artist stats should be refreshed")

				// Album should be updated with new calculated values
				Expect(albumRepo.GetPutCallCount()).To(BeNumerically(">", 0), "Album.Put() should be called to refresh album data")
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

	Describe("Album refresh logic", func() {
		var albumRepo *extendedAlbumRepo

		BeforeEach(func() {
			albumRepo = ds.MockedAlbum.(*extendedAlbumRepo)
		})

		Context("when album has no tracks after deletion", func() {
			It("skips the album without updating it", func() {
				// Setup album with no remaining tracks
				albumRepo.SetData(model.Albums{
					{ID: "album1", Name: "Empty Album", SongCount: 1},
				})
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
				})

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).ToNot(HaveOccurred())

				// Wait for background goroutines to complete
				service.(*maintenanceService).wait()

				// Album should NOT be updated because it has no tracks left
				Expect(albumRepo.GetPutCallCount()).To(Equal(0), "Album with no tracks should not be updated")
			})
		})

		Context("when Put fails for one album", func() {
			It("continues processing other albums", func() {
				albumRepo.SetData(model.Albums{
					{ID: "album1", Name: "Album 1"},
					{ID: "album2", Name: "Album 2"},
				})
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album1", Missing: false, Size: 1000, Duration: 180},
					{ID: "mf3", AlbumID: "album2", Missing: true},
					{ID: "mf4", AlbumID: "album2", Missing: false, Size: 2000, Duration: 200},
				})

				// Make Put fail on first call but succeed on subsequent calls
				albumRepo.putError = errors.New("put failed")
				albumRepo.failOnce = true

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf3"})

				// Should not fail even if one album's Put fails
				Expect(err).ToNot(HaveOccurred())

				// Wait for background goroutines to complete
				service.(*maintenanceService).wait()

				// Put should have been called multiple times
				Expect(albumRepo.GetPutCallCount()).To(BeNumerically(">", 0), "Put should be attempted")
			})
		})

		Context("when media file loading fails", func() {
			It("logs warning but continues when tracking affected albums fails", func() {
				// Set up log capturing
				hook, cleanup := tests.LogHook()
				defer cleanup()

				albumRepo.SetData(model.Albums{
					{ID: "album1", Name: "Album 1"},
				})
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
				})
				// Make GetAll fail when loading media files
				mfRepo.SetError(true)

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				// Deletion should succeed despite the tracking error
				Expect(err).ToNot(HaveOccurred())
				Expect(mfRepo.deleteMissingCalled).To(BeTrue())

				// Verify the warning was logged
				Expect(hook.LastEntry()).ToNot(BeNil())
				Expect(hook.LastEntry().Level).To(Equal(logrus.WarnLevel))
				Expect(hook.LastEntry().Message).To(Equal("Error tracking affected albums for refresh"))
			})
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
	mu           sync.RWMutex
	putCallCount int
	lastPutData  *model.Album
	putError     error
	failOnce     bool
}

func (m *extendedAlbumRepo) Put(album *model.Album) error {
	m.mu.Lock()
	m.putCallCount++
	m.lastPutData = album

	// Handle failOnce behavior
	var err error
	if m.putError != nil {
		if m.failOnce {
			err = m.putError
			m.putError = nil // Clear error after first failure
			m.mu.Unlock()
			return err
		}
		err = m.putError
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()

	return m.MockAlbumRepo.Put(album)
}

func (m *extendedAlbumRepo) GetPutCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.putCallCount
}

// Extension of MockArtistRepo to track RefreshStats calls
type extendedArtistRepo struct {
	*tests.MockArtistRepo
	mu                 sync.RWMutex
	refreshStatsCalled bool
	refreshStatsError  error
}

func (m *extendedArtistRepo) RefreshStats(allArtists bool) (int64, error) {
	m.mu.Lock()
	m.refreshStatsCalled = true
	err := m.refreshStatsError
	m.mu.Unlock()

	if err != nil {
		return 0, err
	}
	return m.MockArtistRepo.RefreshStats(allArtists)
}

func (m *extendedArtistRepo) IsRefreshStatsCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.refreshStatsCalled
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
