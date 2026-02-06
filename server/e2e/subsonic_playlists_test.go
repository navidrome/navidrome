package e2e

import (
	"github.com/navidrome/navidrome/model"
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
		songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Sort: "title", Max: 3})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(songs)).To(BeNumerically(">=", 3))
		for _, s := range songs {
			songIDs = append(songIDs, s.ID)
		}
	})

	It("getPlaylists returns empty list initially", func() {
		r := newReq("getPlaylists")
		resp, err := router.GetPlaylists(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlists).ToNot(BeNil())
		Expect(resp.Playlists.Playlist).To(BeEmpty())
	})

	It("createPlaylist creates a new playlist with songs", func() {
		r := newReq("createPlaylist", "name", "Test Playlist", "songId", songIDs[0], "songId", songIDs[1])
		resp, err := router.CreatePlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlist).ToNot(BeNil())
		Expect(resp.Playlist.Name).To(Equal("Test Playlist"))
		Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		playlistID = resp.Playlist.Id
	})

	It("getPlaylist returns playlist with tracks", func() {
		r := newReq("getPlaylist", "id", playlistID)
		resp, err := router.GetPlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Playlist).ToNot(BeNil())
		Expect(resp.Playlist.Name).To(Equal("Test Playlist"))
		Expect(resp.Playlist.Entry).To(HaveLen(2))
		Expect(resp.Playlist.Entry[0].Id).To(Equal(songIDs[0]))
		Expect(resp.Playlist.Entry[1].Id).To(Equal(songIDs[1]))
	})

	It("createPlaylist without name or playlistId returns error", func() {
		r := newReq("createPlaylist", "songId", songIDs[0])
		_, err := router.CreatePlaylist(r)

		Expect(err).To(HaveOccurred())
	})

	It("updatePlaylist can rename the playlist", func() {
		r := newReq("updatePlaylist", "playlistId", playlistID, "name", "Renamed Playlist")
		resp, err := router.UpdatePlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify the rename
		r = newReq("getPlaylist", "id", playlistID)
		resp, err = router.GetPlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Playlist.Name).To(Equal("Renamed Playlist"))
	})

	It("updatePlaylist can add songs", func() {
		r := newReq("updatePlaylist", "playlistId", playlistID, "songIdToAdd", songIDs[2])
		resp, err := router.UpdatePlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify the song was added
		r = newReq("getPlaylist", "id", playlistID)
		resp, err = router.GetPlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Playlist.SongCount).To(Equal(int32(3)))
		Expect(resp.Playlist.Entry).To(HaveLen(3))
	})

	It("updatePlaylist can remove songs by index", func() {
		// Remove the first song (index 0)
		r := newReq("updatePlaylist", "playlistId", playlistID, "songIndexToRemove", "0")
		resp, err := router.UpdatePlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify the song was removed
		r = newReq("getPlaylist", "id", playlistID)
		resp, err = router.GetPlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
		Expect(resp.Playlist.Entry).To(HaveLen(2))
	})

	It("deletePlaylist removes the playlist", func() {
		r := newReq("deletePlaylist", "id", playlistID)
		resp, err := router.DeletePlaylist(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getPlaylist on deleted playlist returns error", func() {
		r := newReq("getPlaylist", "id", playlistID)
		_, err := router.GetPlaylist(r)

		Expect(err).To(HaveOccurred())
	})
})
