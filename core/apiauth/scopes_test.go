package apiauth

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("scopes", func() {
	BeforeEach(func() {
		saved := KnownScopes
		KnownScopes = []string{ScopeRead, ScopePassword, ScopeAdmin, "playlists", "playlists:write"}
		DeferCleanup(func() { KnownScopes = saved })
	})

	Describe("Entitled", func() {
		It("stores all when nothing is requested", func() {
			Expect(Entitled(nil, false)).To(Equal([]string{ScopeAll}))
		})
		It("drops unknown scopes and admin for non-admins", func() {
			Expect(Entitled([]string{"read", "future", "admin"}, false)).To(Equal([]string{"read"}))
		})
		It("keeps admin for admins and keeps all", func() {
			Expect(Entitled([]string{"admin", "all"}, true)).To(Equal([]string{"admin", "all"}))
		})
	})

	Describe("Expand", func() {
		It("replaces all with every known scope except admin for non-admins", func() {
			Expect(Expand([]string{ScopeAll}, false)).To(Equal([]string{"password", "playlists", "playlists:write", "read"}))
		})
		It("includes admin for admins", func() {
			Expect(Expand([]string{ScopeAll}, true)).To(ContainElement("admin"))
		})
		It("drops admin from explicit scopes when the user is no longer an admin", func() {
			Expect(Expand([]string{"admin", "read"}, false)).To(Equal([]string{"read"}))
		})
		It("drops scopes that are no longer known", func() {
			Expect(Expand([]string{"read", "retired"}, false)).To(Equal([]string{"read"}))
		})
	})

	Describe("Allowed", func() {
		It("never widens all", func() {
			Expect(Allowed([]string{ScopeAll}, true)).To(BeEmpty())
		})
		It("keeps known scopes, and admin only for admins", func() {
			Expect(Allowed([]string{"read", "retired", "admin"}, false)).To(Equal([]string{"read"}))
			Expect(Allowed([]string{"read", "admin"}, true)).To(Equal([]string{"admin", "read"}))
		})
	})

	Describe("Attenuate", func() {
		available := []string{"playlists:write", "read"}
		It("returns everything when no subset is asked", func() {
			Expect(Attenuate(available, nil)).To(Equal([]string{"playlists:write", "read"}))
		})
		It("returns nothing for an explicit empty request", func() {
			Expect(Attenuate(available, []string{})).To(BeEmpty())
		})
		It("returns the overlap and drops unknown scopes", func() {
			Expect(Attenuate(available, []string{"read", "sync"})).To(Equal([]string{"read"}))
		})
		It("grants the base scope when only its :write form is available", func() {
			Expect(Attenuate(available, []string{"playlists"})).To(Equal([]string{"playlists"}))
		})
	})

	Describe("Satisfies", func() {
		It("accepts the exact scope or its :write form", func() {
			Expect(Satisfies([]string{"read"}, "read")).To(BeTrue())
			Expect(Satisfies([]string{"playlists:write"}, "playlists")).To(BeTrue())
			Expect(Satisfies([]string{"playlists"}, "playlists:write")).To(BeFalse())
			Expect(Satisfies(nil, "read")).To(BeFalse())
		})
	})
})
