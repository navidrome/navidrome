package persistence

import (
	"context"

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
	var testLib model.Library

	BeforeEach(func() {
		ctx = request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid"})
		conn = GetDBXBuilder()
		repo = newFolderRepository(ctx, conn)

		// Use existing library ID 1 from test fixtures
		libRepo := NewLibraryRepository(ctx, conn)
		lib, err := libRepo.Get(1)
		Expect(err).ToNot(HaveOccurred())
		testLib = *lib
	})

	AfterEach(func() {
		// Clean up test folders created by these tests
		// Only delete folders with paths starting with our test prefix
		_, _ = conn.NewQuery("DELETE FROM folder WHERE library_id = 1 AND (path LIKE 'TestFolder%' OR path LIKE 'Music/%' OR path = 'Classical' OR path = 'Podcasts')").Execute()
	})

	Describe("GetByPaths", func() {
		Context("with valid targets", func() {
			It("returns folder info for existing folders", func() {
				// Create test folders
				folder1 := model.NewFolder(testLib, "Music/Rock")
				folder2 := model.NewFolder(testLib, "Music/Jazz")
				folder3 := model.NewFolder(testLib, "Classical")

				err := repo.Put(folder1)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder2)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder3)
				Expect(err).ToNot(HaveOccurred())

				// Query by paths
				targets := []model.LibraryPath{
					{LibraryID: testLib.ID, FolderPath: "Music/Rock"},
					{LibraryID: testLib.ID, FolderPath: "Classical"},
				}

				results, err := repo.GetByPaths(targets)
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
				// Create root folder
				rootFolder := model.NewFolder(testLib, ".")
				err := repo.Put(rootFolder)
				Expect(err).ToNot(HaveOccurred())

				targets := []model.LibraryPath{
					{LibraryID: testLib.ID, FolderPath: ""},
				}

				results, err := repo.GetByPaths(targets)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results).To(HaveKey(rootFolder.ID))
			})

			It("returns empty map for non-existent folders", func() {
				targets := []model.LibraryPath{
					{LibraryID: testLib.ID, FolderPath: "NonExistent/Path"},
				}

				results, err := repo.GetByPaths(targets)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})

			It("skips missing folders", func() {
				// Create a folder and mark it as missing
				folder := model.NewFolder(testLib, "Music/Missing")
				folder.Missing = true
				err := repo.Put(folder)
				Expect(err).ToNot(HaveOccurred())

				targets := []model.LibraryPath{
					{LibraryID: testLib.ID, FolderPath: "Music/Missing"},
				}

				results, err := repo.GetByPaths(targets)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})

		Context("with invalid library IDs", func() {
			It("returns empty map for non-existent library", func() {
				targets := []model.LibraryPath{
					{LibraryID: 99999, FolderPath: "Music"},
				}

				results, err := repo.GetByPaths(targets)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})

		Context("with empty targets", func() {
			It("returns empty map", func() {
				results, err := repo.GetByPaths([]model.LibraryPath{})
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})

			It("returns empty map for nil targets", func() {
				results, err := repo.GetByPaths(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})

		Context("with multiple paths in same library", func() {
			It("returns multiple folders", func() {
				// Create multiple folders in the same library
				folder1 := model.NewFolder(testLib, "Music/Pop")
				folder2 := model.NewFolder(testLib, "Music/Electronic")
				folder3 := model.NewFolder(testLib, "Podcasts")

				err := repo.Put(folder1)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder2)
				Expect(err).ToNot(HaveOccurred())
				err = repo.Put(folder3)
				Expect(err).ToNot(HaveOccurred())

				// Query multiple paths
				targets := []model.LibraryPath{
					{LibraryID: testLib.ID, FolderPath: "Music/Pop"},
					{LibraryID: testLib.ID, FolderPath: "Music/Electronic"},
					{LibraryID: testLib.ID, FolderPath: "Podcasts"},
				}

				results, err := repo.GetByPaths(targets)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(3))
				Expect(results).To(HaveKey(folder1.ID))
				Expect(results).To(HaveKey(folder2.ID))
				Expect(results).To(HaveKey(folder3.ID))
			})
		})
	})
})
