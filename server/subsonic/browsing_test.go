package subsonic

import (
	"context"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browsing", func() {
	var api *Router
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		api = &Router{ds: ds}
		ctx = context.Background()
	})

	Describe("GetMusicFolders", func() {
		It("should return all libraries the user has access", func() {
			// Create mock user
			user := model.User{
				ID: "user-id",
			}
			ctx = request.WithUser(ctx, user)

			// Setup mock expectations - admin users get all libraries via GetUserLibraries
			mockUserRepo := ds.User(ctx).(*tests.MockedUserRepo)
			err := mockUserRepo.SetUserLibraries("user-id", []int{1, 2, 3})
			Expect(err).ToNot(HaveOccurred())

			// Create request
			r := httptest.NewRequest("GET", "/rest/getMusicFolders", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetMusicFolders(r)

			// Verify results
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.MusicFolders).ToNot(BeNil())
			Expect(response.MusicFolders.Folders).To(HaveLen(3))
			Expect(response.MusicFolders.Folders[0].Name).To(Equal("Test Library 1"))
			Expect(response.MusicFolders.Folders[1].Name).To(Equal("Test Library 2"))
			Expect(response.MusicFolders.Folders[2].Name).To(Equal("Test Library 3"))
		})
	})
})
