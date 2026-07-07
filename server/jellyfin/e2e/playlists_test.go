package e2e

import (
	"bytes"
	"image"
	jpeglib "image/jpeg"
	"net/http"
	"os"
	"time"

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

		// Jellify's @jellyfin/sdk serializes id arrays as repeated params (ids=X&ids=Y), not a
		// comma-joined value; all ids must be added, not just the first.
		It("adds multiple songs sent as repeated ids params", func() {
			plID := createPlaylist("Multi", nil)
			url := "/Playlists/" + enc(plID) + "/Items?ids=" + enc(songID("So What")) +
				"&ids=" + enc(songID("Come Together")) + "&ids=" + enc(songID("Help!"))
			Expect(post(url, "").Code).To(Equal(http.StatusNoContent))
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(3))
		})

		It("removes an entry by its PlaylistItemId", func() {
			plID := createPlaylist("Remove", []string{enc(songID("Come Together")), enc(songID("Something"))})
			entryID := playlistItems(plID).Items[0].PlaylistItemId
			Expect(del("/Playlists/" + enc(plID) + "/Items?entryIds=" + entryID).Code).To(Equal(http.StatusNoContent))
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(1))
		})

		It("removes multiple entries sent as repeated entryIds params", func() {
			plID := createPlaylist("MultiRemove", []string{enc(songID("Come Together")), enc(songID("Something")), enc(songID("So What"))})
			items := playlistItems(plID).Items
			url := "/Playlists/" + enc(plID) + "/Items?entryIds=" + items[0].PlaylistItemId + "&entryIds=" + items[1].PlaylistItemId
			Expect(del(url).Code).To(Equal(http.StatusNoContent))
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

		It("sorts playlists by name when SortBy=SortName", func() {
			createPlaylist("Charlie", nil)
			createPlaylist("Alpha", nil)
			createPlaylist("Bravo", nil)
			q := queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true&SortBy=SortName"))
			Expect(names(q.Items)).To(Equal([]string{"Alpha", "Bravo", "Charlie"}))
		})
	})

	// Jellify resolves the "playlists library" via a ManualPlaylistsFolder query, then lists
	// playlists with ParentId set to that folder's id (no IncludeItemTypes). Without a folder item
	// whose CollectionType is "playlists", its query resolves undefined and React Query retries in a
	// backoff loop that stalls the home screen.
	Describe("playlists library folder (ManualPlaylistsFolder)", func() {
		It("returns a synthetic playlists folder with CollectionType=playlists", func() {
			q := queryResult(get("/Items?includeItemTypes=ManualPlaylistsFolder&excludeItemTypes=CollectionFolder"))
			Expect(q.Items).To(HaveLen(1))
			Expect(q.Items[0].CollectionType).To(Equal("playlists"))
			Expect(q.Items[0].Id).To(Equal(enc("playlists")))
		})

		It("lists the user's playlists when browsing the folder by ParentId (no IncludeItemTypes)", func() {
			createPlaylist("My Mix", nil)
			q := queryResult(get("/Items?parentId=" + enc("playlists")))
			Expect(names(q.Items)).To(ContainElement("My Mix"))
			Expect(q.Items[0].Type).To(Equal("Playlist"))
			// Jellify keeps only playlists whose Path contains "data".
			Expect(q.Items[0].Path).To(ContainSubstring("data"))
		})

		It("resolves the synthetic playlists folder by its own advertised id", func() {
			var item dto.BaseItemDto
			parseInto(get("/Items/"+enc("playlists")), &item)
			Expect(item.Type).To(Equal("ManualPlaylistsFolder"))
			Expect(item.CollectionType).To(Equal("playlists"))
			Expect(item.Id).To(Equal(enc("playlists")))
		})
	})

	// Real Jellyfin returns a playlist's children for /Items?ParentId=<playlistId> with no
	// IncludeItemTypes; generic clients (not Finamp/Jellify) browse playlists this way.
	Describe("browsing a playlist via the generic /Items path", func() {
		It("lists the playlist's tracks for a typeless ParentId query", func() {
			plID := createPlaylist("Browse Me", []string{enc(songID("Come Together")), enc(songID("So What"))})
			q := queryResult(get("/Items?parentId=" + enc(plID)))
			Expect(q.TotalRecordCount).To(Equal(2))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "So What"))
			Expect(q.Items[0].Type).To(Equal("Audio"))
		})

		It("pages the playlist's tracks", func() {
			plID := createPlaylist("Browse Paged", []string{enc(songID("Come Together")), enc(songID("So What"))})
			q := queryResult(get("/Items?parentId=" + enc(plID) + "&startIndex=1&limit=1"))
			Expect(q.Items).To(HaveLen(1))
			Expect(q.TotalRecordCount).To(Equal(2))
		})

		// Jellify opens a playlist with ParentId=<playlist>&IncludeItemTypes=Audio&Recursive=false.
		// The playlist id must resolve to its tracks, not be treated as an album id (which returns none).
		It("lists the playlist's tracks even when IncludeItemTypes=Audio is set", func() {
			plID := createPlaylist("Typed Browse", []string{enc(songID("Come Together")), enc(songID("So What"))})
			q := queryResult(get("/Items?parentId=" + enc(plID) + "&includeItemTypes=Audio&recursive=false"))
			Expect(q.TotalRecordCount).To(Equal(2))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "So What"))
		})
	})

	Describe("cover art", func() {
		// A real (decodable) image: the upload endpoint validates by decoding, like the native one.
		var jpeg []byte
		BeforeEach(func() {
			var buf bytes.Buffer
			Expect(jpeglib.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil)).To(Succeed())
			jpeg = buf.Bytes()
		})

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

		// Guards the whole chain: SetImage must go through a full Put (which bumps UpdatedAt), and the
		// tag must be versioned by it, or clients keep their blurhash-keyed cover cache forever.
		It("rotates the playlist's image tag and blurhash after a cover upload", func() {
			plID := createPlaylist("Cover Tag", nil)
			imageTag := func() string {
				q := queryResult(get("/Items?ids=" + enc(plID)))
				Expect(q.Items).To(HaveLen(1))
				return q.Items[0].ImageTags["Primary"]
			}
			before := imageTag()
			Expect(before).ToNot(BeEmpty())

			time.Sleep(2 * time.Millisecond) // UpdatedAt has millisecond resolution in the tag
			Expect(upload(adminUser, "/Items/"+enc(plID)+"/Images/Primary", "image/jpeg", jpeg).Code).
				To(Equal(http.StatusNoContent))

			after := imageTag()
			Expect(after).ToNot(Equal(before))
			q := queryResult(get("/Items?ids=" + enc(plID)))
			Expect(q.Items[0].ImageBlurHashes["Primary"]).To(HaveKey(after))
		})
	})

	Describe("update", func() {
		It("makes a playlist public", func() {
			plID := createPlaylist("Make Public", nil)
			Expect(post("/Playlists/"+enc(plID), `{"Name":"Make Public","IsPublic":true}`).Code).To(Equal(http.StatusNoContent))

			var info dto.PlaylistInfo
			parseInto(get("/Playlists/"+enc(plID)), &info)
			Expect(info.OpenAccess).To(BeTrue())
			// Now visible to other users.
			Expect(queryResult(getAs(regularUser, "/Items?IncludeItemTypes=Playlist&Recursive=true")).TotalRecordCount).To(Equal(1))
		})

		It("renames a playlist", func() {
			plID := createPlaylist("Old Name", nil)
			Expect(post("/Playlists/"+enc(plID), `{"Name":"New Name"}`).Code).To(Equal(http.StatusNoContent))
			pls, _ := ds.Playlist(ctx).Get(plID)
			Expect(pls.Name).To(Equal("New Name"))
		})

		It("replaces the track list when Ids are provided", func() {
			plID := createPlaylist("Reorder", []string{enc(songID("Come Together")), enc(songID("Something"))})
			// Replace with a single different track.
			Expect(post("/Playlists/"+enc(plID), `{"Ids":["`+enc(songID("So What"))+`"]}`).Code).To(Equal(http.StatusNoContent))
			q := playlistItems(plID)
			Expect(q.TotalRecordCount).To(Equal(1))
			Expect(q.Items[0].Name).To(Equal("So What"))
		})

		It("clears the track list when an explicit empty Ids array is sent", func() {
			plID := createPlaylist("Clear Me", []string{enc(songID("Come Together")), enc(songID("Something"))})
			Expect(post("/Playlists/"+enc(plID), `{"Ids":[]}`).Code).To(Equal(http.StatusNoContent))
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(0))
		})

		It("leaves the track list intact when Ids is omitted (metadata-only update)", func() {
			plID := createPlaylist("Keep Tracks", []string{enc(songID("Come Together")), enc(songID("Something"))})
			Expect(post("/Playlists/"+enc(plID), `{"Name":"Renamed"}`).Code).To(Equal(http.StatusNoContent))
			Expect(playlistItems(plID).TotalRecordCount).To(Equal(2))
		})

		It("applies Name and IsPublic sent together with a track replacement", func() {
			plID := createPlaylist("Combo", []string{enc(songID("Come Together"))})
			body := `{"Name":"Combo Renamed","IsPublic":true,"Ids":["` + enc(songID("So What")) + `"]}`
			Expect(post("/Playlists/"+enc(plID), body).Code).To(Equal(http.StatusNoContent))
			q := playlistItems(plID)
			Expect(q.TotalRecordCount).To(Equal(1))
			Expect(q.Items[0].Name).To(Equal("So What"))
			pls, _ := ds.Playlist(ctx).Get(plID)
			Expect(pls.Name).To(Equal("Combo Renamed"))
			Expect(pls.Public).To(BeTrue())
		})

		It("forbids a non-owner from updating a public playlist", func() {
			plID := createPlaylist("Owned", nil)
			post("/Playlists/"+enc(plID), `{"IsPublic":true}`) // make it visible to the regular user
			Expect(postAs(regularUser, "/Playlists/"+enc(plID), `{"Name":"Hijacked"}`).Code).To(Equal(http.StatusForbidden))
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
