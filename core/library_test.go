package core_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/deluan/rest"
	_ "github.com/navidrome/navidrome/adapters/taglib" // Register taglib extractor
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	_ "github.com/navidrome/navidrome/core/storage/local" // Register local storage
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests require the local storage adapter and the taglib extractor to be registered.
var _ = Describe("Library Service", func() {
	var service core.Library
	var ds *tests.MockDataStore
	var libraryRepo *tests.MockLibraryRepo
	var userRepo *tests.MockedUserRepo
	var ctx context.Context
	var tempDir string
	var scanner *mockScanner
	var watcherManager *mockWatcherManager
	var broker *mockEventBroker

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		ds = &tests.MockDataStore{}
		libraryRepo = &tests.MockLibraryRepo{}
		userRepo = tests.CreateMockUserRepo()
		ds.MockedLibrary = libraryRepo
		ds.MockedUser = userRepo

		// Create a mock scanner that tracks calls
		scanner = &mockScanner{}
		// Create a mock watcher manager
		watcherManager = &mockWatcherManager{
			libraryStates: make(map[int]model.Library),
		}
		// Create a mock event broker
		broker = &mockEventBroker{}
		service = core.NewLibrary(ds, scanner, watcherManager, broker)
		ctx = context.Background()

		// Create a temporary directory for testing valid paths
		var err error
		tempDir, err = os.MkdirTemp("", "navidrome-library-test-")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			os.RemoveAll(tempDir)
		})
	})

	Describe("Library CRUD Operations", func() {
		var repo rest.Persistable

		BeforeEach(func() {
			r := service.NewRepository(ctx)
			repo = r.(rest.Persistable)
		})

		Describe("Create", func() {
			It("creates a new library successfully", func() {
				library := &model.Library{ID: 1, Name: "New Library", Path: tempDir}

				_, err := repo.Save(library)

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data[1].Name).To(Equal("New Library"))
				Expect(libraryRepo.Data[1].Path).To(Equal(tempDir))
			})

			It("fails when library name is empty", func() {
				library := &model.Library{Path: tempDir}

				_, err := repo.Save(library)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ra.validation.required"))
			})

			It("fails when library path is empty", func() {
				library := &model.Library{Name: "Test"}

				_, err := repo.Save(library)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ra.validation.required"))
			})

			It("fails when library path is not absolute", func() {
				library := &model.Library{Name: "Test", Path: "relative/path"}

				_, err := repo.Save(library)

				Expect(err).To(HaveOccurred())
				var validationErr *rest.ValidationError
				Expect(errors.As(err, &validationErr)).To(BeTrue())
				Expect(validationErr.Errors["path"]).To(Equal("library path must be absolute"))
			})

			Context("Database constraint violations", func() {
				BeforeEach(func() {
					// Set up an existing library that will cause constraint violations
					libraryRepo.SetData(model.Libraries{
						{ID: 1, Name: "Existing Library", Path: tempDir},
					})
				})

				AfterEach(func() {
					// Reset custom PutFn after each test
					libraryRepo.PutFn = nil
				})

				It("handles name uniqueness constraint violation from database", func() {
					// Create the directory that will be used for the test
					otherTempDir, err := os.MkdirTemp("", "navidrome-other-")
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func() { os.RemoveAll(otherTempDir) })

					// Try to create another library with the same name
					library := &model.Library{ID: 2, Name: "Existing Library", Path: otherTempDir}

					// Mock the repository to return a UNIQUE constraint error
					libraryRepo.PutFn = func(library *model.Library) error {
						return errors.New("UNIQUE constraint failed: library.name")
					}

					_, err = repo.Save(library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["name"]).To(Equal("ra.validation.unique"))
				})

				It("handles path uniqueness constraint violation from database", func() {
					// Try to create another library with the same path
					library := &model.Library{ID: 2, Name: "Different Library", Path: tempDir}

					// Mock the repository to return a UNIQUE constraint error
					libraryRepo.PutFn = func(library *model.Library) error {
						return errors.New("UNIQUE constraint failed: library.path")
					}

					_, err := repo.Save(library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("ra.validation.unique"))
				})
			})
		})

		Describe("Update", func() {
			BeforeEach(func() {
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Original Library", Path: tempDir},
				})
			})

			It("updates an existing library successfully", func() {
				newTempDir, err := os.MkdirTemp("", "navidrome-library-update-")
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { os.RemoveAll(newTempDir) })

				library := &model.Library{ID: 1, Name: "Updated Library", Path: newTempDir}

				err = repo.Update("1", library)

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data[1].Name).To(Equal("Updated Library"))
				Expect(libraryRepo.Data[1].Path).To(Equal(newTempDir))
			})

			It("fails when library doesn't exist", func() {
				// Create a unique temporary directory to avoid path conflicts
				uniqueTempDir, err := os.MkdirTemp("", "navidrome-nonexistent-")
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { os.RemoveAll(uniqueTempDir) })

				library := &model.Library{ID: 999, Name: "Non-existent", Path: uniqueTempDir}

				err = repo.Update("999", library)

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(model.ErrNotFound))
			})

			It("fails when library name is empty", func() {
				library := &model.Library{ID: 1, Path: tempDir}

				err := repo.Update("1", library)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ra.validation.required"))
			})

			It("cleans and normalizes the path on update", func() {
				unnormalizedPath := tempDir + "//../" + filepath.Base(tempDir)
				library := &model.Library{ID: 1, Name: "Updated Library", Path: unnormalizedPath}

				err := repo.Update("1", library)

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data[1].Path).To(Equal(filepath.Clean(unnormalizedPath)))
			})

			It("allows updating library with same name (no change)", func() {
				// Set up a library
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Test Library", Path: tempDir},
				})

				// Update the library keeping the same name (should be allowed)
				library := &model.Library{ID: 1, Name: "Test Library", Path: tempDir}

				err := repo.Update("1", library)

				Expect(err).NotTo(HaveOccurred())
			})

			It("allows updating library with same path (no change)", func() {
				// Set up a library
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Test Library", Path: tempDir},
				})

				// Update the library keeping the same path (should be allowed)
				library := &model.Library{ID: 1, Name: "Test Library", Path: tempDir}

				err := repo.Update("1", library)

				Expect(err).NotTo(HaveOccurred())
			})

			Context("Database constraint violations during update", func() {
				BeforeEach(func() {
					// Reset any custom PutFn from previous tests
					libraryRepo.PutFn = nil
				})

				It("handles name uniqueness constraint violation during update", func() {
					// Create additional temp directory for the test
					otherTempDir, err := os.MkdirTemp("", "navidrome-other-")
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func() { os.RemoveAll(otherTempDir) })

					// Set up two libraries
					libraryRepo.SetData(model.Libraries{
						{ID: 1, Name: "Library One", Path: tempDir},
						{ID: 2, Name: "Library Two", Path: otherTempDir},
					})

					// Mock database constraint violation
					libraryRepo.PutFn = func(library *model.Library) error {
						return errors.New("UNIQUE constraint failed: library.name")
					}

					// Try to update library 2 to have the same name as library 1
					library := &model.Library{ID: 2, Name: "Library One", Path: otherTempDir}

					err = repo.Update("2", library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["name"]).To(Equal("ra.validation.unique"))
				})

				It("handles path uniqueness constraint violation during update", func() {
					// Create additional temp directory for the test
					otherTempDir, err := os.MkdirTemp("", "navidrome-other-")
					Expect(err).NotTo(HaveOccurred())
					DeferCleanup(func() { os.RemoveAll(otherTempDir) })

					// Set up two libraries
					libraryRepo.SetData(model.Libraries{
						{ID: 1, Name: "Library One", Path: tempDir},
						{ID: 2, Name: "Library Two", Path: otherTempDir},
					})

					// Mock database constraint violation
					libraryRepo.PutFn = func(library *model.Library) error {
						return errors.New("UNIQUE constraint failed: library.path")
					}

					// Try to update library 2 to have the same path as library 1
					library := &model.Library{ID: 2, Name: "Library Two", Path: tempDir}

					err = repo.Update("2", library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("ra.validation.unique"))
				})
			})
		})

		Describe("Path Validation", func() {
			Context("Create operation", func() {
				It("fails when path is not absolute", func() {
					library := &model.Library{Name: "Test", Path: "relative/path"}

					_, err := repo.Save(library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("library path must be absolute"))
				})

				It("fails when path does not exist", func() {
					nonExistentPath := filepath.Join(tempDir, "nonexistent")
					library := &model.Library{Name: "Test", Path: nonExistentPath}

					_, err := repo.Save(library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("resources.library.validation.pathInvalid"))
				})

				It("fails when path is a file instead of directory", func() {
					testFile := filepath.Join(tempDir, "testfile.txt")
					err := os.WriteFile(testFile, []byte("test"), 0600)
					Expect(err).NotTo(HaveOccurred())

					library := &model.Library{Name: "Test", Path: testFile}

					_, err = repo.Save(library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("resources.library.validation.pathNotDirectory"))
				})

				It("fails when path is not accessible due to permissions", func() {
					Skip("Permission tests are environment-dependent and may fail in CI")
					// This test is skipped because creating a directory with no read permissions
					// is complex and may not work consistently across different environments
				})

				It("handles multiple validation errors", func() {
					library := &model.Library{Name: "", Path: "relative/path"}

					_, err := repo.Save(library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors).To(HaveKey("name"))
					Expect(validationErr.Errors).To(HaveKey("path"))
					Expect(validationErr.Errors["name"]).To(Equal("ra.validation.required"))
					Expect(validationErr.Errors["path"]).To(Equal("library path must be absolute"))
				})
			})

			Context("Update operation", func() {
				BeforeEach(func() {
					libraryRepo.SetData(model.Libraries{
						{ID: 1, Name: "Test Library", Path: tempDir},
					})
				})

				It("fails when updated path is not absolute", func() {
					library := &model.Library{ID: 1, Name: "Test", Path: "relative/path"}

					err := repo.Update("1", library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("library path must be absolute"))
				})

				It("allows updating library with same name (no change)", func() {
					// Set up a library
					libraryRepo.SetData(model.Libraries{
						{ID: 1, Name: "Test Library", Path: tempDir},
					})

					// Update the library keeping the same name (should be allowed)
					library := &model.Library{ID: 1, Name: "Test Library", Path: tempDir}

					err := repo.Update("1", library)

					Expect(err).NotTo(HaveOccurred())
				})

				It("fails when updated path does not exist", func() {
					nonExistentPath := filepath.Join(tempDir, "nonexistent")
					library := &model.Library{ID: 1, Name: "Test", Path: nonExistentPath}

					err := repo.Update("1", library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("resources.library.validation.pathInvalid"))
				})

				It("fails when updated path is a file instead of directory", func() {
					testFile := filepath.Join(tempDir, "updatefile.txt")
					err := os.WriteFile(testFile, []byte("test"), 0600)
					Expect(err).NotTo(HaveOccurred())

					library := &model.Library{ID: 1, Name: "Test", Path: testFile}

					err = repo.Update("1", library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors["path"]).To(Equal("resources.library.validation.pathNotDirectory"))
				})

				It("handles multiple validation errors on update", func() {
					// Try to update with empty name and invalid path
					library := &model.Library{ID: 1, Name: "", Path: "relative/path"}

					err := repo.Update("1", library)

					Expect(err).To(HaveOccurred())
					var validationErr *rest.ValidationError
					Expect(errors.As(err, &validationErr)).To(BeTrue())
					Expect(validationErr.Errors).To(HaveKey("name"))
					Expect(validationErr.Errors).To(HaveKey("path"))
					Expect(validationErr.Errors["name"]).To(Equal("ra.validation.required"))
					Expect(validationErr.Errors["path"]).To(Equal("library path must be absolute"))
				})
			})
		})

		Describe("Delete", func() {
			BeforeEach(func() {
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Library to Delete", Path: tempDir},
				})
			})

			It("deletes an existing library successfully", func() {
				err := repo.Delete("1")

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data).To(HaveLen(0))
			})

			It("fails when library doesn't exist", func() {
				err := repo.Delete("999")

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(model.ErrNotFound))
			})
		})
	})

	Describe("User-Library Association Operations", func() {
		var regularUser, adminUser *model.User

		BeforeEach(func() {
			regularUser = &model.User{ID: "user1", UserName: "regular", IsAdmin: false}
			adminUser = &model.User{ID: "admin1", UserName: "admin", IsAdmin: true}

			userRepo.Data = map[string]*model.User{
				"regular": regularUser,
				"admin":   adminUser,
			}
			libraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Library 1", Path: "/music1"},
				{ID: 2, Name: "Library 2", Path: "/music2"},
				{ID: 3, Name: "Library 3", Path: "/music3"},
			})
		})

		Describe("GetUserLibraries", func() {
			It("returns user's libraries", func() {
				userRepo.UserLibraries = map[string][]int{
					"user1": {1},
				}

				result, err := service.GetUserLibraries(ctx, "user1")

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].ID).To(Equal(1))
			})

			It("fails when user doesn't exist", func() {
				_, err := service.GetUserLibraries(ctx, "nonexistent")

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(model.ErrNotFound))
			})
		})

		Describe("SetUserLibraries", func() {
			It("sets libraries for regular user successfully", func() {
				err := service.SetUserLibraries(ctx, "user1", []int{1, 2})

				Expect(err).NotTo(HaveOccurred())
				libraries := userRepo.UserLibraries["user1"]
				Expect(libraries).To(HaveLen(2))
			})

			It("fails when user doesn't exist", func() {
				err := service.SetUserLibraries(ctx, "nonexistent", []int{1})

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(model.ErrNotFound))
			})

			It("fails when trying to set libraries for admin user", func() {
				err := service.SetUserLibraries(ctx, "admin1", []int{1})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cannot manually assign libraries to admin users"))
			})

			It("fails when no libraries provided for regular user", func() {
				err := service.SetUserLibraries(ctx, "user1", []int{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least one library must be assigned to non-admin users"))
			})

			It("fails when library doesn't exist", func() {
				err := service.SetUserLibraries(ctx, "user1", []int{999})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("one or more library IDs are invalid"))
			})

			It("fails when some libraries don't exist", func() {
				err := service.SetUserLibraries(ctx, "user1", []int{1, 999, 2})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("one or more library IDs are invalid"))
			})
		})

		Describe("ValidateLibraryAccess", func() {
			Context("admin user", func() {
				BeforeEach(func() {
					ctx = request.WithUser(ctx, *adminUser)
				})

				It("allows access to any library", func() {
					err := service.ValidateLibraryAccess(ctx, "admin1", 1)

					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("regular user", func() {
				BeforeEach(func() {
					ctx = request.WithUser(ctx, *regularUser)
					userRepo.UserLibraries = map[string][]int{
						"user1": {1},
					}
				})

				It("allows access to user's libraries", func() {
					err := service.ValidateLibraryAccess(ctx, "user1", 1)

					Expect(err).NotTo(HaveOccurred())
				})

				It("denies access to libraries user doesn't have", func() {
					err := service.ValidateLibraryAccess(ctx, "user1", 2)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("user does not have access to library 2"))
				})
			})

			Context("no user in context", func() {
				It("fails with user not found error", func() {
					err := service.ValidateLibraryAccess(ctx, "user1", 1)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("user not found in context"))
				})
			})
		})
	})

	Describe("Scan Triggering", func() {
		var repo rest.Persistable

		BeforeEach(func() {
			r := service.NewRepository(ctx)
			repo = r.(rest.Persistable)
		})

		It("triggers scan when creating a new library", func() {
			library := &model.Library{ID: 1, Name: "New Library", Path: tempDir}

			_, err := repo.Save(library)
			Expect(err).NotTo(HaveOccurred())

			// Wait briefly for the goroutine to complete
			Eventually(func() int {
				return scanner.len()
			}, "1s", "10ms").Should(Equal(1))

			// Verify scan was called with correct parameters
			Expect(scanner.ScanCalls[0].FullScan).To(BeFalse()) // Should be quick scan
		})

		It("triggers scan when updating library path", func() {
			// First create a library
			libraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Original Library", Path: tempDir},
			})

			// Create a new temporary directory for the update
			newTempDir, err := os.MkdirTemp("", "navidrome-library-update-")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { os.RemoveAll(newTempDir) })

			// Update the library with a new path
			library := &model.Library{ID: 1, Name: "Updated Library", Path: newTempDir}
			err = repo.Update("1", library)
			Expect(err).NotTo(HaveOccurred())

			// Wait briefly for the goroutine to complete
			Eventually(func() int {
				return scanner.len()
			}, "1s", "10ms").Should(Equal(1))

			// Verify scan was called with correct parameters
			Expect(scanner.ScanCalls[0].FullScan).To(BeFalse()) // Should be quick scan
		})

		It("does not trigger scan when updating library without path change", func() {
			// First create a library
			libraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Original Library", Path: tempDir},
			})

			// Update the library name only (same path)
			library := &model.Library{ID: 1, Name: "Updated Name", Path: tempDir}
			err := repo.Update("1", library)
			Expect(err).NotTo(HaveOccurred())

			// Wait a bit to ensure no scan was triggered
			Consistently(func() int {
				return scanner.len()
			}, "100ms", "10ms").Should(Equal(0))
		})

		It("does not trigger scan when library creation fails", func() {
			// Try to create library with invalid data (empty name)
			library := &model.Library{Path: tempDir}

			_, err := repo.Save(library)
			Expect(err).To(HaveOccurred())

			// Ensure no scan was triggered since creation failed
			Consistently(func() int {
				return scanner.len()
			}, "100ms", "10ms").Should(Equal(0))
		})

		It("does not trigger scan when library update fails", func() {
			// First create a library
			libraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Original Library", Path: tempDir},
			})

			// Try to update with invalid data (empty name)
			library := &model.Library{ID: 1, Name: "", Path: tempDir}
			err := repo.Update("1", library)
			Expect(err).To(HaveOccurred())

			// Ensure no scan was triggered since update failed
			Consistently(func() int {
				return scanner.len()
			}, "100ms", "10ms").Should(Equal(0))
		})

		It("triggers scan when deleting a library", func() {
			// First create a library
			libraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Library to Delete", Path: tempDir},
			})

			// Delete the library
			err := repo.Delete("1")
			Expect(err).NotTo(HaveOccurred())

			// Wait briefly for the goroutine to complete
			Eventually(func() int {
				return scanner.len()
			}, "1s", "10ms").Should(Equal(1))

			// Verify scan was called with correct parameters
			Expect(scanner.ScanCalls[0].FullScan).To(BeFalse()) // Should be quick scan
		})

		It("does not trigger scan when library deletion fails", func() {
			// Try to delete a non-existent library
			err := repo.Delete("999")
			Expect(err).To(HaveOccurred())

			// Ensure no scan was triggered since deletion failed
			Consistently(func() int {
				return scanner.len()
			}, "100ms", "10ms").Should(Equal(0))
		})

		Context("Watcher Integration", func() {
			It("starts watcher when creating a new library", func() {
				library := &model.Library{ID: 1, Name: "New Library", Path: tempDir}

				_, err := repo.Save(library)
				Expect(err).NotTo(HaveOccurred())

				// Verify watcher was started
				Eventually(func() int {
					return watcherManager.lenStarted()
				}, "1s", "10ms").Should(Equal(1))

				Expect(watcherManager.StartedWatchers[0].ID).To(Equal(1))
				Expect(watcherManager.StartedWatchers[0].Name).To(Equal("New Library"))
				Expect(watcherManager.StartedWatchers[0].Path).To(Equal(tempDir))
			})

			It("restarts watcher when library path is updated", func() {
				// First create a library
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Original Library", Path: tempDir},
				})

				// Simulate that this library already has a watcher
				watcherManager.simulateExistingLibrary(model.Library{ID: 1, Name: "Original Library", Path: tempDir})

				// Create a new temp directory for the update
				newTempDir, err := os.MkdirTemp("", "navidrome-library-update-")
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { os.RemoveAll(newTempDir) })

				// Update library with new path
				library := &model.Library{ID: 1, Name: "Updated Library", Path: newTempDir}
				err = repo.Update("1", library)
				Expect(err).NotTo(HaveOccurred())

				// Verify watcher was restarted
				Eventually(func() int {
					return watcherManager.lenRestarted()
				}, "1s", "10ms").Should(Equal(1))

				Expect(watcherManager.RestartedWatchers[0].ID).To(Equal(1))
				Expect(watcherManager.RestartedWatchers[0].Path).To(Equal(newTempDir))
			})

			It("does not restart watcher when only library name is updated", func() {
				// First create a library
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Original Library", Path: tempDir},
				})

				// Update library with same path but different name
				library := &model.Library{ID: 1, Name: "Updated Name", Path: tempDir}
				err := repo.Update("1", library)
				Expect(err).NotTo(HaveOccurred())

				// Verify watcher was NOT restarted (since path didn't change)
				Consistently(func() int {
					return watcherManager.lenRestarted()
				}, "100ms", "10ms").Should(Equal(0))
			})

			It("stops watcher when library is deleted", func() {
				// Set up a library
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Test Library", Path: tempDir},
				})

				err := repo.Delete("1")
				Expect(err).NotTo(HaveOccurred())

				// Verify watcher was stopped
				Eventually(func() int {
					return watcherManager.lenStopped()
				}, "1s", "10ms").Should(Equal(1))

				Expect(watcherManager.StoppedWatchers[0]).To(Equal(1))
			})

			It("does not stop watcher when library deletion fails", func() {
				// Set up a library
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Test Library", Path: tempDir},
				})

				// Mock deletion to fail by trying to delete non-existent library
				err := repo.Delete("999")
				Expect(err).To(HaveOccurred())

				// Verify watcher was NOT stopped since deletion failed
				Consistently(func() int {
					return watcherManager.lenStopped()
				}, "100ms", "10ms").Should(Equal(0))
			})
		})
	})

	Describe("Event Broadcasting", func() {
		var repo rest.Persistable

		BeforeEach(func() {
			r := service.NewRepository(ctx)
			repo = r.(rest.Persistable)
			// Clear any events from broker
			broker.Events = []events.Event{}
		})

		It("sends refresh event when creating a library", func() {
			library := &model.Library{ID: 1, Name: "New Library", Path: tempDir}

			_, err := repo.Save(library)

			Expect(err).NotTo(HaveOccurred())
			Expect(broker.Events).To(HaveLen(1))
		})

		It("sends refresh event when updating a library", func() {
			// First create a library
			libraryRepo.SetData(model.Libraries{
				{ID: 1, Name: "Original Library", Path: tempDir},
			})

			library := &model.Library{ID: 1, Name: "Updated Library", Path: tempDir}
			err := repo.Update("1", library)

			Expect(err).NotTo(HaveOccurred())
			Expect(broker.Events).To(HaveLen(1))
		})

		It("sends refresh event when deleting a library", func() {
			// First create a library
			libraryRepo.SetData(model.Libraries{
				{ID: 2, Name: "Library to Delete", Path: tempDir},
			})

			err := repo.Delete("2")

			Expect(err).NotTo(HaveOccurred())
			Expect(broker.Events).To(HaveLen(1))
		})
	})
})

