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
	var ds *tests.MockDataStore
	var mfRepo *extendedMediaFileRepo
	var service Maintenance
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		ctx = request.WithUser(ctx, model.User{ID: "user1", IsAdmin: true})

		ds, mfRepo = createTestDataStore()
		service = NewMaintenance(ds)
	})

	Describe("DeleteMissingFiles", func() {
		Context("with specific IDs", func() {
			It("deletes specific missing files", func() {
				// Setup: mock missing files with album IDs
				mfRepo.SetData(model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album2", Missing: true},
				})

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
				Expect(mfRepo.deleteMissingCalled).To(BeTrue())
				Expect(mfRepo.deletedIDs).To(Equal([]string{"mf1", "mf2"}))
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

				// Create a wrapper that returns error on GC
				dsWithGCError := &mockDataStoreWithGCError{MockDataStore: ds}
				serviceWithError := NewMaintenance(dsWithGCError)

				err := serviceWithError.DeleteMissingFiles(ctx, []string{"mf1"})

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
		It("deletes all missing files", func() {
			mfRepo.SetData(model.MediaFiles{
				{ID: "mf1", AlbumID: "album1", Missing: true},
				{ID: "mf2", AlbumID: "album2", Missing: true},
				{ID: "mf3", AlbumID: "album3", Missing: true},
			})

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
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
func createTestDataStore() (*tests.MockDataStore, *extendedMediaFileRepo) {
	ds := &tests.MockDataStore{}
	ds.MockedAlbum = tests.CreateMockAlbumRepo()
	ds.MockedArtist = tests.CreateMockArtistRepo()

	// Create extended media file repo with DeleteMissing support
	mfRepo := &extendedMediaFileRepo{
		MockMediaFileRepo: tests.CreateMockMediaFileRepo(),
	}
	ds.MockedMediaFile = mfRepo

	return ds, mfRepo
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

// Wrapper to override GC method to return error
type mockDataStoreWithGCError struct {
	*tests.MockDataStore
}

func (ds *mockDataStoreWithGCError) GC(ctx context.Context) error {
	return errors.New("gc failed")
}
