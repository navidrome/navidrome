package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/slice"
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

	Describe("folderSubtreeFilter", func() {
		var parent, child, grandchild, other *model.Folder

		matching := func(paths ...string) []string {
			GinkgoHelper()
			folders, err := repo.GetAll(model.QueryOptions{Filters: folderSubtreeFilter(testLib, paths)})
			Expect(err).ToNot(HaveOccurred())
			return slice.Map(folders, func(f model.Folder) string { return f.ID })
		}

		BeforeEach(func() {
			parent = model.NewFolder(testLib, "TestSubtree")
			child = model.NewFolder(testLib, "TestSubtree/Child")
			grandchild = model.NewFolder(testLib, "TestSubtree/Child/Grandchild")
			other = model.NewFolder(testLib, "TestSubtreeOther")
			for _, f := range []*model.Folder{parent, child, grandchild, other} {
				Expect(repo.Put(f)).To(Succeed())
			}
			DeferCleanup(func() {
				_, _ = conn.NewQuery("DELETE FROM folder WHERE name LIKE 'TestSubtree%' OR path LIKE 'TestSubtree%'").Execute()
			})
		})

		It("matches a folder and all its descendants", func() {
			Expect(matching("TestSubtree")).To(ConsistOf(parent.ID, child.ID, grandchild.ID))
		})

		It("matches the descendants of a nested slash-form path", func() {
			Expect(matching("TestSubtree/Child")).To(ConsistOf(child.ID, grandchild.ID))
		})

		It("matches the whole library for the root path", func() {
			Expect(matching(".")).To(ContainElements(parent.ID, child.ID, grandchild.ID, other.ID))
		})
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

	Describe("GetRootSubfoldersWithAudio and GetSubfoldersWithAudio", func() {
		var (
			rootFolder, otherRoot           *model.Folder
			rootWithDirect                  *model.Folder
			rootWithDesc                    *model.Folder
			rootWithDescChild               *model.Folder
			rootWithGrandchild              *model.Folder
			rootWithGrandchildChild         *model.Folder
			rootWithGrandchildGrandchild    *model.Folder
			rootEmpty                       *model.Folder
			rootEmptyChild                  *model.Folder
			rootMissing                     *model.Folder
			rootDescMissing                 *model.Folder
			rootDescMissingChild            *model.Folder
			rootSortA, rootSortB, rootSortC *model.Folder
			rootOtherLib                    *model.Folder
			rootSpecialWildcard             *model.Folder
			rootSpecialChild                *model.Folder
			rootSpecialFalseMatch           *model.Folder

			parentFolder               *model.Folder
			childDirect                *model.Folder
			childWithDesc              *model.Folder
			childWithDescGrandchild    *model.Folder
			childEmpty                 *model.Folder
			childEmptyGrandchild       *model.Folder
			childMissing               *model.Folder
			childDescMissing           *model.Folder
			childDescMissingGrandchild *model.Folder
			otherParentFolder          *model.Folder
			otherParentChild           *model.Folder
		)

		BeforeEach(func() {
			// Ensure root folders exist for both libraries
			rootFolder = model.NewFolder(testLib, ".")
			Expect(repo.Put(rootFolder)).To(Succeed())

			otherRoot = model.NewFolder(otherLib, ".")
			Expect(repo.Put(otherRoot)).To(Succeed())

			// 1. Top-level folder with direct audio
			rootWithDirect = model.NewFolder(testLib, "TestAudioRootDirect")
			rootWithDirect.NumAudioFiles = 3
			Expect(repo.Put(rootWithDirect)).To(Succeed())

			// 2. Top-level folder with 0 direct audio, but child has audio
			rootWithDesc = model.NewFolder(testLib, "TestAudioRootDesc")
			rootWithDesc.NumAudioFiles = 0
			Expect(repo.Put(rootWithDesc)).To(Succeed())
			rootWithDescChild = model.NewFolder(testLib, "TestAudioRootDesc/Child")
			rootWithDescChild.NumAudioFiles = 4
			Expect(repo.Put(rootWithDescChild)).To(Succeed())

			// 3. Top-level folder with 0 direct audio, grandchild has audio
			rootWithGrandchild = model.NewFolder(testLib, "TestAudioRootGrandchild")
			rootWithGrandchild.NumAudioFiles = 0
			Expect(repo.Put(rootWithGrandchild)).To(Succeed())
			rootWithGrandchildChild = model.NewFolder(testLib, "TestAudioRootGrandchild/Album")
			rootWithGrandchildChild.NumAudioFiles = 0
			Expect(repo.Put(rootWithGrandchildChild)).To(Succeed())
			rootWithGrandchildGrandchild = model.NewFolder(testLib, "TestAudioRootGrandchild/Album/CD1")
			rootWithGrandchildGrandchild.NumAudioFiles = 2
			Expect(repo.Put(rootWithGrandchildGrandchild)).To(Succeed())

			// 4. Empty top-level folder
			rootEmpty = model.NewFolder(testLib, "TestAudioRootEmpty")
			rootEmpty.NumAudioFiles = 0
			Expect(repo.Put(rootEmpty)).To(Succeed())
			rootEmptyChild = model.NewFolder(testLib, "TestAudioRootEmpty/Sub")
			rootEmptyChild.NumAudioFiles = 0
			Expect(repo.Put(rootEmptyChild)).To(Succeed())

			// 5. Missing top-level folder (has audio directly)
			rootMissing = model.NewFolder(testLib, "TestAudioRootMissing")
			rootMissing.NumAudioFiles = 5
			rootMissing.Missing = true
			Expect(repo.Put(rootMissing)).To(Succeed())

			// 6. Top-level folder whose only audio-bearing child is missing
			rootDescMissing = model.NewFolder(testLib, "TestAudioRootDescMissing")
			rootDescMissing.NumAudioFiles = 0
			Expect(repo.Put(rootDescMissing)).To(Succeed())
			rootDescMissingChild = model.NewFolder(testLib, "TestAudioRootDescMissing/Sub")
			rootDescMissingChild.NumAudioFiles = 3
			rootDescMissingChild.Missing = true
			Expect(repo.Put(rootDescMissingChild)).To(Succeed())

			// 7. Case-insensitive sorting folders
			rootSortB = model.NewFolder(testLib, "TestAudioSort_b")
			rootSortB.NumAudioFiles = 1
			Expect(repo.Put(rootSortB)).To(Succeed())
			rootSortA = model.NewFolder(testLib, "TestAudioSort_A")
			rootSortA.NumAudioFiles = 1
			Expect(repo.Put(rootSortA)).To(Succeed())
			rootSortC = model.NewFolder(testLib, "TestAudioSort_c")
			rootSortC.NumAudioFiles = 1
			Expect(repo.Put(rootSortC)).To(Succeed())

			// 8. Other library folder
			rootOtherLib = model.NewFolder(otherLib, "TestAudioOtherLibRoot")
			rootOtherLib.NumAudioFiles = 2
			Expect(repo.Put(rootOtherLib)).To(Succeed())

			// 9. Special character escaping: "TestAudio_AC%DC" vs "TestAudio_AC-DC"
			rootSpecialWildcard = model.NewFolder(testLib, "TestAudio_AC%DC")
			rootSpecialWildcard.NumAudioFiles = 0
			Expect(repo.Put(rootSpecialWildcard)).To(Succeed())
			rootSpecialChild = model.NewFolder(testLib, "TestAudio_AC%DC/Album")
			rootSpecialChild.NumAudioFiles = 1
			Expect(repo.Put(rootSpecialChild)).To(Succeed())

			rootSpecialFalseMatch = model.NewFolder(testLib, "TestAudio_AC-DC")
			rootSpecialFalseMatch.NumAudioFiles = 0
			Expect(repo.Put(rootSpecialFalseMatch)).To(Succeed())

			// Hierarchy for GetSubfoldersWithAudio
			parentFolder = model.NewFolder(testLib, "TestAudioSub_Parent")
			parentFolder.NumAudioFiles = 0
			Expect(repo.Put(parentFolder)).To(Succeed())

			childDirect = model.NewFolder(testLib, "TestAudioSub_Parent/ChildDirect")
			childDirect.NumAudioFiles = 2
			Expect(repo.Put(childDirect)).To(Succeed())

			childWithDesc = model.NewFolder(testLib, "TestAudioSub_Parent/ChildWithDesc")
			childWithDesc.NumAudioFiles = 0
			Expect(repo.Put(childWithDesc)).To(Succeed())
			childWithDescGrandchild = model.NewFolder(testLib, "TestAudioSub_Parent/ChildWithDesc/Sub")
			childWithDescGrandchild.NumAudioFiles = 3
			Expect(repo.Put(childWithDescGrandchild)).To(Succeed())

			childEmpty = model.NewFolder(testLib, "TestAudioSub_Parent/ChildEmpty")
			childEmpty.NumAudioFiles = 0
			Expect(repo.Put(childEmpty)).To(Succeed())
			childEmptyGrandchild = model.NewFolder(testLib, "TestAudioSub_Parent/ChildEmpty/Sub")
			childEmptyGrandchild.NumAudioFiles = 0
			Expect(repo.Put(childEmptyGrandchild)).To(Succeed())

			childMissing = model.NewFolder(testLib, "TestAudioSub_Parent/ChildMissing")
			childMissing.NumAudioFiles = 4
			childMissing.Missing = true
			Expect(repo.Put(childMissing)).To(Succeed())

			childDescMissing = model.NewFolder(testLib, "TestAudioSub_Parent/ChildDescMissing")
			childDescMissing.NumAudioFiles = 0
			Expect(repo.Put(childDescMissing)).To(Succeed())
			childDescMissingGrandchild = model.NewFolder(testLib, "TestAudioSub_Parent/ChildDescMissing/Sub")
			childDescMissingGrandchild.NumAudioFiles = 2
			childDescMissingGrandchild.Missing = true
			Expect(repo.Put(childDescMissingGrandchild)).To(Succeed())

			otherParentFolder = model.NewFolder(testLib, "TestAudioSub_OtherParent")
			otherParentFolder.NumAudioFiles = 0
			Expect(repo.Put(otherParentFolder)).To(Succeed())
			otherParentChild = model.NewFolder(testLib, "TestAudioSub_OtherParent/OtherChild")
			otherParentChild.NumAudioFiles = 5
			Expect(repo.Put(otherParentChild)).To(Succeed())

			DeferCleanup(func() {
				_, _ = conn.NewQuery("DELETE FROM folder WHERE name LIKE 'TestAudio%' OR path LIKE 'TestAudio%'").Execute()
				_, _ = conn.NewQuery("DELETE FROM folder WHERE path = '' AND name = '.'").Execute()
			})
		})

		Describe("GetRootSubfoldersWithAudio", func() {
			It("returns only root subfolders with audio in their subtree, sorted by name NOCASE", func() {
				folders, err := repo.GetRootSubfoldersWithAudio(testLib.ID)
				Expect(err).ToNot(HaveOccurred())

				ids := slice.Map(folders, func(f model.Folder) string { return f.ID })

				// Should include folders with audio directly or in descendants
				Expect(ids).To(ContainElements(
					rootWithDirect.ID,
					rootWithDesc.ID,
					rootWithGrandchild.ID,
					rootSpecialWildcard.ID,
				))

				// Should exclude empty, missing, or false-wildcard folders
				Expect(ids).ToNot(ContainElements(
					rootEmpty.ID,
					rootMissing.ID,
					rootDescMissing.ID,
					rootSpecialFalseMatch.ID,
				))

				// Should exclude folders from other library when filtering by testLib.ID
				Expect(ids).ToNot(ContainElement(rootOtherLib.ID))
			})

			It("filters by libraryIDs when provided", func() {
				folders, err := repo.GetRootSubfoldersWithAudio(otherLib.ID)
				Expect(err).ToNot(HaveOccurred())

				ids := slice.Map(folders, func(f model.Folder) string { return f.ID })
				Expect(ids).To(ConsistOf(rootOtherLib.ID))
			})

			It("sorts results by name COLLATE NOCASE ASC", func() {
				folders, err := repo.GetRootSubfoldersWithAudio(testLib.ID)
				Expect(err).ToNot(HaveOccurred())

				// Filter down to the three sort test folders
				var sortNames []string
				for _, f := range folders {
					if f.ID == rootSortA.ID || f.ID == rootSortB.ID || f.ID == rootSortC.ID {
						sortNames = append(sortNames, f.Name)
					}
				}
				Expect(sortNames).To(Equal([]string{
					"TestAudioSort_A",
					"TestAudioSort_b",
					"TestAudioSort_c",
				}))
			})
		})

		Describe("GetSubfoldersWithAudio", func() {
			It("returns direct children with audio directly or in descendants", func() {
				children, err := repo.GetSubfoldersWithAudio(parentFolder.ID)
				Expect(err).ToNot(HaveOccurred())

				ids := slice.Map(children, func(f model.Folder) string { return f.ID })

				// Should include child with direct audio and child with descendant audio
				Expect(ids).To(ConsistOf(childDirect.ID, childWithDesc.ID))

				// Should exclude child of other parent
				Expect(ids).ToNot(ContainElement(otherParentChild.ID))
			})

			It("filters by libraryIDs when provided", func() {
				// Asking with otherLib.ID should return empty slice since parent is in testLib
				children, err := repo.GetSubfoldersWithAudio(parentFolder.ID, otherLib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(children).To(BeEmpty())

				// Asking with testLib.ID should return the matching children
				children, err = repo.GetSubfoldersWithAudio(parentFolder.ID, testLib.ID)
				Expect(err).ToNot(HaveOccurred())
				ids := slice.Map(children, func(f model.Folder) string { return f.ID })
				Expect(ids).To(ConsistOf(childDirect.ID, childWithDesc.ID))
			})

			It("sorts children by name COLLATE NOCASE ASC", func() {
				sortParent := model.NewFolder(testLib, "TestAudioSortParent")
				Expect(repo.Put(sortParent)).To(Succeed())

				childB := model.NewFolder(testLib, "TestAudioSortParent/beta")
				childB.NumAudioFiles = 1
				Expect(repo.Put(childB)).To(Succeed())

				childA := model.NewFolder(testLib, "TestAudioSortParent/Alpha")
				childA.NumAudioFiles = 1
				Expect(repo.Put(childA)).To(Succeed())

				childC := model.NewFolder(testLib, "TestAudioSortParent/gamma")
				childC.NumAudioFiles = 1
				Expect(repo.Put(childC)).To(Succeed())

				children, err := repo.GetSubfoldersWithAudio(sortParent.ID)
				Expect(err).ToNot(HaveOccurred())
				names := slice.Map(children, func(f model.Folder) string { return f.Name })
				Expect(names).To(Equal([]string{"Alpha", "beta", "gamma"}))
			})

			It("returns empty slice when parentID has no children", func() {
				children, err := repo.GetSubfoldersWithAudio("non-existent-id")
				Expect(err).ToNot(HaveOccurred())
				Expect(children).To(BeEmpty())
			})
		})

		Describe("GetCoverArtForFolders", func() {
			var albumRepo model.AlbumRepository
			var mfRepo model.MediaFileRepository
			var alDirect, alDisc *model.Album
			var folDirect, folMultiDisc, folDisc1, folEmpty *model.Folder

			BeforeEach(func() {
				albumRepo = NewAlbumRepository(ctx, conn)
				mfRepo = NewMediaFileRepository(ctx, conn)

				alDirect = &model.Album{
					ID:        "test-cov-al-1",
					Name:      "Direct Album",
					LibraryID: testLib.ID,
					ItemImage: model.ItemImage{ImageHash: "hashdirect", ImageAbsent: false},
				}
				Expect(albumRepo.Put(alDirect)).To(Succeed())

				alDisc = &model.Album{
					ID:        "test-cov-al-2",
					Name:      "MultiDisc Album",
					LibraryID: testLib.ID,
					ItemImage: model.ItemImage{ImageHash: "hashdisc", ImageAbsent: false},
				}
				Expect(albumRepo.Put(alDisc)).To(Succeed())

				folDirect = model.NewFolder(testLib, "TestCover/Direct")
				folDirect.NumAudioFiles = 1
				Expect(repo.Put(folDirect)).To(Succeed())

				folMultiDisc = model.NewFolder(testLib, "TestCover/MultiDisc")
				Expect(repo.Put(folMultiDisc)).To(Succeed())

				folDisc1 = model.NewFolder(testLib, "TestCover/MultiDisc/CD1")
				folDisc1.NumAudioFiles = 1
				Expect(repo.Put(folDisc1)).To(Succeed())

				folEmpty = model.NewFolder(testLib, "TestCover/Empty")
				Expect(repo.Put(folEmpty)).To(Succeed())

				mfDirect := &model.MediaFile{
					ID:        "test-cov-mf-1",
					LibraryID: testLib.ID,
					AlbumID:   alDirect.ID,
					FolderID:  folDirect.ID,
					Path:      "TestCover/Direct/01.mp3",
				}
				Expect(mfRepo.Put(mfDirect)).To(Succeed())

				mfDisc := &model.MediaFile{
					ID:        "test-cov-mf-2",
					LibraryID: testLib.ID,
					AlbumID:   alDisc.ID,
					FolderID:  folDisc1.ID,
					Path:      "TestCover/MultiDisc/CD1/01.mp3",
				}
				Expect(mfRepo.Put(mfDisc)).To(Succeed())

				artworkRepo := NewArtworkRepository(ctx, conn)
				Expect(artworkRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: alDirect.ID, Hash: "hashdirect"})).To(Succeed())
				Expect(artworkRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: alDisc.ID, Hash: "hashdisc"})).To(Succeed())

				DeferCleanup(func() {
					_, _ = conn.NewQuery("DELETE FROM media_file WHERE id LIKE 'test-cov-mf-%'").Execute()
					_, _ = conn.NewQuery("DELETE FROM album WHERE id LIKE 'test-cov-al-%'").Execute()
					_, _ = conn.NewQuery("DELETE FROM item_artwork WHERE item_id LIKE 'test-cov-al-%'").Execute()
				})
			})

			It("returns coverArt for folders with direct audio files", func() {
				res, err := repo.GetCoverArtForFolders(folDirect.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveKeyWithValue(folDirect.ID, "al-test-cov-al-1_hashdirect"))
			})

			It("returns coverArt for multi-disc parent folders from disc subfolders", func() {
				res, err := repo.GetCoverArtForFolders(folMultiDisc.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveKeyWithValue(folMultiDisc.ID, "al-test-cov-al-2_hashdisc"))
			})

			It("does not return coverArt for empty folders", func() {
				res, err := repo.GetCoverArtForFolders(folEmpty.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(HaveKey(folEmpty.ID))
			})

			It("does not return coverArt for artist folders containing multiple albums", func() {
				folArtist := model.NewFolder(testLib, "TestCover/Artist")
				Expect(repo.Put(folArtist)).To(Succeed())

				folAlb1 := model.NewFolder(testLib, "TestCover/Artist/Album1")
				folAlb1.NumAudioFiles = 1
				Expect(repo.Put(folAlb1)).To(Succeed())

				folAlb2 := model.NewFolder(testLib, "TestCover/Artist/Album2")
				folAlb2.NumAudioFiles = 1
				Expect(repo.Put(folAlb2)).To(Succeed())

				mfAlb1 := &model.MediaFile{
					ID:        "test-cov-mf-alb1",
					LibraryID: testLib.ID,
					AlbumID:   alDirect.ID,
					FolderID:  folAlb1.ID,
					Path:      "TestCover/Artist/Album1/01.mp3",
				}
				Expect(mfRepo.Put(mfAlb1)).To(Succeed())

				mfAlb2 := &model.MediaFile{
					ID:        "test-cov-mf-alb2",
					LibraryID: testLib.ID,
					AlbumID:   alDisc.ID,
					FolderID:  folAlb2.ID,
					Path:      "TestCover/Artist/Album2/01.mp3",
				}
				Expect(mfRepo.Put(mfAlb2)).To(Succeed())

				res, err := repo.GetCoverArtForFolders(folArtist.ID, folAlb1.ID, folAlb2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).ToNot(HaveKey(folArtist.ID))
				Expect(res).To(HaveKeyWithValue(folAlb1.ID, "al-test-cov-al-1_hashdirect"))
				Expect(res).To(HaveKeyWithValue(folAlb2.ID, "al-test-cov-al-2_hashdisc"))
			})

			It("resolves multiple folders in a single call", func() {
				res, err := repo.GetCoverArtForFolders(folDirect.ID, folMultiDisc.ID, folEmpty.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(HaveLen(2))
				Expect(res).To(HaveKeyWithValue(folDirect.ID, "al-test-cov-al-1_hashdirect"))
				Expect(res).To(HaveKeyWithValue(folMultiDisc.ID, "al-test-cov-al-2_hashdisc"))
			})
		})
	})
})
