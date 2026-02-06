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
		r := newReq("getUser", "username", adminUser.UserName)
		resp, err := router.GetUser(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.User).ToNot(BeNil())
		Expect(resp.User.Username).To(Equal(adminUser.UserName))
		Expect(resp.User.AdminRole).To(BeTrue())
		Expect(resp.User.StreamRole).To(BeTrue())
		Expect(resp.User.Folder).ToNot(BeEmpty())
	})

	It("getUser with matching username case-insensitive succeeds", func() {
		r := newReq("getUser", "username", "Admin")
		resp, err := router.GetUser(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.User).ToNot(BeNil())
		Expect(resp.User.Username).To(Equal(adminUser.UserName))
	})

	It("getUser with different username returns authorization error", func() {
		r := newReq("getUser", "username", "otheruser")
		_, err := router.GetUser(r)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not authorized"))
	})

	It("getUsers returns list with current user only", func() {
		r := newReq("getUsers")
		resp, err := router.GetUsers(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Users).ToNot(BeNil())
		Expect(resp.Users.User).To(HaveLen(1))
		Expect(resp.Users.User[0].Username).To(Equal(adminUser.UserName))
		Expect(resp.Users.User[0].AdminRole).To(BeTrue())
	})
})
