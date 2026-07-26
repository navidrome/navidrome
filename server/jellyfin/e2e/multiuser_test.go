package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multi-user access control", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("library scoping", func() {
		It("lets a library member browse its content", func() {
			q := queryResult(getAs(regularUser, "/Items?IncludeItemTypes=MusicAlbum&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(5))
		})

		It("hides all content from a user with no library access", func() {
			noAccess := model.User{ID: "noaccess-1", UserName: "noaccess", Name: "No Access", NewPassword: "password"}
			Expect(ds.User(ctx).Put(&noAccess)).To(Succeed())
			loaded, err := ds.User(ctx).FindByUsername("noaccess")
			Expect(err).ToNot(HaveOccurred())

			q := queryResult(getAs(*loaded, "/Items?IncludeItemTypes=MusicAlbum&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(0))
		})
	})

	Describe("private playlists", func() {
		It("does not expose another user's private playlist", func() {
			adminPl := createPlaylist("Admin Private", nil)

			// Owner sees it.
			Expect(queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(1))
			// A different user does not.
			Expect(queryResult(getAs(regularUser, "/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(0))
			// And can't read its items.
			Expect(getAs(regularUser, "/Playlists/"+enc(adminPl)+"/Items").Code).To(Equal(http.StatusNotFound))
		})

		It("does not let a non-owner delete another user's private playlist", func() {
			adminPl := createPlaylist("Admin Private", nil)
			// The playlist is invisible to the regular user, so delete resolves to 404 (not 403) —
			// the API never reveals that someone else's private playlist exists.
			Expect(delAs(regularUser, "/Items/"+enc(adminPl)).Code).To(Equal(http.StatusNotFound))
			// Still present for the owner.
			Expect(queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(1))
		})

		It("does not let a non-owner annotate another user's private playlist", func() {
			adminPl := createPlaylist("Admin Private", nil)
			Expect(postAs(regularUser, "/Users/user-1/FavoriteItems/"+enc(adminPl), "").Code).To(Equal(http.StatusNotFound))
			Expect(postAs(regularUser, "/Users/user-1/Items/"+enc(adminPl)+"/Rating?Rating=10", "").Code).To(Equal(http.StatusNotFound))
		})

		It("lets each user manage their own playlist", func() {
			regularPl := createPlaylistAs(regularUser, "Regular's Mix")
			Expect(queryResult(getAs(regularUser, "/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(1))
			Expect(delAs(regularUser, "/Items/"+enc(regularPl)).Code).To(Equal(http.StatusNoContent))
		})
	})
})
