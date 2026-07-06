package e2e

import (
	"net/http"
	"os"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	BeforeEach(func() { setupTestDB() })

	playlistItems := func(plID string) dto.QueryResult {
		return queryResult(get("/Playlists/" + enc(plID) + "/Items"))
	}

	Describe("create", func() {
		It("creates an empty playlist", func() {
			plID := createPlaylist("Empty", nil)
			var info dto.PlaylistInfo
			parseInto(get("/Playlists/"+enc(plID)), &info)
			Expect(info.OpenAccess).To(BeFalse())
			Expect(info.Shares).To(BeEmpty())
			Expect(info.ItemIds).To(BeEmpty())
		})

		It("creates a playlist from song ids", func() {
			plID := createPlaylist("Songs", []string{enc(songID("Come Together")), enc(songID("So What"))})
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(2))
		})

		It("expands an album id into its tracks", func() {
			plID := createPlaylist("From Album", []string{enc(albumID("Abbey Road"))})
			q := playlistItems(plID)
			Expect(q.TotalRecordCount).To(Equal(2))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something"))
		})

		It("expands an artist id into its tracks", func() {
			plID := createPlaylist("From Artist", []string{enc(artistID("The Beatles"))})
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(3)) // Abbey Road (2) + Help! (1)
		})
	})

	Describe("items", func() {
		It("tags each entry with a PlaylistItemId", func() {
			plID := createPlaylist("Tagged", []string{enc(songID("Help!"))})
			q := playlistItems(plID)
			Expect(q.Items).To(HaveLen(1))
			Expect(q.Items[0].Type).To(Equal("Audio"))
			Expect(q.Items[0].PlaylistItemId).ToNot(BeEmpty())
		})
	})

	Describe("add and remove", func() {
		It("adds a song by id", func() {
			plID := createPlaylist("Add", nil)
			Expect(post("/Playlists/"+enc(plID)+"/Items?ids="+enc(songID("So What")), "").Code).To(Equal(http.StatusNoContent))
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(1))
		})

		It("adds an album (expanding to its tracks)", func() {
			plID := createPlaylist("AddAlbum", []string{enc(songID("So What"))})
			post("/Playlists/"+enc(plID)+"/Items?ids="+enc(albumID("Abbey Road")), "")
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(3)) // 1 + Abbey Road (2)
		})

		It("removes an entry by its PlaylistItemId", func() {
			plID := createPlaylist("Remove", []string{enc(songID("Come Together")), enc(songID("Something"))})
			entryID := playlistItems(plID).Items[0].PlaylistItemId
			Expect(del("/Playlists/" + enc(plID) + "/Items?entryIds=" + entryID).Code).To(Equal(http.StatusNoContent))
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(1))
		})
	})

	Describe("users", func() {
		It("reports the current user as an editor", func() {
			plID := createPlaylist("Perms", nil)
			var perms []dto.PlaylistUserPermissions
			parseInto(get("/Playlists/"+enc(plID)+"/Users"), &perms)
			Expect(perms).To(HaveLen(1))
			Expect(perms[0].UserId).To(Equal("admin-1"))
			Expect(perms[0].CanEdit).To(BeTrue())
		})
	})

	Describe("listing", func() {
		It("lists a created playlist advertising a Primary image tag", func() {
			createPlaylist("Listed", nil)
			q := queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(1))
			Expect(q.Items[0].Name).To(Equal("Listed"))
			Expect(q.Items[0].ImageTags).To(HaveKey("Primary"))
		})
	})

	Describe("cover art", func() {
		jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}

		It("uploads and removes a playlist cover", func() {
			plID := createPlaylist("Cover", nil)

			Expect(upload(adminUser, "/Items/"+enc(plID)+"/Images/Primary", "image/jpeg", jpeg).Code).
				To(Equal(http.StatusNoContent))

			pls, err := ds.Playlist(ctx).Get(plID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.UploadedImage).ToNot(BeEmpty())
			_, statErr := os.Stat(pls.UploadedImagePath())
			Expect(statErr).ToNot(HaveOccurred(), "cover file should exist on disk")

			Expect(del("/Items/" + enc(plID) + "/Images/Primary").Code).To(Equal(http.StatusNoContent))
			pls, _ = ds.Playlist(ctx).Get(plID)
			Expect(pls.UploadedImage).To(BeEmpty())
		})

		It("rejects cover upload for a non-playlist item", func() {
			Expect(upload(adminUser, "/Items/"+enc(albumID("IV"))+"/Images/Primary", "image/jpeg", jpeg).Code).
				To(Equal(http.StatusNotImplemented))
		})
	})

	Describe("delete", func() {
		It("deletes a playlist", func() {
			plID := createPlaylist("ToDelete", nil)
			Expect(del("/Items/" + enc(plID)).Code).To(Equal(http.StatusNoContent))
			Expect(queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(0))
		})

		It("returns 404 when deleting a non-playlist item", func() {
			Expect(del("/Items/" + enc(albumID("IV"))).Code).To(Equal(http.StatusNotFound))
		})
	})
})
