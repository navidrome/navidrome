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
			r := newReqWithUser(regularUser, "startScan")
			_, err := router.StartScan(r)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not authorized"))
		})
	})

	Describe("Browsing as regular user", func() {
		It("regular user can browse the library", func() {
			r := newReqWithUser(regularUser, "getArtists")
			resp, err := router.GetArtists(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Artist).ToNot(BeNil())
			Expect(resp.Artist.Index).ToNot(BeEmpty())
		})

		It("regular user can search", func() {
			r := newReqWithUser(regularUser, "search3", "query", "Beatles")
			resp, err := router.Search3(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SearchResult3).ToNot(BeNil())
			Expect(resp.SearchResult3.Artist).ToNot(BeEmpty())
		})
	})

	Describe("getUser authorization", func() {
		It("regular user can get their own info", func() {
			r := newReqWithUser(regularUser, "getUser", "username", "regular")
			resp, err := router.GetUser(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.User.Username).To(Equal("regular"))
			Expect(resp.User.AdminRole).To(BeFalse())
		})

		It("regular user cannot get another user's info", func() {
			r := newReqWithUser(regularUser, "getUser", "username", "admin")
			_, err := router.GetUser(r)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not authorized"))
		})
	})

	Describe("getUsers for regular user", func() {
		It("returns only the requesting user's info", func() {
			r := newReqWithUser(regularUser, "getUsers")
			resp, err := router.GetUsers(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Users).ToNot(BeNil())
			Expect(resp.Users.User).To(HaveLen(1))
			Expect(resp.Users.User[0].Username).To(Equal("regular"))
			Expect(resp.Users.User[0].AdminRole).To(BeFalse())
		})
	})
})
