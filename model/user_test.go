package model_test

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User", func() {
	var user model.User
	var libraries model.Libraries

	BeforeEach(func() {
		libraries = model.Libraries{
			{ID: 1, Name: "Rock Library", Path: "/music/rock"},
			{ID: 2, Name: "Jazz Library", Path: "/music/jazz"},
			{ID: 3, Name: "Classical Library", Path: "/music/classical"},
		}

		user = model.User{
			ID:        "user1",
			UserName:  "testuser",
			Name:      "Test User",
			Email:     "test@example.com",
			IsAdmin:   false,
			Libraries: libraries,
		}
	})

	Describe("HasLibraryAccess", func() {
		Context("when user is admin", func() {
			BeforeEach(func() {
				user.IsAdmin = true
			})

			It("returns true for any library ID", func() {
				Expect(user.HasLibraryAccess(1)).To(BeTrue())
				Expect(user.HasLibraryAccess(99)).To(BeTrue())
				Expect(user.HasLibraryAccess(-1)).To(BeTrue())
			})

			It("returns true even when user has no libraries assigned", func() {
				user.Libraries = nil
				Expect(user.HasLibraryAccess(1)).To(BeTrue())
			})
		})

		Context("when user is not admin", func() {
			BeforeEach(func() {
				user.IsAdmin = false
			})

			It("returns true for libraries the user has access to", func() {
				Expect(user.HasLibraryAccess(1)).To(BeTrue())
				Expect(user.HasLibraryAccess(2)).To(BeTrue())
				Expect(user.HasLibraryAccess(3)).To(BeTrue())
			})

			It("returns false for libraries the user does not have access to", func() {
				Expect(user.HasLibraryAccess(4)).To(BeFalse())
				Expect(user.HasLibraryAccess(99)).To(BeFalse())
				Expect(user.HasLibraryAccess(-1)).To(BeFalse())
				Expect(user.HasLibraryAccess(0)).To(BeFalse())
			})

			It("returns false when user has no libraries assigned", func() {
				user.Libraries = nil
				Expect(user.HasLibraryAccess(1)).To(BeFalse())
			})

			It("handles duplicate library IDs correctly", func() {
				user.Libraries = model.Libraries{
					{ID: 1, Name: "Library 1", Path: "/music1"},
					{ID: 1, Name: "Library 1 Duplicate", Path: "/music1-dup"},
					{ID: 2, Name: "Library 2", Path: "/music2"},
				}
				Expect(user.HasLibraryAccess(1)).To(BeTrue())
				Expect(user.HasLibraryAccess(2)).To(BeTrue())
				Expect(user.HasLibraryAccess(3)).To(BeFalse())
			})
		})
	})

	Describe("FilteredLibraries", func() {
		It("returns only libraries that match the filter", func() {
			filtered := user.FilteredLibraries([]int{1, 3})
			expected := model.Libraries{
				{ID: 1, Name: "Rock Library", Path: "/music/rock"},
				{ID: 3, Name: "Classical Library", Path: "/music/classical"},
			}
			Expect(filtered).To(Equal(expected))
		})

		It("returns single library when filter contains one matching ID", func() {
			filtered := user.FilteredLibraries([]int{2})
			expected := model.Libraries{
				{ID: 2, Name: "Jazz Library", Path: "/music/jazz"},
			}
			Expect(filtered).To(Equal(expected))
		})

		It("returns nil slice when filter has no matching IDs", func() {
			filtered := user.FilteredLibraries([]int{99, 100})
			Expect(filtered).To(BeNil())
		})

		It("returns nil slice when filter is empty", func() {
			filtered := user.FilteredLibraries([]int{})
			Expect(filtered).To(BeNil())
		})

		It("returns nil slice when filter is nil", func() {
			filtered := user.FilteredLibraries(nil)
			Expect(filtered).To(BeNil())
		})

		It("returns nil slice when user has no libraries", func() {
			user.Libraries = nil
			filtered := user.FilteredLibraries([]int{1, 2, 3})
			Expect(filtered).To(BeNil())
		})

		It("returns nil slice when user has empty libraries", func() {
			user.Libraries = model.Libraries{}
			filtered := user.FilteredLibraries([]int{1, 2, 3})
			Expect(filtered).To(BeNil())
		})

		It("handles partial matches correctly", func() {
			filtered := user.FilteredLibraries([]int{1, 99, 2, 100})
			expected := model.Libraries{
				{ID: 1, Name: "Rock Library", Path: "/music/rock"},
				{ID: 2, Name: "Jazz Library", Path: "/music/jazz"},
			}
			Expect(filtered).To(Equal(expected))
		})
	})

	Describe("AccessibleLibraryIDs", func() {
		It("returns all library IDs when user has libraries", func() {
			ids := user.AccessibleLibraryIDs()
			expected := []int{1, 2, 3}
			Expect(ids).To(Equal(expected))
		})

		It("returns nil when user has no libraries", func() {
			user.Libraries = nil
			ids := user.AccessibleLibraryIDs()
			Expect(ids).To(BeNil())
		})
	})
})
