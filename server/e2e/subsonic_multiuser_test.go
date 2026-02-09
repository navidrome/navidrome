package e2e

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multi-User Isolation", Ordered, func() {
	var regularUser model.User

	BeforeAll(func() {
		setupTestDB()

		regularUser = createUser("regular-1", "regular", "Regular User", false)
	})

	Describe("Admin-only endpoint restrictions", func() {
		It("startScan fails for regular user", func() {
			resp := doReqWithUser(regularUser, "startScan")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("Browsing as regular user", func() {
		It("regular user can browse the library", func() {
			resp := doReqWithUser(regularUser, "getArtists")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Artist).ToNot(BeNil())
			Expect(resp.Artist.Index).ToNot(BeEmpty())
		})

		It("regular user can search", func() {
			resp := doReqWithUser(regularUser, "search3", "query", "Beatles")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SearchResult3).ToNot(BeNil())
			Expect(resp.SearchResult3.Artist).ToNot(BeEmpty())
		})
	})

	Describe("getUser authorization", func() {
		It("regular user can get their own info", func() {
			resp := doReqWithUser(regularUser, "getUser", "username", "regular")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.User.Username).To(Equal("regular"))
			Expect(resp.User.AdminRole).To(BeFalse())
		})

		It("regular user cannot get another user's info", func() {
			resp := doReqWithUser(regularUser, "getUser", "username", "admin")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("getUsers for regular user", func() {
		It("returns only the requesting user's info", func() {
			resp := doReqWithUser(regularUser, "getUsers")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Users).ToNot(BeNil())
			Expect(resp.Users.User).To(HaveLen(1))
			Expect(resp.Users.User[0].Username).To(Equal("regular"))
			Expect(resp.Users.User[0].AdminRole).To(BeFalse())
		})
	})
})
