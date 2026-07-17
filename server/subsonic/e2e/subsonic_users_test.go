package e2e

import (
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	It("getUser returns current user info", func() {
		resp := doReq("getUser", "username", adminUser.UserName)

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.User).ToNot(BeNil())
		Expect(resp.User.Username).To(Equal(adminUser.UserName))
		Expect(resp.User.AdminRole).To(BeTrue())
		Expect(resp.User.StreamRole).To(BeTrue())
		Expect(resp.User.Folder).ToNot(BeEmpty())
	})

	It("getUser with matching username case-insensitive succeeds", func() {
		resp := doReq("getUser", "username", "Admin")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.User).ToNot(BeNil())
		Expect(resp.User.Username).To(Equal(adminUser.UserName))
	})

	It("getUser with different username returns authorization error", func() {
		resp := doReq("getUser", "username", "otheruser")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("getUsers returns list with current user only", func() {
		resp := doReq("getUsers")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Users).ToNot(BeNil())
		Expect(resp.Users.User).To(HaveLen(1))
		Expect(resp.Users.User[0].Username).To(Equal(adminUser.UserName))
		Expect(resp.Users.User[0].AdminRole).To(BeTrue())
	})
})
