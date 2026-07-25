package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
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

			It("includes all child folders when querying parent", func() {
				tests.SkipOnWindows("path storage (#TBD-path-sep-persistence)")
				// Create a parent folder with multiple children
				parent := model.NewFolder(testLib, "TestParent/Music")
				child1 := model.NewFolder(testLib, "TestParent/Music/Rock/Queen")
				child2 := model.NewFolder(testLib, "TestParent/Music/Jazz")
				otherParent := model.NewFolder(testLib, "TestParent2/Music/Jazz")

				Expect(repo.Put(parent)).To(Succeed())
				Expect(repo.Put(child1)).To(Succeed())
				Expect(repo.Put(child2)).To(Succeed())

				// Query the parent folder - should return parent and all children
				results, err := repo.GetFolderUpdateInfo(testLib, "TestParent/Music")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(3))
				Expect(results).To(HaveKey(parent.ID))
				Expect(results).To(HaveKey(child1.ID))
				Expect(results).To(HaveKey(child2.ID))
				Expect(results).ToNot(HaveKey(otherParent.ID))
			})

			It("excludes children from other libraries", func() {
				tests.SkipOnWindows("path storage (#TBD-path-sep-persistence)")
				// Create parent in testLib
				parent := model.NewFolder(testLib, "TestIsolation/Parent")
				child := model.NewFolder(testLib, "TestIsolation/Parent/Child")

				Expect(repo.Put(parent)).To(Succeed())
				Expect(repo.Put(child)).To(Succeed())

				// Create similar path in other library
				otherParent := model.NewFolder(otherLib, "TestIsolation/Parent")
				otherChild := model.NewFolder(otherLib, "TestIsolation/Parent/Child")

				Expect(repo.Put(otherParent)).To(Succeed())
				Expect(repo.Put(otherChild)).To(Succeed())

				// Query should only return folders from testLib
				results, err := repo.GetFolderUpdateInfo(testLib, "TestIsolation/Parent")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(2))
				Expect(results).To(HaveKey(parent.ID))
				Expect(results).To(HaveKey(child.ID))
				Expect(results).ToNot(HaveKey(otherParent.ID))
				Expect(results).ToNot(HaveKey(otherChild.ID))
			})

			It("excludes missing children when querying parent", func() {
				tests.SkipOnWindows("path storage (#TBD-path-sep-persistence)")
				// Create parent and children, mark one as missing
				parent := model.NewFolder(testLib, "TestMissingChild/Parent")
				child1 := model.NewFolder(testLib, "TestMissingChild/Parent/Child1")
				child2 := model.NewFolder(testLib, "TestMissingChild/Parent/Child2")
				child2.Missing = true

				Expect(repo.Put(parent)).To(Succeed())
				Expect(repo.Put(child1)).To(Succeed())
				Expect(repo.Put(child2)).To(Succeed())

				// Query parent - should only return parent and non-missing child
				results, err := repo.GetFolderUpdateInfo(testLib, "TestMissingChild/Parent")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(2))
				Expect(results).To(HaveKey(parent.ID))
				Expect(results).To(HaveKey(child1.ID))
				Expect(results).ToNot(HaveKey(child2.ID))
			})

			It("handles mix of existing and non-existing target paths", func() {
				tests.SkipOnWindows("path storage (#TBD-path-sep-persistence)")
				// Create folders for one path but not the other
				existingParent := model.NewFolder(testLib, "TestMixed/Exists")
				existingChild := model.NewFolder(testLib, "TestMixed/Exists/Child")

				Expect(repo.Put(existingParent)).To(Succeed())
				Expect(repo.Put(existingChild)).To(Succeed())

				// Query both existing and non-existing paths
				results, err := repo.GetFolderUpdateInfo(testLib, "TestMixed/Exists", "TestMixed/DoesNotExist")
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(2))
				Expect(results).To(HaveKey(existingParent.ID))
				Expect(results).To(HaveKey(existingChild.ID))
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

	Describe("HasAudioOutsideFolders", func() {
		var albumRoot, disc1, disc2 *model.Folder

		// TestHasAudio/Album/
		// ├── CD1/   (audio, belongs to the album)
		// └── CD2/   (audio, belongs to the album)
		BeforeEach(func() {
			albumRoot = model.NewFolder(testLib, "TestHasAudio/Album")
			disc1 = model.NewFolder(testLib, "TestHasAudio/Album/CD1")
			disc1.NumAudioFiles = 5
			disc2 = model.NewFolder(testLib, "TestHasAudio/Album/CD2")
			disc2.NumAudioFiles = 5
			for _, f := range []*model.Folder{albumRoot, disc1, disc2} {
				Expect(repo.Put(f)).To(Succeed())
			}
		})

		It("returns false when all audio under the parent belongs to the given folders", func() {
			Expect(repo.HasAudioOutsideFolders(*albumRoot, []string{disc1.ID, disc2.ID})).To(BeFalse())
		})

		It("returns true when another folder under the parent has audio", func() {
			bonus := model.NewFolder(testLib, "TestHasAudio/Album/Bonus")
			bonus.NumAudioFiles = 1
			Expect(repo.Put(bonus)).To(Succeed())

			Expect(repo.HasAudioOutsideFolders(*albumRoot, []string{disc1.ID, disc2.ID})).To(BeTrue())
		})

		It("returns true when the parent itself contains audio files", func() {
			albumRoot.NumAudioFiles = 2

			Expect(repo.HasAudioOutsideFolders(*albumRoot, []string{disc1.ID, disc2.ID})).To(BeTrue())
		})

		It("ignores audio outside the parent's subtree", func() {
			other := model.NewFolder(testLib, "TestHasAudio/Other Album")
			other.NumAudioFiles = 10
			Expect(repo.Put(other)).To(Succeed())

			Expect(repo.HasAudioOutsideFolders(*albumRoot, []string{disc1.ID, disc2.ID})).To(BeFalse())
		})

		It("ignores missing folders", func() {
			gone := model.NewFolder(testLib, "TestHasAudio/Album/Gone")
			gone.NumAudioFiles = 3
			gone.Missing = true
			Expect(repo.Put(gone)).To(Succeed())

			Expect(repo.HasAudioOutsideFolders(*albumRoot, []string{disc1.ID, disc2.ID})).To(BeFalse())
		})

		It("does not treat LIKE wildcards in the parent path as patterns", func() {
			// "TestHas_udio" would LIKE-match "TestHasAudio" if "_" were not escaped
			wildcardRoot := model.NewFolder(testLib, "TestHas_udio/Album")
			Expect(repo.Put(wildcardRoot)).To(Succeed())

			Expect(repo.HasAudioOutsideFolders(*wildcardRoot, []string{"none"})).To(BeFalse())
		})
	})

	Describe("wrapFolderCursor", func() {
		It("does not panic when the cursor yields a dbFolder with nil Folder", func() {
			// Simulate what queryWithStableResults does on the rows.Err() path:
			// it yields a zero-value dbFolder (where Folder is nil) with an error.
			dbErr := fmt.Errorf("database is locked")
			cursor := func(yield func(dbFolder, error) bool) {
				var empty dbFolder // Folder pointer is nil
				yield(empty, dbErr)
			}

			// wrapFolderCursor should handle the nil Folder without panicking
			wrappedCursor := wrapFolderCursor(cursor)
			var gotErr error
			Expect(func() {
				for _, err := range wrappedCursor {
					gotErr = err
				}
			}).ToNot(Panic())
			Expect(gotErr).To(HaveOccurred())
			Expect(gotErr.Error()).To(ContainSubstring("unexpected nil model.Folder"))
			Expect(errors.Is(gotErr, dbErr)).To(BeTrue(), "should wrap the original cursor error")
		})

		It("yields folders from a valid cursor", func() {
			folder := &model.Folder{ID: "f1", Name: "Test"}
			cursor := func(yield func(dbFolder, error) bool) {
				yield(dbFolder{Folder: folder}, nil)
			}

			wrappedCursor := wrapFolderCursor(cursor)
			var folders []model.Folder
			for f, err := range wrappedCursor {
				Expect(err).ToNot(HaveOccurred())
				folders = append(folders, f)
			}
			Expect(folders).To(HaveLen(1))
			Expect(folders[0].ID).To(Equal("f1"))
		})
	})

	Describe("GetAllWithPlaylists", func() {
		It("returns all non-missing folders with playlists, ignoring the scan-timestamp gate", func() {
			withPls := model.NewFolder(testLib, "TestAllPls/WithPls")
			withPls.NumPlaylists = 2
			noPls := model.NewFolder(testLib, "TestAllPls/NoPls")
			noPls.NumPlaylists = 0
			missingWithPls := model.NewFolder(testLib, "TestAllPls/Missing")
			missingWithPls.NumPlaylists = 1
			missingWithPls.Missing = true

			Expect(repo.Put(withPls)).To(Succeed())
			Expect(repo.Put(noPls)).To(Succeed())
			Expect(repo.Put(missingWithPls)).To(Succeed())

			// Force the folder's updated_at to the past so GetTouchedWithPlaylists
			// (which gates on updated_at > last_scan_at) would NOT return it.
			_, err := conn.NewQuery("UPDATE folder SET updated_at = {:t} WHERE id = {:id}").
				Bind(dbx.Params{"t": "2000-01-01 00:00:00", "id": withPls.ID}).Execute()
			Expect(err).ToNot(HaveOccurred())

			var ids []string
			cursor, err := repo.GetAllWithPlaylists()
			Expect(err).ToNot(HaveOccurred())
			for f, err := range cursor {
				Expect(err).ToNot(HaveOccurred())
				ids = append(ids, f.ID)
			}

			Expect(ids).To(ConsistOf(withPls.ID)) // only the non-missing folder with playlists
		})
	})
})
