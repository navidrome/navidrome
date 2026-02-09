package e2e

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Media Annotation Endpoints", Ordered, func() {
	BeforeAll(func() {
		setupTestDB()
	})

	Describe("Star/Unstar", Ordered, func() {
		var songID, albumID, artistID string

		BeforeAll(func() {
			// Look up a song from the scanned data
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID = songs[0].ID

			// Look up an album
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "name"})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			albumID = albums[0].ID

			// Look up an artist
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "name"})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			artistID = artists[0].ID
		})

		It("stars a song by id", func() {
			resp := doReq("star", "id", songID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("starred song appears in getStarred response", func() {
			resp := doReq("getStarred")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Starred).ToNot(BeNil())
			Expect(resp.Starred.Song).To(HaveLen(1))
			Expect(resp.Starred.Song[0].Id).To(Equal(songID))
		})

		It("unstars a previously starred song", func() {
			resp := doReq("unstar", "id", songID)

			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify song no longer appears in starred
			resp = doReq("getStarred")

			Expect(resp.Starred.Song).To(BeEmpty())
		})

		It("stars an album by albumId", func() {
			resp := doReq("star", "albumId", albumID)

			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify album appears in starred
			resp = doReq("getStarred")

			Expect(resp.Starred.Album).To(HaveLen(1))
			Expect(resp.Starred.Album[0].Id).To(Equal(albumID))
		})

		It("stars an artist by artistId", func() {
			resp := doReq("star", "artistId", artistID)

			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify artist appears in starred
			resp = doReq("getStarred")

			Expect(resp.Starred.Artist).To(HaveLen(1))
			Expect(resp.Starred.Artist[0].Id).To(Equal(artistID))
		})

		It("returns error when no id provided", func() {
			resp := doReq("star")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("SetRating", Ordered, func() {
		var songID, albumID string

		BeforeAll(func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID = songs[0].ID

			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "name"})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			albumID = albums[0].ID
		})

		It("sets rating on a song", func() {
			resp := doReq("setRating", "id", songID, "rating", "4")

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("rated song has correct userRating in getSong", func() {
			resp := doReq("getSong", "id", songID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Song).ToNot(BeNil())
			Expect(resp.Song.UserRating).To(Equal(int32(4)))
		})

		It("sets rating on an album", func() {
			resp := doReq("setRating", "id", albumID, "rating", "3")

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("returns error for missing parameters", func() {
			// Missing both id and rating
			resp := doReq("setRating")
			Expect(resp.Status).To(Equal(responses.StatusFailed))

			// Missing rating
			resp = doReq("setRating", "id", songID)
			Expect(resp.Status).To(Equal(responses.StatusFailed))
		})
	})

	Describe("Scrobble", func() {
		It("submits a scrobble for a song", func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())

			resp := doReq("scrobble", "id", songs[0].ID, "submission", "true")

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("returns error when id is missing", func() {
			resp := doReq("scrobble")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})
})
