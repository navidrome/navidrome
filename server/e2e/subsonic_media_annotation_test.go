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
			r := newReq("star", "id", songID)
			resp, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("starred song appears in getStarred response", func() {
			r := newReq("getStarred")
			resp, err := router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Starred).ToNot(BeNil())
			Expect(resp.Starred.Song).To(HaveLen(1))
			Expect(resp.Starred.Song[0].Id).To(Equal(songID))
		})

		It("unstars a previously starred song", func() {
			r := newReq("unstar", "id", songID)
			resp, err := router.Unstar(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify song no longer appears in starred
			r = newReq("getStarred")
			resp, err = router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Starred.Song).To(BeEmpty())
		})

		It("stars an album by albumId", func() {
			r := newReq("star", "albumId", albumID)
			resp, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify album appears in starred
			r = newReq("getStarred")
			resp, err = router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Starred.Album).To(HaveLen(1))
			Expect(resp.Starred.Album[0].Id).To(Equal(albumID))
		})

		It("stars an artist by artistId", func() {
			r := newReq("star", "artistId", artistID)
			resp, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify artist appears in starred
			r = newReq("getStarred")
			resp, err = router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Starred.Artist).To(HaveLen(1))
			Expect(resp.Starred.Artist[0].Id).To(Equal(artistID))
		})

		It("returns error when no id provided", func() {
			r := newReq("star")
			_, err := router.Star(r)

			Expect(err).To(HaveOccurred())
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
			r := newReq("setRating", "id", songID, "rating", "4")
			resp, err := router.SetRating(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("rated song has correct userRating in getSong", func() {
			r := newReq("getSong", "id", songID)
			resp, err := router.GetSong(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Song).ToNot(BeNil())
			Expect(resp.Song.UserRating).To(Equal(int32(4)))
		})

		It("sets rating on an album", func() {
			r := newReq("setRating", "id", albumID, "rating", "3")
			resp, err := router.SetRating(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("returns error for missing parameters", func() {
			// Missing both id and rating
			r := newReq("setRating")
			_, err := router.SetRating(r)
			Expect(err).To(HaveOccurred())

			// Missing rating
			r = newReq("setRating", "id", songID)
			_, err = router.SetRating(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Scrobble", func() {
		It("submits a scrobble for a song", func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())

			r := newReq("scrobble", "id", songs[0].ID, "submission", "true")
			resp, err := router.Scrobble(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("returns error when id is missing", func() {
			r := newReq("scrobble")
			_, err := router.Scrobble(r)

			Expect(err).To(HaveOccurred())
		})
	})
})
