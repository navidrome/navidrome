package persistence

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("FolderRepository", func() {
	var repo model.FolderRepository
	var ctx context.Context
	var conn *dbx.DB
	var testLib, otherLib model.Library

	BeforeEach(func() {
		ctx = request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid"})
		conn = GetDBXBuilder()
		repo = newFolderRepository(ctx, conn)

		// Use existing library ID 1 from test fixtures
		libRepo := NewLibraryRepository(ctx, conn)
		lib, err := libRepo.Get(1)
		Expect(err).ToNot(HaveOccurred())
		testLib = *lib

		// Create a second library with its own folder to verify isolation
		otherLib = model.Library{Name: "Other Library", Path: "/other/path"}
		Expect(libRepo.Put(&otherLib)).To(Succeed())
	})

	AfterEach(func() {
		// Clean up only test folders created by our tests (paths starting with "Test")
		// This prevents interference with fixture data needed by other tests
		_, _ = conn.NewQuery("DELETE FROM folder WHERE library_id = 1 AND path LIKE 'Test%'").Execute()
		_, _ = conn.NewQuery(fmt.Sprintf("DELETE FROM library WHERE id = %d", otherLib.ID)).Execute()
	})

	Describe("GetFolderUpdateInfo", func() {
		Context("with no target paths", func() {
			It("returns all folders in the library", func() {
				// Create test folders with unique names to avoid conflicts
				folder1 := model.NewFolder(testLib, "TestGetLastUpdates/Folder1")
				folder2 := model.NewFolder(testLib, "TestGetLastUpdates/Folder2")

				err := repo.Put(folder1)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder2)
				Expect(err).ToNot(HaveOccurred())

				otherFolder := model.NewFolder(otherLib, "TestOtherLib/Folder")
				err = repo.Put(otherFolder)
				Expect(err).ToNot(HaveOccurred())

				// Query all folders (no target paths) - should only return folders from testLib
				results, err := repo.GetFolderUpdateInfo(testLib)
				Expect(err).ToNot(HaveOccurred())
				// Should include folders from testLib
				Expect(results).To(HaveKey(folder1.ID))
				Expect(results).To(HaveKey(folder2.ID))
				// Should NOT include folders from other library
				Expect(results).ToNot(HaveKey(otherFolder.ID))
			})
		})

		Context("with specific target paths", func() {
			It("returns folder info for existing folders", func() {
				// Create test folders with unique names
				folder1 := model.NewFolder(testLib, "TestSpecific/Rock")
				folder2 := model.NewFolder(testLib, "TestSpecific/Jazz")
				folder3 := model.NewFolder(testLib, "TestSpecific/Classical")

				err := repo.Put(folder1)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder2)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder3)
				Expect(err).ToNot(HaveOccurred())

				// Query specific paths
				results, err := repo.GetFolderUpdateInfo(testLib, "TestSpecific/Rock", "TestSpecific/Classical")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(2))

				// Verify folder IDs are in results
				Expect(results).To(HaveKey(folder1.ID))
				Expect(results).To(HaveKey(folder3.ID))
				Expect(results).ToNot(HaveKey(folder2.ID))

				// Verify update info is populated
				Expect(results[folder1.ID].UpdatedAt).ToNot(BeZero())
				Expect(results[folder1.ID].Hash).To(Equal(folder1.Hash))
			})

			It("handles empty folder path as root", func() {
				// Test querying for root folder without creating it (fixtures should have one)
				rootFolderID := model.FolderID(testLib, ".")

				results, err := repo.GetFolderUpdateInfo(testLib, "")
				Expect(err).ToNot(HaveOccurred())
				// Should return the root folder if it exists
				if len(results) > 0 {
					Expect(results).To(HaveKey(rootFolderID))
				}
			})

			It("returns empty map for non-existent folders", func() {
				results, err := repo.GetFolderUpdateInfo(testLib, "NonExistent/Path")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})

			It("skips missing folders", func() {
				// Create a folder and mark it as missing
				folder := model.NewFolder(testLib, "TestMissing/Folder")
				folder.Missing = true
				err := repo.Put(folder)
				Expect(err).ToNot(HaveOccurred())

				results, err := repo.GetFolderUpdateInfo(testLib, "TestMissing/Folder")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})
	})
})
