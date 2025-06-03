package subsonic

import (
	"context"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Users", func() {
	var router *Router
	var testUser model.User

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		router = &Router{}

		testUser = model.User{
			ID:       "user123",
			UserName: "testuser",
			Name:     "Test User",
			Email:    "test@example.com",
			IsAdmin:  false,
		}
	})

	Describe("Happy path", func() {
		It("should return consistent user data in both GetUser and GetUsers", func() {
			conf.Server.EnableDownloads = true
			conf.Server.EnableSharing = true
			conf.Server.Jukebox.Enabled = false

			// Create request with user in context
			req := httptest.NewRequest("GET", "/rest/getUser", nil)
			ctx := request.WithUser(context.Background(), testUser)
			req = req.WithContext(ctx)

			userResponse, err1 := router.GetUser(req)
			usersResponse, err2 := router.GetUsers(req)

			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())

			// Verify GetUser response structure
			Expect(userResponse.Status).To(Equal(responses.StatusOK))
			Expect(userResponse.User).ToNot(BeNil())
			Expect(userResponse.User.Username).To(Equal("testuser"))
			Expect(userResponse.User.Email).To(Equal("test@example.com"))
			Expect(userResponse.User.AdminRole).To(BeFalse())
			Expect(userResponse.User.StreamRole).To(BeTrue())
			Expect(userResponse.User.ScrobblingEnabled).To(BeTrue())
			Expect(userResponse.User.DownloadRole).To(BeTrue())
			Expect(userResponse.User.ShareRole).To(BeTrue())

			// Verify GetUsers response structure
			Expect(usersResponse.Status).To(Equal(responses.StatusOK))
			Expect(usersResponse.Users).ToNot(BeNil())
			Expect(usersResponse.Users.User).To(HaveLen(1))

			// Verify both methods return identical user data
			singleUser := userResponse.User
			userFromList := &usersResponse.Users.User[0]

			Expect(singleUser.Username).To(Equal(userFromList.Username))
			Expect(singleUser.Email).To(Equal(userFromList.Email))
			Expect(singleUser.AdminRole).To(Equal(userFromList.AdminRole))
			Expect(singleUser.StreamRole).To(Equal(userFromList.StreamRole))
			Expect(singleUser.ScrobblingEnabled).To(Equal(userFromList.ScrobblingEnabled))
			Expect(singleUser.DownloadRole).To(Equal(userFromList.DownloadRole))
			Expect(singleUser.ShareRole).To(Equal(userFromList.ShareRole))
			Expect(singleUser.JukeboxRole).To(Equal(userFromList.JukeboxRole))
		})
	})

	DescribeTable("Jukebox role permissions",
		func(jukeboxEnabled, adminOnly, isAdmin, expectedJukeboxRole bool) {
			conf.Server.Jukebox.Enabled = jukeboxEnabled
			conf.Server.Jukebox.AdminOnly = adminOnly
			testUser.IsAdmin = isAdmin

			response := buildUserResponse(testUser)
			Expect(response.JukeboxRole).To(Equal(expectedJukeboxRole))
		},
		Entry("jukebox disabled", false, false, false, false),
		Entry("jukebox enabled, not admin-only, regular user", true, false, false, true),
		Entry("jukebox enabled, not admin-only, admin user", true, false, true, true),
		Entry("jukebox enabled, admin-only, regular user", true, true, false, false),
		Entry("jukebox enabled, admin-only, admin user", true, true, true, true),
	)
})