// mockScanner provides a simple mock implementation of core.Scanner for testing
type mockScanner struct {
	ScanCalls []ScanCall
	mu        sync.RWMutex
}

type ScanCall struct {
	FullScan bool
}

func (m *mockScanner) ScanAll(ctx context.Context, fullScan bool) (warnings []string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ScanCalls = append(m.ScanCalls, ScanCall{
		FullScan: fullScan,
	})
	return []string{}, nil
}

func (m *mockScanner) len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.ScanCalls)
}

// mockWatcherManager provides a simple mock implementation of core.Watcher for testing
type mockWatcherManager struct {
	StartedWatchers   []model.Library
	StoppedWatchers   []int
	RestartedWatchers []model.Library
	libraryStates     map[int]model.Library // Track which libraries we know about
	mu                sync.RWMutex
}

func (m *mockWatcherManager) Watch(ctx context.Context, lib *model.Library) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we already know about this library ID
	if _, exists := m.libraryStates[lib.ID]; exists {
		// This is a restart - the library already existed
		// Update our tracking and record the restart
		for i, startedLib := range m.StartedWatchers {
			if startedLib.ID == lib.ID {
				m.StartedWatchers[i] = *lib
				break
			}
		}
		m.RestartedWatchers = append(m.RestartedWatchers, *lib)
		m.libraryStates[lib.ID] = *lib
		return nil
	}

	// This is a new library - first time we're seeing it
	m.StartedWatchers = append(m.StartedWatchers, *lib)
	m.libraryStates[lib.ID] = *lib
	return nil
}

func (m *mockWatcherManager) StopWatching(ctx context.Context, libraryID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StoppedWatchers = append(m.StoppedWatchers, libraryID)
	return nil
}

func (m *mockWatcherManager) lenStarted() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.StartedWatchers)
}

func (m *mockWatcherManager) lenStopped() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.StoppedWatchers)
}

func (m *mockWatcherManager) lenRestarted() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.RestartedWatchers)
}

// simulateExistingLibrary simulates the scenario where a library already exists
// and has a watcher running (used by tests to set up the initial state)
func (m *mockWatcherManager) simulateExistingLibrary(lib model.Library) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.libraryStates[lib.ID] = lib
}

// mockEventBroker provides a mock implementation of events.Broker for testing
type mockEventBroker struct {
	http.Handler
	Events []events.Event
	mu     sync.RWMutex
}

func (m *mockEventBroker) SendMessage(ctx context.Context, event events.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, event)
}

func (m *mockEventBroker) SendBroadcastMessage(ctx context.Context, event events.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, event)
}
