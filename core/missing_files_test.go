package core

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MissingFiles", func() {
	var ds *testDataStore
	var service MissingFiles
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		ctx = request.WithUser(ctx, model.User{ID: "user1", IsAdmin: true})

		ds = &testDataStore{
			mfRepo:     &testMediaFileRepo{},
			albumRepo:  &testAlbumRepo{},
			artistRepo: &testArtistRepo{},
		}

		service = NewMissingFiles(ds)
	})

	Describe("DeleteMissingFiles", func() {
		Context("with specific IDs", func() {
			It("deletes specific missing files", func() {
				// Setup: mock missing files with album IDs
				ds.mfRepo.files = model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album2", Missing: true},
				}

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
				Expect(ds.mfRepo.deleteMissingCalled).To(BeTrue())
				Expect(ds.mfRepo.deletedIDs).To(Equal([]string{"mf1", "mf2"}))
				Expect(ds.gcCalled).To(BeTrue())
			})

			It("returns error if deletion fails", func() {
				ds.mfRepo.deleteMissingError = errors.New("delete failed")

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("delete failed"))
			})

			It("continues even if album tracking fails", func() {
				ds.mfRepo.getAllError = errors.New("tracking failed")

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				// Should not fail, just log warning
				Expect(err).ToNot(HaveOccurred())
				Expect(ds.mfRepo.deleteMissingCalled).To(BeTrue())
			})

			It("returns error if GC fails", func() {
				ds.mfRepo.files = model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
				}
				ds.gcError = errors.New("gc failed")

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("gc failed"))
			})
		})

		Context("album ID extraction", func() {
			It("extracts unique album IDs from missing files", func() {
				ds.mfRepo.files = model.MediaFiles{
					{ID: "mf1", AlbumID: "album1", Missing: true},
					{ID: "mf2", AlbumID: "album1", Missing: true},
					{ID: "mf3", AlbumID: "album2", Missing: true},
				}

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2", "mf3"})

				Expect(err).ToNot(HaveOccurred())
				Expect(ds.mfRepo.getAllCalled).To(BeTrue())
			})

			It("skips files without album IDs", func() {
				ds.mfRepo.files = model.MediaFiles{
					{ID: "mf1", AlbumID: "", Missing: true},
					{ID: "mf2", AlbumID: "album1", Missing: true},
				}

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("DeleteAllMissingFiles", func() {
		It("deletes all missing files", func() {
			ds.mfRepo.files = model.MediaFiles{
				{ID: "mf1", AlbumID: "album1", Missing: true},
				{ID: "mf2", AlbumID: "album2", Missing: true},
				{ID: "mf3", AlbumID: "album3", Missing: true},
			}

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(ds.mfRepo.deleteAllMissingCalled).To(BeTrue())
			Expect(ds.gcCalled).To(BeTrue())
		})

		It("returns error if deletion fails", func() {
			ds.mfRepo.deleteAllMissingError = errors.New("delete all failed")

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete all failed"))
		})

		It("handles empty result gracefully", func() {
			ds.mfRepo.files = model.MediaFiles{}

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(ds.mfRepo.deleteAllMissingCalled).To(BeTrue())
		})
	})
})

// Test implementations
type testDataStore struct {
	tests.MockDataStore
	mfRepo     *testMediaFileRepo
	albumRepo  *testAlbumRepo
	artistRepo *testArtistRepo
	gcCalled   bool
	gcError    error
}

func (ds *testDataStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	return ds.mfRepo
}

func (ds *testDataStore) Album(ctx context.Context) model.AlbumRepository {
	return ds.albumRepo
}

func (ds *testDataStore) Artist(ctx context.Context) model.ArtistRepository {
	return ds.artistRepo
}

func (ds *testDataStore) WithTx(block func(tx model.DataStore) error, label ...string) error {
	return block(ds)
}

func (ds *testDataStore) GC(ctx context.Context) error {
	ds.gcCalled = true
	return ds.gcError
}

type testMediaFileRepo struct {
	tests.MockMediaFileRepo
	files                  model.MediaFiles
	getAllCalled           bool
	getAllError            error
	deleteMissingCalled    bool
	deletedIDs             []string
	deleteMissingError     error
	deleteAllMissingCalled bool
	deleteAllMissingError  error
}

func (m *testMediaFileRepo) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	m.getAllCalled = true
	if m.getAllError != nil {
		return nil, m.getAllError
	}

	if len(options) == 0 {
		return m.files, nil
	}

	// Filter based on the query options
	opt := options[0]
	if filters, ok := opt.Filters.(squirrel.And); ok {
		// Check for ID filter
		for _, filter := range filters {
			if eq, ok := filter.(squirrel.Eq); ok {
				if ids, exists := eq["id"]; exists {
					// Filter files by IDs
					idList := ids.([]string)
					var filtered model.MediaFiles
					for _, f := range m.files {
						for _, id := range idList {
							if f.ID == id {
								filtered = append(filtered, f)
								break
							}
						}
					}
					return filtered, nil
				}
			}
		}
	}
	return m.files, nil
}

func (m *testMediaFileRepo) DeleteMissing(ids []string) error {
	m.deleteMissingCalled = true
	m.deletedIDs = ids
	return m.deleteMissingError
}

func (m *testMediaFileRepo) DeleteAllMissing() (int64, error) {
	m.deleteAllMissingCalled = true
	if m.deleteAllMissingError != nil {
		return 0, m.deleteAllMissingError
	}
	return int64(len(m.files)), nil
}

type testAlbumRepo struct {
	tests.MockAlbumRepo
	refreshAlbumsCalled bool
	refreshAlbumsIDs    []string
	refreshAlbumsError  error
}

func (m *testAlbumRepo) RefreshAlbums(albumIDs []string) error {
	m.refreshAlbumsCalled = true
	m.refreshAlbumsIDs = albumIDs
	return m.refreshAlbumsError
}

type testArtistRepo struct {
	tests.MockArtistRepo
	refreshStatsCalled bool
	refreshStatsError  error
}

func (m *testArtistRepo) RefreshStats(allArtists bool) (int64, error) {
	m.refreshStatsCalled = true
	if m.refreshStatsError != nil {
		return 0, m.refreshStatsError
	}
	return 1, nil
}
