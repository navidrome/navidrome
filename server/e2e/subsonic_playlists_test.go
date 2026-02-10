package e2e

import (
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlist Endpoints", Ordered, func() {
	var playlistID string
	var songIDs []string

	BeforeAll(func() {
		setupTestDB()

		// Look up song IDs from scanned data for playlist operations
		songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Sort: "title", Max: 6})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(songs)).To(BeNumerically(">=", 5))
		for _, s := range songs {
			songIDs = append(songIDs, s.ID)
		}
	})

	It("getPlaylists returns empty list initially", func() {
		resp := doReq("getPlaylists")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlists).ToNot(BeNil())
		Expect(resp.Playlists.Playlist).To(BeEmpty())
	})

	It("createPlaylist creates a new playlist with songs", func() {
		resp := doReq("createPlaylist", "name", "Test Playlist",
			"songId", songIDs[0], "songId", songIDs[1], "songId", songIDs[2])

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlist).ToNot(BeNil())
		Expect(resp.Playlist.Name).To(Equal("Test Playlist"))
		Expect(resp.Playlist.SongCount).To(Equal(int32(3)))
		Expect(resp.Playlist.Entry).To(HaveLen(3))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[0]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[1]))
		Expect(resp.Playlist.Entry[2].Id).To(Equal(songIDs[2]))
		playlistID = resp.Playlist.Id
	})

	It("getPlaylist returns playlist with tracks in order", func() {
		resp := doReq("getPlaylist", "id", playlistID)

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlist).ToNot(BeNil())
		Expect(resp.Playlist.Name).To(Equal("Test Playlist"))
		Expect(resp.Playlist.Entry).To(HaveLen(3))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[0]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[1]))
		Expect(resp.Playlist.Entry[2].Id).To(Equal(songIDs[2]))
	})

	It("createPlaylist without name or playlistId returns error", func() {
		resp := doReq("createPlaylist", "songId", songIDs[0])

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("createPlaylist with playlistId replaces tracks on existing playlist", func() {
		// Replace tracks: the playlist had [song0, song1, song2], replace with [song3, song4]
		resp := doReq("createPlaylist", "playlistId", playlistID,
			"songId", songIDs[3], "songId", songIDs[4])

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlist).ToNot(BeNil())
		Expect(resp.Playlist.Id).To(Equal(playlistID))
		Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		Expect(resp.Playlist.Entry).To(HaveLen(2))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[3]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[4]))
	})

	It("updatePlaylist can rename the playlist", func() {
		resp := doReq("updatePlaylist", "playlistId", playlistID, "name", "Renamed Playlist")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify the rename
		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.Name).To(Equal("Renamed Playlist"))
		// Tracks should be unchanged
		Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
	})

	It("updatePlaylist can set comment", func() {
		resp := doReq("updatePlaylist", "playlistId", playlistID, "comment", "My favorite songs")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.Comment).To(Equal("My favorite songs"))
	})

	It("updatePlaylist can set public visibility", func() {
		resp := doReq("updatePlaylist", "playlistId", playlistID, "public", "true")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.Public).To(BeTrue())
	})

	It("updatePlaylist can add songs", func() {
		// Playlist currently has [song3, song4], add song0
		resp := doReq("updatePlaylist", "playlistId", playlistID, "songIdToAdd", songIDs[0])
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.SongCount).To(Equal(int32(3)))
		Expect(resp.Playlist.Entry).To(HaveLen(3))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[3]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[4]))
		Expect(resp.Playlist.Entry[2].Id).To(Equal(songIDs[0]))
	})

	It("updatePlaylist can add multiple songs at once", func() {
		// Playlist currently has [song3, song4, song0], add song1 and song2
		resp := doReq("updatePlaylist", "playlistId", playlistID,
			"songIdToAdd", songIDs[1], "songIdToAdd", songIDs[2])
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.SongCount).To(Equal(int32(5)))
		Expect(resp.Playlist.Entry).To(HaveLen(5))
	})

	It("updatePlaylist can remove songs by index and verifies correct songs remain", func() {
		// Playlist has [song3, song4, song0, song1, song2]
		// Remove index 0 (song3) and index 2 (song0)
		resp := doReq("updatePlaylist", "playlistId", playlistID,
			"songIndexToRemove", "0", "songIndexToRemove", "2")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.SongCount).To(Equal(int32(3)))
		Expect(resp.Playlist.Entry).To(HaveLen(3))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[4]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[1]))
		Expect(resp.Playlist.Entry[2].Id).To(Equal(songIDs[2]))
	})

	It("updatePlaylist can remove and add songs in a single call", func() {
		// Playlist has [song4, song1, song2]
		// Remove index 1 (song1) and add song3
		resp := doReq("updatePlaylist", "playlistId", playlistID,
			"songIndexToRemove", "1", "songIdToAdd", songIDs[3])
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.SongCount).To(Equal(int32(3)))
		Expect(resp.Playlist.Entry).To(HaveLen(3))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[4]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[2]))
		Expect(resp.Playlist.Entry[2].Id).To(Equal(songIDs[3]))
	})

	It("updatePlaylist can combine metadata change with track removal", func() {
		// Playlist has [song4, song2, song3]
		// Rename + remove index 0 (song4)
		resp := doReq("updatePlaylist", "playlistId", playlistID,
			"name", "Final Playlist", "songIndexToRemove", "0")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.Name).To(Equal("Final Playlist"))
		Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[2]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[3]))
	})

	It("updatePlaylist can remove all songs from playlist", func() {
		// Playlist has [song2, song3] — remove both
		resp := doReq("updatePlaylist", "playlistId", playlistID,
			"songIndexToRemove", "0", "songIndexToRemove", "1")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.SongCount).To(Equal(int32(0)))
		Expect(resp.Playlist.Entry).To(BeEmpty())
	})

	It("updatePlaylist can add songs to an empty playlist", func() {
		resp := doReq("updatePlaylist", "playlistId", playlistID,
			"songIdToAdd", songIDs[0])
		Expect(resp.Status).To(Equal(responses.StatusOK))

		resp = doReq("getPlaylist", "id", playlistID)
		Expect(resp.Playlist.SongCount).To(Equal(int32(1)))
		Expect(resp.Playlist.Entry).To(HaveLen(1))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[0]))
	})

	It("updatePlaylist without playlistId returns error", func() {
		resp := doReq("updatePlaylist", "name", "No ID")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("getPlaylists shows the playlist", func() {
		resp := doReq("getPlaylists")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlists.Playlist).To(HaveLen(1))
		Expect(resp.Playlists.Playlist[0].Id).To(Equal(playlistID))
	})

	It("deletePlaylist removes the playlist", func() {
		resp := doReq("deletePlaylist", "id", playlistID)

		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getPlaylist on deleted playlist returns error", func() {
		resp := doReq("getPlaylist", "id", playlistID)

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("getPlaylists returns empty after deletion", func() {
		resp := doReq("getPlaylists")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlists.Playlist).To(BeEmpty())
	})

	Describe("Playlist Permissions", Ordered, func() {
		var songIDs []string
		var adminPrivateID string
		var adminPublicID string
		var regularPlaylistID string

		BeforeAll(func() {
			setupTestDB()

			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Sort: "title", Max: 6})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(songs)).To(BeNumerically(">=", 3))
			for _, s := range songs {
				songIDs = append(songIDs, s.ID)
			}
		})

		It("admin creates a private playlist", func() {
			resp := doReqWithUser(adminUser, "createPlaylist", "name", "Admin Private",
				"songId", songIDs[0], "songId", songIDs[1])

			Expect(resp.Status).To(Equal(responses.StatusOK))
			adminPrivateID = resp.Playlist.Id
		})

		It("admin creates a public playlist", func() {
			resp := doReqWithUser(adminUser, "createPlaylist", "name", "Admin Public",
				"songId", songIDs[0], "songId", songIDs[1])

			Expect(resp.Status).To(Equal(responses.StatusOK))
			adminPublicID = resp.Playlist.Id

			// Make it public
			resp = doReqWithUser(adminUser, "updatePlaylist",
				"playlistId", adminPublicID, "public", "true")
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("regular user creates a playlist", func() {
			resp := doReqWithUser(regularUser, "createPlaylist", "name", "Regular Playlist",
				"songId", songIDs[0])

			Expect(resp.Status).To(Equal(responses.StatusOK))
			regularPlaylistID = resp.Playlist.Id
		})

		// --- Private playlist: regular user gets "not found" (repo hides it entirely) ---

		It("regular user cannot see admin's private playlist", func() {
			resp := doReqWithUser(regularUser, "getPlaylist", "id", adminPrivateID)

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("regular user cannot update admin's private playlist (not found)", func() {
			resp := doReqWithUser(regularUser, "updatePlaylist",
				"playlistId", adminPrivateID, "name", "Hacked")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("regular user cannot delete admin's private playlist (not found)", func() {
			resp := doReqWithUser(regularUser, "deletePlaylist", "id", adminPrivateID)

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		// --- Public playlist: regular user can see but cannot modify (authorization fail, code 50) ---

		It("regular user can see admin's public playlist", func() {
			resp := doReqWithUser(regularUser, "getPlaylist", "id", adminPublicID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Playlist.Name).To(Equal("Admin Public"))
		})

		It("regular user cannot update admin's public playlist", func() {
			resp := doReqWithUser(regularUser, "updatePlaylist",
				"playlistId", adminPublicID, "name", "Hacked")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("regular user cannot add songs to admin's public playlist", func() {
			resp := doReqWithUser(regularUser, "updatePlaylist",
				"playlistId", adminPublicID, "songIdToAdd", songIDs[2])

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("regular user cannot remove songs from admin's public playlist", func() {
			resp := doReqWithUser(regularUser, "updatePlaylist",
				"playlistId", adminPublicID, "songIndexToRemove", "0")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("regular user cannot delete admin's public playlist", func() {
			resp := doReqWithUser(regularUser, "deletePlaylist", "id", adminPublicID)

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("regular user cannot replace tracks on admin's public playlist via createPlaylist", func() {
			resp := doReqWithUser(regularUser, "createPlaylist",
				"playlistId", adminPublicID, "songId", songIDs[2])

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		// --- Regular user can manage their own playlists ---

		It("regular user can update their own playlist", func() {
			resp := doReqWithUser(regularUser, "updatePlaylist",
				"playlistId", regularPlaylistID, "name", "My Updated Playlist")

			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReqWithUser(regularUser, "getPlaylist", "id", regularPlaylistID)
			Expect(resp.Playlist.Name).To(Equal("My Updated Playlist"))
		})

		It("regular user can add songs to their own playlist", func() {
			resp := doReqWithUser(regularUser, "updatePlaylist",
				"playlistId", regularPlaylistID, "songIdToAdd", songIDs[1])

			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReqWithUser(regularUser, "getPlaylist", "id", regularPlaylistID)
			Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		})

		It("regular user can delete their own playlist", func() {
			resp := doReqWithUser(regularUser, "deletePlaylist", "id", regularPlaylistID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		// --- Admin can manage any user's playlists ---

		It("admin can update any user's playlist", func() {
			resp := doReqWithUser(regularUser, "createPlaylist", "name", "To Be Admin-Edited",
				"songId", songIDs[0])
			Expect(resp.Status).To(Equal(responses.StatusOK))
			plsID := resp.Playlist.Id

			resp = doReqWithUser(adminUser, "updatePlaylist",
				"playlistId", plsID, "name", "Admin Edited")
			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReqWithUser(adminUser, "getPlaylist", "id", plsID)
			Expect(resp.Playlist.Name).To(Equal("Admin Edited"))
		})

		It("admin can delete any user's playlist", func() {
			resp := doReqWithUser(regularUser, "createPlaylist", "name", "To Be Admin-Deleted",
				"songId", songIDs[0])
			Expect(resp.Status).To(Equal(responses.StatusOK))
			plsID := resp.Playlist.Id

			resp = doReqWithUser(adminUser, "deletePlaylist", "id", plsID)
			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReqWithUser(adminUser, "getPlaylist", "id", plsID)
			Expect(resp.Status).To(Equal(responses.StatusFailed))
		})

		// --- Verify admin's playlists are unchanged ---

		It("admin's private playlist is unchanged after failed regular user operations", func() {
			resp := doReqWithUser(adminUser, "getPlaylist", "id", adminPrivateID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Playlist.Name).To(Equal("Admin Private"))
			Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		})

		It("admin's public playlist is unchanged after failed regular user operations", func() {
			resp := doReqWithUser(adminUser, "getPlaylist", "id", adminPublicID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Playlist.Name).To(Equal("Admin Public"))
			Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		})
	})

	Describe("Smart Playlist Protection", Ordered, func() {
		var smartPlaylistID string
		var songID string

		BeforeAll(func() {
			setupTestDB()

			// Look up a song ID for mutation tests
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Sort: "title", Max: 1})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID = songs[0].ID

			// Insert a smart playlist directly into the DB
			smartPls := &model.Playlist{
				Name:    "Smart Playlist",
				OwnerID: adminUser.ID,
				Public:  false,
				Rules:   &criteria.Criteria{Expression: criteria.Contains{"title": ""}},
			}
			Expect(ds.Playlist(ctx).Put(smartPls)).To(Succeed())
			smartPlaylistID = smartPls.ID
		})

		It("getPlaylist returns smart playlist with readonly flag and validUntil", func() {
			resp := doReq("getPlaylist", "id", smartPlaylistID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Playlist.Name).To(Equal("Smart Playlist"))
			Expect(resp.Playlist.OpenSubsonicPlaylist).ToNot(BeNil())
			Expect(resp.Playlist.OpenSubsonicPlaylist.Readonly).To(BeTrue())
			expectedValidUntil := time.Now().Add(conf.Server.SmartPlaylistRefreshDelay)
			Expect(*resp.Playlist.OpenSubsonicPlaylist.ValidUntil).To(BeTemporally("~", expectedValidUntil, time.Second))
		})

		It("createPlaylist rejects replacing tracks on smart playlist", func() {
			resp := doReq("createPlaylist", "playlistId", smartPlaylistID, "songId", songID)

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("updatePlaylist rejects adding songs to smart playlist", func() {
			resp := doReq("updatePlaylist", "playlistId", smartPlaylistID,
				"songIdToAdd", songID)

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("updatePlaylist rejects removing songs from smart playlist", func() {
			resp := doReq("updatePlaylist", "playlistId", smartPlaylistID,
				"songIndexToRemove", "0")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(int32(50)))
		})

		It("updatePlaylist allows renaming smart playlist", func() {
			resp := doReq("updatePlaylist", "playlistId", smartPlaylistID,
				"name", "Renamed Smart")
			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReq("getPlaylist", "id", smartPlaylistID)
			Expect(resp.Playlist.Name).To(Equal("Renamed Smart"))
		})

		It("updatePlaylist allows setting comment on smart playlist", func() {
			resp := doReq("updatePlaylist", "playlistId", smartPlaylistID,
				"comment", "Auto-generated playlist")
			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReq("getPlaylist", "id", smartPlaylistID)
			Expect(resp.Playlist.Comment).To(Equal("Auto-generated playlist"))
		})

		It("deletePlaylist can delete smart playlist", func() {
			resp := doReq("deletePlaylist", "id", smartPlaylistID)
			Expect(resp.Status).To(Equal(responses.StatusOK))

			resp = doReq("getPlaylist", "id", smartPlaylistID)
			Expect(resp.Status).To(Equal(responses.StatusFailed))
		})
	})
})
