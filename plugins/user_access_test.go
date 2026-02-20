//go:build !windows

package plugins

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserAccess", func() {
	Describe("IsAllowed", func() {
		It("returns true when allUsers is true", func() {
			ua := NewUserAccess(true, nil)
			Expect(ua.IsAllowed("any-user")).To(BeTrue())
		})

		It("returns true when allUsers is true even with an explicit list", func() {
			ua := NewUserAccess(true, []string{"user-1"})
			Expect(ua.IsAllowed("other-user")).To(BeTrue())
		})

		It("returns false when userIDs is empty", func() {
			ua := NewUserAccess(false, []string{})
			Expect(ua.IsAllowed("user-1")).To(BeFalse())
		})

		It("returns false when userIDs is nil", func() {
			ua := NewUserAccess(false, nil)
			Expect(ua.IsAllowed("user-1")).To(BeFalse())
		})

		It("returns true when user is in the list", func() {
			ua := NewUserAccess(false, []string{"user-1", "user-2"})
			Expect(ua.IsAllowed("user-1")).To(BeTrue())
		})

		It("returns false when user is not in the list", func() {
			ua := NewUserAccess(false, []string{"user-1", "user-2"})
			Expect(ua.IsAllowed("user-3")).To(BeFalse())
		})
	})

	Describe("HasConfiguredUsers", func() {
		It("returns true when allUsers is true", func() {
			ua := NewUserAccess(true, nil)
			Expect(ua.HasConfiguredUsers()).To(BeTrue())
		})

		It("returns true when specific users are configured", func() {
			ua := NewUserAccess(false, []string{"user-1"})
			Expect(ua.HasConfiguredUsers()).To(BeTrue())
		})

		It("returns false when no users are configured", func() {
			ua := NewUserAccess(false, nil)
			Expect(ua.HasConfiguredUsers()).To(BeFalse())
		})

		It("returns false when user list is empty", func() {
			ua := NewUserAccess(false, []string{})
			Expect(ua.HasConfiguredUsers()).To(BeFalse())
		})
	})
})
