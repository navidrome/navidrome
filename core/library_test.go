package core

import (
	"context"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Library Service", func() {
	var service Library
	var ds *tests.MockDataStore
	var libraryRepo *tests.MockLibraryRepo
	var userRepo *tests.MockedUserRepo
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		ds = &tests.MockDataStore{}
		libraryRepo = &tests.MockLibraryRepo{}
		userRepo = tests.CreateMockUserRepo()
		ds.MockedLibrary = libraryRepo
		ds.MockedUser = userRepo

		service = NewLibrary(ds)
		ctx = context.Background()
	})

	Describe("Library CRUD Operations", func() {
		Describe("GetAll", func() {
			It("returns all libraries", func() {
				libraries := model.Libraries{
					{ID: 1, Name: "Library 1", Path: "/music1"},
					{ID: 2, Name: "Library 2", Path: "/music2"},
				}
				libraryRepo.SetData(libraries)

				result, err := service.GetAll(ctx)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(2))
				Expect(result[0].Name).To(Equal("Library 1"))
				Expect(result[1].Name).To(Equal("Library 2"))
			})

			It("returns empty list when no libraries exist", func() {
				result, err := service.GetAll(ctx)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(0))
			})
		})

		Describe("Get", func() {
			It("returns a specific library", func() {
				library := &model.Library{ID: 1, Name: "Test Library", Path: "/music"}
				libraryRepo.SetData(model.Libraries{*library})

				result, err := service.Get(ctx, 1)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Name).To(Equal("Test Library"))
				Expect(result.Path).To(Equal("/music"))
			})

			It("returns error when library not found", func() {
				_, err := service.Get(ctx, 999)

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(model.ErrNotFound))
			})
		})

		Describe("Create", func() {
			It("creates a new library successfully", func() {
				library := &model.Library{ID: 1, Name: "New Library", Path: "/new/music"}

				err := service.Create(ctx, library)

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data[1].Name).To(Equal("New Library"))
				Expect(libraryRepo.Data[1].Path).To(Equal("/new/music"))
			})

			It("fails when library name is empty", func() {
				library := &model.Library{Path: "/music"}

				err := service.Create(ctx, library)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("library name is required"))
			})

			It("fails when library path is empty", func() {
				library := &model.Library{Name: "Test"}

				err := service.Create(ctx, library)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("library path is required"))
			})
		})

		Describe("Update", func() {
			BeforeEach(func() {
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Original Library", Path: "/original"},
				})
			})

			It("updates an existing library successfully", func() {
				library := &model.Library{ID: 1, Name: "Updated Library", Path: "/updated"}

				err := service.Update(ctx, library)

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data[1].Name).To(Equal("Updated Library"))
				Expect(libraryRepo.Data[1].Path).To(Equal("/updated"))
			})

			It("fails when library doesn't exist", func() {
				library := &model.Library{ID: 999, Name: "Non-existent", Path: "/path"}

				err := service.Update(ctx, library)

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(model.ErrNotFound))
			})

			It("fails when library name is empty", func() {
				library := &model.Library{ID: 1, Path: "/music"}

				err := service.Update(ctx, library)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("library name is required"))
			})
		})

		Describe("Delete", func() {
			BeforeEach(func() {
				libraryRepo.SetData(model.Libraries{
					{ID: 1, Name: "Library to Delete", Path: "/delete"},
				})
			})

			It("deletes an existing library successfully", func() {
				err := service.Delete(ctx, 1)

				Expect(err).NotTo(HaveOccurred())
				Expect(libraryRepo.Data).To(HaveLen(0))
			})

			It("fails when library doesn't exist", func() {
				err := service.Delete(ctx, 999)

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
				Expect(err.Error()).To(ContainSubstring("library ID 999 does not exist"))
			})

			It("fails when some libraries don't exist", func() {
				err := service.SetUserLibraries(ctx, "user1", []int{1, 999, 2})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("library ID 999 does not exist"))
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
})
