package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
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
			noAccess := model.User{ID: testID("noaccess-1"), UserName: "noaccess", Name: "No Access", NewPassword: "password"}
			Expect(ds.User().Put(ctx, &noAccess)).To(Succeed())
			loaded, err := ds.User().FindByUsername(ctx, "noaccess")
			Expect(err).ToNot(HaveOccurred())

			q := queryResult(getAs(*loaded, "/Items?IncludeItemTypes=MusicAlbum&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(0))
		})
	})

	Describe("personal ratings", func() {
		userData := func(u model.User, id string) *dto.UserItemDataDto {
			var data dto.UserItemDataDto
			parseInto(getAs(u, "/UserItems/"+enc(id)+"/UserData"), &data)
			return &data
		}

		It("keeps each user's rating separate", func() {
			id := songID("So What")
			Expect(post("/UserItems/"+enc(id)+"/UserData", `{"Rating":10}`).Code).To(Equal(http.StatusOK))
			Expect(postAs(regularUser, "/UserItems/"+enc(id)+"/Rating?likes=false", "").Code).To(Equal(http.StatusOK))

			Expect(*userData(adminUser, id).Rating).To(Equal(float64(10)))
			Expect(*userData(regularUser, id).Rating).To(Equal(float64(2)))
		})

		It("refuses writes that name another user, even for an admin, and changes nothing", func() {
			id := songID("So What")
			Expect(postAs(regularUser, "/UserItems/"+enc(id)+"/UserData", `{"Rating":4}`).Code).To(Equal(http.StatusOK))

			Expect(postAs(regularUser, "/Users/"+enc(adminUser.ID)+"/Items/"+enc(id)+"/UserData", `{"Rating":10}`).Code).To(Equal(http.StatusForbidden))
			Expect(post("/Users/"+enc(regularUser.ID)+"/Items/"+enc(id)+"/Rating?likes=true", "").Code).To(Equal(http.StatusForbidden))
			Expect(del("/Users/" + enc(regularUser.ID) + "/Items/" + enc(id) + "/Rating").Code).To(Equal(http.StatusForbidden))
			Expect(post("/UserItems/"+enc(id)+"/UserData?userId="+enc(regularUser.ID), `{"Rating":10}`).Code).To(Equal(http.StatusForbidden))

			Expect(userData(adminUser, id).Rating).To(BeNil())
			Expect(*userData(regularUser, id).Rating).To(Equal(float64(4)))
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
			Expect(postAs(regularUser, "/UserItems/"+enc(adminPl)+"/UserData", `{"Rating":10}`).Code).To(Equal(http.StatusNotFound))
		})

		It("lets each user manage their own playlist", func() {
			regularPl := createPlaylistAs(regularUser, "Regular's Mix")
			Expect(queryResult(getAs(regularUser, "/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(1))
			Expect(delAs(regularUser, "/Items/"+enc(regularPl)).Code).To(Equal(http.StatusNoContent))
		})
	})
})
