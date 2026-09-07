package model_test

import (
	"time"

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

	Describe("AvatarTag", func() {
		It("returns an empty tag when there is no avatar", func() {
			u := model.User{ID: "1", UpdatedAt: time.Now()}
			Expect(u.AvatarTag()).To(BeEmpty())
		})

		It("changes the tag when the image changes", func() {
			at := time.Unix(1000, 0)
			a := model.User{ID: "1", UploadedImage: "1_deluan.png", UpdatedAt: at}
			b := model.User{ID: "1", UploadedImage: "1_deluan.jpg", UpdatedAt: at}
			Expect(a.AvatarTag()).NotTo(BeEmpty())
			Expect(a.AvatarTag()).NotTo(Equal(b.AvatarTag()))
		})

		It("changes the tag when the same file is replaced", func() {
			a := model.User{ID: "1", UploadedImage: "1_deluan.png", UpdatedAt: time.Unix(1000, 0)}
			b := model.User{ID: "1", UploadedImage: "1_deluan.png", UpdatedAt: time.Unix(2000, 0)}
			Expect(a.AvatarTag()).NotTo(Equal(b.AvatarTag()))
		})
	})
})
