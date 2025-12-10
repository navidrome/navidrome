package subsonic

import (
	"errors"
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

			// Set up user with libraries
			testUser.Libraries = model.Libraries{
				{ID: 10, Name: "Music"},
				{ID: 20, Name: "Podcasts"},
			}

			// Create request with user in context
			req := httptest.NewRequest("GET", "/rest/getUser?username=testuser", nil)
			ctx := request.WithUser(GinkgoT().Context(), testUser)
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
			Expect(userResponse.User.Folder).To(ContainElements(int32(10), int32(20)))

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
			Expect(singleUser.Folder).To(Equal(userFromList.Folder))
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

	Describe("Folder list population", func() {
		It("should populate Folder field with user's accessible library IDs", func() {
			testUser.Libraries = model.Libraries{
				{ID: 1, Name: "Music"},
				{ID: 2, Name: "Podcasts"},
				{ID: 5, Name: "Audiobooks"},
			}

			response := buildUserResponse(testUser)

			Expect(response.Folder).To(HaveLen(3))
			Expect(response.Folder).To(ContainElements(int32(1), int32(2), int32(5)))
		})
	})

	Describe("GetUser authorization", func() {
		It("should allow user to request their own information", func() {
			req := httptest.NewRequest("GET", "/rest/getUser?username=testuser", nil)
			ctx := request.WithUser(GinkgoT().Context(), testUser)
			req = req.WithContext(ctx)

			response, err := router.GetUser(req)

			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.User).ToNot(BeNil())
			Expect(response.User.Username).To(Equal("testuser"))
		})

		It("should deny user from requesting another user's information", func() {
			req := httptest.NewRequest("GET", "/rest/getUser?username=anotheruser", nil)
			ctx := request.WithUser(GinkgoT().Context(), testUser)
			req = req.WithContext(ctx)

			response, err := router.GetUser(req)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())

			var subErr subError
			ok := errors.As(err, &subErr)
			Expect(ok).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorAuthorizationFail))
		})

		It("should return error when username parameter is missing", func() {
			req := httptest.NewRequest("GET", "/rest/getUser", nil)
			ctx := request.WithUser(GinkgoT().Context(), testUser)
			req = req.WithContext(ctx)

			response, err := router.GetUser(req)

			Expect(err).To(MatchError("missing parameter: 'username'"))
			Expect(response).To(BeNil())
		})

		It("should return error when user context is missing", func() {
			req := httptest.NewRequest("GET", "/rest/getUser?username=testuser", nil)

			response, err := router.GetUser(req)

			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())

			var subErr subError
			ok := errors.As(err, &subErr)
			Expect(ok).To(BeTrue())
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})
	})
})
